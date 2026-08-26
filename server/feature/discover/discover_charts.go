package discover

import (
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"gorm.io/gorm"
)

// ChartItem is one entry of a provider's Top 10 chart.
type ChartItem struct {
	domain.Media
	// "up", "down", "same", or "" if there's no ~7-day-old snapshot to
	// compare against yet (eg this provider was only just selected).
	Movement string `json:"movement,omitempty"`
}

// ProviderChart is the ranked Top 10 of movies+shows currently popular on
// one streaming provider.
type ProviderChart struct {
	ProviderID int         `json:"providerId"`
	Items      []ChartItem `json:"items"`
}

const chartSize = 10

// Charts returns a Top 10 chart for each given provider id, movies and
// shows merged and ranked by TMDB popularity, with a movement indicator
// comparing today's rank against ~7 days ago. Every provider requested here
// also gets tracked (its chart snapshotted today) so movement becomes
// available on future visits, and RefreshProviderCharts (a daily recurring
// task) keeps that history going even on days the page isn't visited.
//
// TMDB has no single "trending for this provider" endpoint (unlike
// /trending, /discover has no per-provider chart concept either) - this is
// built from /discover/movie and /discover/tv (TMDB's default sort is
// already popularity.desc) filtered to that one provider, then merged and
// re-sorted by popularity ourselves, since the two lists were only sorted
// within themselves.
func (s *Service) Charts(providerIds []string, region string) ([]ProviderChart, error) {
	charts := make([]ProviderChart, 0, len(providerIds))
	for _, pid := range providerIds {
		id, err := strconv.Atoi(pid)
		if err != nil {
			slog.Warn("Charts: invalid provider id, skipping", "provider", pid)
			continue
		}
		items, err := chartForProvider(s.tmdb, pid, region)
		if err != nil {
			slog.Warn("Charts: chartForProvider failed, skipping", "provider", pid, "error", err)
			continue
		}
		if err := saveChartSnapshot(s.db, id, items); err != nil {
			slog.Warn("Charts: failed to save snapshot", "provider", id, "error", err)
		}
		chartItems := make([]ChartItem, len(items))
		for i, m := range items {
			chartItems[i] = ChartItem{
				Media:    m,
				Movement: movementFor(s.db, id, m.IDs.TMDB, mediaContentType(m), i+1),
			}
		}
		charts = append(charts, ProviderChart{ProviderID: id, Items: chartItems})
	}
	if len(charts) == 0 && len(providerIds) > 0 {
		return nil, errors.New("failed to build any charts")
	}
	return charts, nil
}

func mediaContentType(m domain.Media) entity.ContentType {
	if m.Type == domain.MediaTypeTMDBShow {
		return entity.SHOW
	}
	return entity.MOVIE
}

func chartForProvider(tmdbSvc *tmdb.TMDB, providerId string, region string) ([]domain.Media, error) {
	type ranked struct {
		media      domain.Media
		popularity float64
	}
	var all []ranked

	movies, err := tmdbSvc.DiscoverMovies(
		tmdb.DiscoverOptions{WithWatchProviders: providerId},
		1,
		region,
	)
	if err != nil {
		slog.Error("chartForProvider: movies request failed", "provider", providerId, "error", err)
	} else {
		for _, v := range movies.Results {
			all = append(all, ranked{media: v.AsMedia(), popularity: v.Popularity})
		}
	}

	shows, err := tmdbSvc.DiscoverShows(
		tmdb.DiscoverOptions{WithWatchProviders: providerId},
		1,
		region,
	)
	if err != nil {
		slog.Error("chartForProvider: shows request failed", "provider", providerId, "error", err)
	} else {
		for _, v := range shows.Results {
			all = append(all, ranked{media: v.AsMedia(), popularity: v.Popularity})
		}
	}

	if len(all) == 0 {
		return nil, errors.New("no results")
	}

	sort.Slice(all, func(a, b int) bool { return all[a].popularity > all[b].popularity })
	if len(all) > chartSize {
		all = all[:chartSize]
	}

	items := make([]domain.Media, len(all))
	for i, r := range all {
		items[i] = r.media
	}
	return items, nil
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// saveChartSnapshot replaces today's stored snapshot rows for one provider
// with the given chart. Idempotent - safe to call more than once a day.
func saveChartSnapshot(db *gorm.DB, providerId int, items []domain.Media) error {
	today := truncateToDay(time.Now())
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("provider_id = ? AND snapshot_date = ?", providerId, today).
			Delete(&entity.ProviderChartSnapshot{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		rows := make([]entity.ProviderChartSnapshot, len(items))
		for i, m := range items {
			rows[i] = entity.ProviderChartSnapshot{
				ProviderID:   providerId,
				SnapshotDate: today,
				TmdbID:       m.IDs.TMDB,
				ContentType:  mediaContentType(m),
				Rank:         i + 1,
			}
		}
		return tx.Create(&rows).Error
	})
}

// movementFor compares today's rank against the nearest snapshot 7-10 days
// ago (a small window rather than exactly 7 days, in case a day's snapshot
// was missed) for the same item on the same provider.
func movementFor(db *gorm.DB, providerId int, tmdbId int, contentType entity.ContentType, todayRank int) string {
	var prev entity.ProviderChartSnapshot
	target := truncateToDay(time.Now().AddDate(0, 0, -7))
	cutoff := truncateToDay(time.Now().AddDate(0, 0, -10))
	res := db.
		Where(
			"provider_id = ? AND tmdb_id = ? AND content_type = ? AND snapshot_date <= ? AND snapshot_date >= ?",
			providerId, tmdbId, contentType, target, cutoff,
		).
		Order("snapshot_date DESC").
		First(&prev)
	if res.Error != nil {
		return ""
	}
	if prev.Rank < todayRank {
		return "down"
	} else if prev.Rank > todayRank {
		return "up"
	}
	return "same"
}

// RefreshProviderCharts is a recurring task: re-snapshot today's chart for
// every provider that's ever been requested via the Charts page, so
// tracking history continues even on days nobody visits it (which would
// otherwise leave gaps that break the ~7-day-ago movement lookup).
func RefreshProviderCharts(db *gorm.DB, tmdbSvc *tmdb.TMDB, region string) {
	var providerIds []int
	if res := db.Model(&entity.ProviderChartSnapshot{}).
		Distinct("provider_id").
		Pluck("provider_id", &providerIds); res.Error != nil {
		slog.Error("RefreshProviderCharts: Failed to list tracked providers!", "error", res.Error)
		return
	}
	for _, id := range providerIds {
		items, err := chartForProvider(tmdbSvc, strconv.Itoa(id), region)
		if err != nil {
			slog.Warn("RefreshProviderCharts: chartForProvider failed, skipping", "provider", id, "error", err)
			continue
		}
		if err := saveChartSnapshot(db, id, items); err != nil {
			slog.Error("RefreshProviderCharts: Failed to save snapshot!", "provider", id, "error", err)
		}
	}
}
