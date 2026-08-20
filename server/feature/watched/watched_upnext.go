package watched

import (
	"log/slog"
	"sort"
	"strconv"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/watched/watchedutil"
)

// UpNextItem describes the next unwatched episode of an in-progress show,
// for the "Up Next" row on the overview page.
type UpNextItem struct {
	WatchedID     uint   `json:"watchedId"`
	TmdbID        int    `json:"tmdbId"`
	ShowTitle     string `json:"showTitle"`
	PosterPath    string `json:"posterPath"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	// Total number of episodes in that season (for a "S6.E2/12" style display).
	SeasonEpisodeCount int    `json:"seasonEpisodeCount"`
	EpisodeName        string `json:"episodeName"`
	StillPath          string `json:"stillPath"`
	AirDate            string `json:"airDate"`
	// Same fields as WatchedDto, so the card can use the exact same
	// PosterEpisodeBadge/PosterProgressBar/PosterRating/PosterStatus
	// components as the rest of the app.
	RemainingEpisodes int     `json:"remainingEpisodes,omitempty"`
	WatchProgress     int     `json:"watchProgress,omitempty"`
	Rating            float64 `json:"rating"`
}

// UpNext returns, for each show the user is currently WATCHING, the next
// unwatched episode. Shows that are fully watched (or whose next episode
// couldn't be determined) are omitted.
//
// Note: we deliberately do NOT filter on TMDB air_date. TMDB's air_date is
// the TV broadcast schedule, which doesn't reflect streaming/replay
// availability (a fully-streamable show can still have "future" broadcast
// dates). We offer the next episode that exists and hasn't been watched;
// the air date is still returned so the client can flag it.
func (s *Service) UpNext(userId uint, wpr domain.WatchedGetPageRequest) ([]UpNextItem, error) {
	var w []entity.Watched
	res := s.db.
		Joins("Content").
		// Game is joined only so the shared sort scope (which references Game
		// columns for some sorts) resolves; UpNext is shows-only so it's null.
		Joins("Game").
		Preload("WatchedEpisodes").
		Where("watcheds.user_id = ? AND Content.type = ? AND watcheds.status = ?",
			userId, entity.SHOW, entity.WATCHING).
		// Apply the same sort as the watched list so Up Next matches its order.
		Scopes(watchedRefineSort(wpr, userId)).
		Find(&w)
	if res.Error != nil {
		slog.Error("UpNext: query failed", "error", res.Error)
		return nil, res.Error
	}

	items := []UpNextItem{}
	for i := range w {
		if item, ok := s.nextEpisodeFor(&w[i]); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

// nextEpisodeFor computes the next unwatched episode for one show. It scans
// seasons starting from the highest season the user has already touched
// (episodes are usually watched contiguously), fetching season details from
// TMDB (cached) only as needed.
func (s *Service) nextEpisodeFor(w *entity.Watched) (UpNextItem, bool) {
	if w.Content == nil {
		return UpNextItem{}, false
	}
	// Set of already-watched (season, episode).
	seen := make(map[[2]int]bool, len(w.WatchedEpisodes))
	maxSeason := 1
	for _, we := range w.WatchedEpisodes {
		seen[[2]int{we.SeasonNumber, we.EpisodeNumber}] = true
		if we.SeasonNumber > maxSeason {
			maxSeason = we.SeasonNumber
		}
	}

	totalSeasons := int(w.Content.NumberOfSeasons)
	if totalSeasons < maxSeason {
		totalSeasons = maxSeason
	}
	tvId := strconv.Itoa(w.Content.TmdbID)
	remaining, progress := watchedutil.GetWatchProgress(w.Content.NumberOfEpisodes, w.WatchedEpisodes)

	for season := maxSeason; season <= totalSeasons; season++ {
		if season <= 0 { // skip specials (season 0)
			continue
		}
		sd, err := s.tmdb.SeasonDetails(tvId, strconv.Itoa(season))
		if err != nil {
			slog.Warn("nextEpisodeFor: season details failed", "tmdbId", w.Content.TmdbID, "season", season, "error", err)
			continue
		}
		eps := sd.Episodes
		sort.Slice(eps, func(a, b int) bool { return eps[a].EpisodeNumber < eps[b].EpisodeNumber })
		for _, ep := range eps {
			if seen[[2]int{ep.SeasonNumber, ep.EpisodeNumber}] {
				continue
			}
			// `remaining` (from GetWatchProgress) counts every not-yet-watched
			// episode, including this next one itself. Here we're already
			// displaying that next episode explicitly, so subtract it out -
			// the badge should read "how many more after this one", not
			// double-count the one being shown (unlike the Watching row's
			// badge, which shows the *last watched* episode, where the
			// unadjusted count is correct).
			remainingAfterThis := remaining - 1
			if remainingAfterThis < 0 {
				remainingAfterThis = 0
			}
			return UpNextItem{
				WatchedID:          w.ID,
				TmdbID:             w.Content.TmdbID,
				ShowTitle:          w.Content.Title,
				PosterPath:         w.Content.PosterPath,
				SeasonNumber:       ep.SeasonNumber,
				EpisodeNumber:      ep.EpisodeNumber,
				SeasonEpisodeCount: len(eps),
				EpisodeName:        ep.Name,
				StillPath:          ep.StillPath,
				AirDate:            ep.AirDate,
				RemainingEpisodes:  remainingAfterThis,
				WatchProgress:      progress,
				Rating:             w.Rating,
			}, true
		}
	}
	return UpNextItem{}, false
}
