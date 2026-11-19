package watched

import (
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/watched/newseason"
	"github.com/sbondCo/Watcharr/feature/watched/watchedutil"
	"github.com/sbondCo/Watcharr/media/tmdb"
)

// UpNextItemKind distinguishes the kinds of card the "Up Next" row can
// show: the next unwatched episode of a show being WATCHING, a previously
// watched show with a new season that has already started airing, a
// previously finished show with a new season that is announced with a date
// but hasn't aired yet, or a PLANNED movie/show with a known, still-upcoming
// release date.
type UpNextItemKind string

const (
	UpNextKindEpisode        UpNextItemKind = "episode"
	UpNextKindNewSeason      UpNextItemKind = "newseason"
	UpNextKindUpcomingSeason UpNextItemKind = "upcomingseason"
	UpNextKindRelease        UpNextItemKind = "release"
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
	// The watched entry's own status, so the card's status button shows the
	// real one (eg FINISHED for an upcoming season of a finished show).
	ShowStatus entity.WatchedStatus `json:"showStatus"`
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
		Preload("WatchedSeasons").
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

	finishedItems, err := s.finishedShowsNewSeasons(userId)
	if err != nil {
		slog.Warn("UpNext: finishedShowsNewSeasons query failed", "error", err)
	} else {
		items = append(items, finishedItems...)
	}

	releases, err := s.upcomingPlannedReleases(userId)
	if err != nil {
		slog.Warn("UpNext: upcoming planned releases query failed", "error", err)
	} else {
		items = append(items, releases...)
	}

	// Merge all three kinds into one date-ascending row - each kind is
	// gathered (and sorted) independently above, so without this the row
	// would just show episodes, then new-seasons, then releases as three
	// separate blocks instead of one soonest-first list. Items with no
	// known date (AirDate/ReleaseDate both empty) sort last.
	sort.SliceStable(items, func(a, b int) bool {
		da, db := itemDate(items[a]), itemDate(items[b])
		if da == "" {
			return false
		}
		if db == "" {
			return true
		}
		return da < db
	})

	return items, nil
}

// itemDate returns the date an UpNextItem should be sorted by: the episode
// air date for episode/newseason cards, the release date for release cards.
// Both are formatted "2006-01-02" so plain string comparison sorts correctly.
func itemDate(item UpNextItem) string {
	if item.AirDate != "" {
		return item.AirDate
	}
	return item.ReleaseDate
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
		Preload("WatchedSeasons").
		Where("watcheds.user_id = ? AND Content.type = ? AND watcheds.status = ?",
			userId, entity.SHOW, entity.PLANNED).
		Scopes(watchedRefineSort(wpr, userId)).
		Find(&w)
	if res.Error != nil {
		return nil, res.Error
	}
	items := []UpNextItem{}
	for i := range w {
		if len(w[i].WatchedEpisodes) == 0 && !hasCompletedSeason(w[i].WatchedSeasons) {
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
			ShowStatus:  w[i].Status,
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

	// Seasons marked FINISHED/DROPPED as a whole count as fully seen, even
	// if (older data) their episodes have no entries of their own.
	seasonDone := make(map[int]bool, len(w.WatchedSeasons))
	for _, ws := range w.WatchedSeasons {
		if ws.Status == entity.FINISHED || ws.Status == entity.DROPPED {
			seasonDone[ws.SeasonNumber] = true
			if ws.SeasonNumber > maxSeason {
				maxSeason = ws.SeasonNumber
			}
		}
	}

	tvId := strconv.Itoa(w.Content.TmdbID)
	// Prefer TMDB's (cached) live season count: the stored Content row can
	// be stale and miss a season that was added since it was cached.
	totalSeasons := int(w.Content.NumberOfSeasons)
	if sd, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{ID: tvId}); err == nil && int(sd.NumberOfSeasons) > totalSeasons {
		totalSeasons = int(sd.NumberOfSeasons)
	}
	if totalSeasons < maxSeason {
		totalSeasons = maxSeason
	}
	remaining, progress := watchedutil.GetWatchProgress(w.Content.NumberOfEpisodes, w.WatchedEpisodes)

	for season := maxSeason; season <= totalSeasons; season++ {
		if season <= 0 || seasonDone[season] { // skip specials (season 0) and completed seasons
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
				ShowStatus:          w.Status,
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

// True if any season is marked FINISHED or DROPPED - ie the user has watch
// history for the show even without per-episode entries.
func hasCompletedSeason(ws []entity.WatchedSeason) bool {
	for _, s := range ws {
		if s.Status == entity.FINISHED || s.Status == entity.DROPPED {
			return true
		}
	}
	return false
}

// finishedShowsNewSeasons covers FINISHED shows that got a new season since
// the user marked them finished (see newseason.NewSeasonsSince). Works
// straight off TMDB, independent of the daily task that moves such shows
// back to PLANNED, and regardless of whether the user tracked individual
// seasons/episodes or just marked the whole show finished.
//
// Per show, the earliest new season produces one card for its first
// episode: Kind "newseason" if it has already started airing (the task
// hasn't flipped the show yet), "upcomingseason" if it is only announced
// with a date (eg 3 Body Problem S2) - the "coming soon" reminder for shows
// you've already finished.
//
// Only shows whose stored TMDB status says they can still continue (not
// Ended/Canceled) are checked, to keep the number of TMDB lookups (cached
// for 24h) small.
func (s *Service) finishedShowsNewSeasons(userId uint) ([]UpNextItem, error) {
	var w []entity.Watched
	res := s.db.
		Joins("Content").
		Preload("WatchedSeasons").
		Where("watcheds.user_id = ? AND Content.type = ? AND watcheds.status = ? AND Content.status NOT IN ?",
			userId, entity.SHOW, entity.FINISHED, []string{"Ended", "Canceled"}).
		Find(&w)
	if res.Error != nil {
		return nil, res.Error
	}
	if len(w) == 0 {
		return []UpNextItem{}, nil
	}
	ids := make([]uint, 0, len(w))
	for i := range w {
		ids = append(ids, w[i].ID)
	}
	finishedAt, err := newseason.FinishedAtByWatched(s.db, ids)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	results := make([]*UpNextItem, len(w))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6) // bound parallel TMDB lookups (first, uncached load)
	for i := range w {
		if w[i].Content == nil {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fa, ok := finishedAt[w[i].ID]
			if !ok {
				fa = w[i].UpdatedAt
			}
			results[i] = s.newSeasonCardFor(&w[i], fa, now)
		}(i)
	}
	wg.Wait()

	items := []UpNextItem{}
	for _, it := range results {
		if it != nil {
			items = append(items, *it)
		}
	}
	return items, nil
}

// Builds the card for the earliest season of `w` that is new since
// `finishedAt`, or nil if there is none.
func (s *Service) newSeasonCardFor(w *entity.Watched, finishedAt time.Time, now time.Time) *UpNextItem {
	tvId := strconv.Itoa(w.Content.TmdbID)
	showDetails, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{ID: tvId})
	if err != nil {
		slog.Warn("newSeasonCardFor: show details failed", "tmdbId", w.Content.TmdbID, "error", err)
		return nil
	}
	newSeasons := newseason.NewSeasonsSince(showDetails, w.WatchedSeasons, finishedAt, now)
	if len(newSeasons) == 0 {
		return nil
	}
	ns := newSeasons[0]
	kind := UpNextKindUpcomingSeason
	if ns.Aired {
		kind = UpNextKindNewSeason
	}
	item := &UpNextItem{
		Kind:         kind,
		WatchedID:    w.ID,
		TmdbID:       w.Content.TmdbID,
		ContentType:  "tv",
		ShowTitle:    w.Content.Title,
		ShowStatus:   w.Status,
		PosterPath:   w.Content.PosterPath,
		SeasonNumber: ns.SeasonNumber,
		// Fallbacks in case season details fail below.
		EpisodeNumber: 1,
		AirDate:       ns.AirDate.Format("2006-01-02"),
		Rating:        w.Rating,
	}
	sd, err := s.tmdb.SeasonDetails(tvId, strconv.Itoa(ns.SeasonNumber))
	if err != nil {
		slog.Warn("newSeasonCardFor: season details failed", "tmdbId", w.Content.TmdbID, "season", ns.SeasonNumber, "error", err)
		return item
	}
	eps := sd.Episodes
	sort.Slice(eps, func(a, b int) bool { return eps[a].EpisodeNumber < eps[b].EpisodeNumber })
	item.SeasonEpisodeCount = len(eps)
	if len(eps) > 0 {
		ep := eps[0]
		item.EpisodeNumber = ep.EpisodeNumber
		item.EpisodeName = ep.Name
		item.StillPath = ep.StillPath
		if ep.AirDate != "" {
			item.AirDate = ep.AirDate
		}
		// Everything in the new season is still ahead, incl. this episode;
		// count the ones after it, same as the other cards.
		item.RemainingEpisodes = len(eps) - 1
	}
	return item
}
