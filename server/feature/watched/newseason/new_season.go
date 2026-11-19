package newseason

import (
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"gorm.io/gorm"
)

// Activity types whose data can record the show itself being set to a
// status (as opposed to SEASON_*/EPISODE_* activities, whose data also has a
// "status" key but refers to a single season/episode).
var showStatusActivityTypes = []entity.ActivityType{
	entity.ADDED_WATCHED,
	entity.STATUS_CHANGED,
	entity.STATUS_CHANGED_AUTO,
	entity.IMPORTED_WATCHED,
	entity.IMPORTED_WATCHED_JF,
	entity.IMPORTED_WATCHED_PLEX,
}

// FinishedAtByWatched returns, per watched id, when the show was last set to
// FINISHED (the newest show-level activity recording status FINISHED). Ids
// without such an activity are missing from the map - callers fall back to
// the watched entry's own UpdatedAt for those.
//
// This is the "the user considered the show done as of then" moment that
// NewSeasonsSince compares season air dates against: a season that first
// aired before it was already out when the user said "finished", so it is
// not new to them (they either watched it or deliberately ignored it).
func FinishedAtByWatched(db *gorm.DB, watchedIds []uint) (map[uint]time.Time, error) {
	out := make(map[uint]time.Time, len(watchedIds))
	if len(watchedIds) == 0 {
		return out, nil
	}
	// Plain rows (not MAX()) so the sqlite driver parses created_at into a
	// time.Time - aggregate results come back as raw strings.
	var rows []struct {
		WatchedID uint
		CreatedAt time.Time
		Data      string
	}
	if res := db.Model(&entity.Activity{}).
		Select("watched_id", "created_at", "data").
		Where("watched_id IN ? AND type IN ?", watchedIds, showStatusActivityTypes).
		Find(&rows); res.Error != nil {
		return nil, res.Error
	}
	for _, r := range rows {
		if !strings.Contains(r.Data, `"status":"FINISHED"`) {
			continue
		}
		if cur, ok := out[r.WatchedID]; !ok || r.CreatedAt.After(cur) {
			out[r.WatchedID] = r.CreatedAt
		}
	}
	return out, nil
}

// NewSeason is a season of a show that started airing (or is scheduled to)
// after the user marked the show FINISHED.
type NewSeason struct {
	SeasonNumber int
	// First air date of the season (TMDB "air_date").
	AirDate time.Time
	// True if AirDate is today or earlier.
	Aired bool
}

// NewSeasonsSince returns the seasons of a show that are new relative to
// `finishedAt`, ascending by season number. A season counts as new when:
//   - it is a real season (not specials) that TMDB lists episodes for,
//   - TMDB has a first air date for it, and that date is after finishedAt,
//   - the user doesn't already have it FINISHED or DROPPED.
//
// Comparing against the finish date (instead of "any season without a
// watched entry") matters because shows can be marked FINISHED as a whole
// without any per-season entries - with the old check every such show
// looked like it had new seasons.
func NewSeasonsSince(
	showDetails tmdb.ShowDetails,
	watchedSeasons []entity.WatchedSeason,
	finishedAt time.Time,
	now time.Time,
) []NewSeason {
	done := make(map[int]bool, len(watchedSeasons))
	for _, ws := range watchedSeasons {
		if ws.Status == entity.FINISHED || ws.Status == entity.DROPPED {
			done[ws.SeasonNumber] = true
		}
	}
	// Compare whole days: TMDB air dates have no time, and the finish
	// timestamp is somewhere in that day.
	finishedDay := truncateDay(finishedAt)
	today := truncateDay(now)
	out := []NewSeason{}
	for _, se := range showDetails.Seasons {
		if se.SeasonNumber <= 0 || se.EpisodeCount <= 0 || done[se.SeasonNumber] {
			continue
		}
		airDate, err := time.Parse("2006-01-02", se.AirDate)
		if err != nil {
			continue // no (valid) date yet - can't tell when it's coming
		}
		if !airDate.After(finishedDay) {
			continue
		}
		out = append(out, NewSeason{
			SeasonNumber: se.SeasonNumber,
			AirDate:      airDate,
			Aired:        !airDate.After(today),
		})
	}
	return out
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
