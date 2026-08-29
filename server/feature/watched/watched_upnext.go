package watched

import (
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/watched/watchedutil"
)

// UpNextItemKind distinguishes the three kinds of card the "Up Next" row can
// show: the next unwatched episode of a show being WATCHING, a show that
// was previously watched (has watched history) and has a new, unwatched
// season now that a PLANNED->WATCHING automation moved it back to PLANNED
// for, or a PLANNED movie/show with a known, still-upcoming release date.
type UpNextItemKind string

const (
	UpNextKindEpisode   UpNextItemKind = "episode"
	UpNextKindNewSeason UpNextItemKind = "newseason"
	UpNextKindRelease   UpNextItemKind = "release"
)

// UpNextItem describes one card in the "Up Next" row on the overview page -
// the next unwatched episode of an in-progress show (Kind ==
// UpNextKindEpisode), the next unwatched episode of a show that got a new
// season since being finished (Kind == UpNextKindNewSeason), or a PLANNED
// movie/show that hasn't released yet (Kind == UpNextKindRelease).
type UpNextItem struct {
	Kind      UpNextItemKind `json:"kind"`
	WatchedID uint           `json:"watchedId"`
	TmdbID    int            `json:"tmdbId"`
	// "movie" or "tv" - episode-kind items are always shows, but release-kind
	// items can be either, so the client needs this to link to the right page.
	ContentType string `json:"contentType"`
	ShowTitle   string `json:"showTitle"`
	PosterPath  string `json:"posterPath"`
	SeasonNumber  int `json:"seasonNumber,omitempty"`
	EpisodeNumber int `json:"episodeNumber,omitempty"`
	// Total number of episodes in that season (for a "S6.E2/12" style display).
	SeasonEpisodeCount int    `json:"seasonEpisodeCount,omitempty"`
	EpisodeName        string `json:"episodeName,omitempty"`
	StillPath          string `json:"stillPath,omitempty"`
	AirDate            string `json:"airDate,omitempty"`
	// Release-kind only: the movie/show's release (or first air) date.
	ReleaseDate string `json:"releaseDate,omitempty"`
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

	newSeasonItems, err := s.newSeasonEpisodes(userId, wpr)
	if err != nil {
		slog.Warn("UpNext: newSeasonEpisodes query failed", "error", err)
	} else {
		items = append(items, newSeasonItems...)
	}

	releases, err := s.upcomingPlannedReleases(userId)
	if err != nil {
		slog.Warn("UpNext: upcoming planned releases query failed", "error", err)
	} else {
		items = append(items, releases...)
	}

	return items, nil
}

// newSeasonEpisodes finds shows the user previously watched at least one
// episode of that are now PLANNED again - ie a new season arrived since
// they finished it (see season.RefreshFinishedShowsForNewSeasons, which
// moves a FINISHED show back to PLANNED when TMDB shows a season with no
// watched entry at all) - and returns the next unwatched episode for each,
// same as an actively WATCHING show, just flagged with UpNextKindNewSeason
// so the client can call it out distinctly. Without this, a finished show
// getting a new season is easy to miss - it's not WATCHING (so not in the
// row above) and its new season has already aired (so it's not an
// upcoming release either).
func (s *Service) newSeasonEpisodes(userId uint, wpr domain.WatchedGetPageRequest) ([]UpNextItem, error) {
	var w []entity.Watched
	res := s.db.
		Joins("Content").
		Joins("Game").
		Preload("WatchedEpisodes").
		Where("watcheds.user_id = ? AND Content.type = ? AND watcheds.status = ?",
			userId, entity.SHOW, entity.PLANNED).
		Scopes(watchedRefineSort(wpr, userId)).
		Find(&w)
	if res.Error != nil {
		return nil, res.Error
	}
	items := []UpNextItem{}
	for i := range w {
		if len(w[i].WatchedEpisodes) == 0 {
			// Never actually started - a plain new PLANNED add, not a
			// "resuming after a new season" case.
			continue
		}
		if item, ok := s.nextEpisodeFor(&w[i]); ok {
			item.Kind = UpNextKindNewSeason
			items = append(items, item)
		}
	}
	return items, nil
}

// upcomingPlannedReleases returns PLANNED movies/shows with a known release
// date that hasn't happened yet, soonest first - the "coming soon" half of
// Up Next, alongside the in-progress shows' next episodes above.
func (s *Service) upcomingPlannedReleases(userId uint) ([]UpNextItem, error) {
	var w []entity.Watched
	res := s.db.
		Joins("Content").
		Where(
			"watcheds.user_id = ? AND watcheds.status = ? AND Content.release_date IS NOT NULL AND Content.release_date > ?",
			userId, entity.PLANNED, time.Now(),
		).
		Order("Content.release_date ASC").
		Find(&w)
	if res.Error != nil {
		return nil, res.Error
	}
	items := make([]UpNextItem, 0, len(w))
	for i := range w {
		if w[i].Content == nil || w[i].Content.ReleaseDate == nil {
			continue
		}
		contentType := "tv"
		if w[i].Content.Type == entity.MOVIE {
			contentType = "movie"
		}
		items = append(items, UpNextItem{
			Kind:        UpNextKindRelease,
			WatchedID:   w[i].ID,
			TmdbID:      w[i].Content.TmdbID,
			ContentType: contentType,
			ShowTitle:   w[i].Content.Title,
			PosterPath:  w[i].Content.PosterPath,
			ReleaseDate: w[i].Content.ReleaseDate.Format("2006-01-02"),
			Rating:      w[i].Rating,
		})
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
				Kind:                UpNextKindEpisode,
				WatchedID:           w.ID,
				TmdbID:              w.Content.TmdbID,
				ContentType:         "tv",
				ShowTitle:           w.Content.Title,
				PosterPath:          w.Content.PosterPath,
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
