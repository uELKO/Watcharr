package season

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/watched/newseason"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/notify"
	"gorm.io/gorm"
)

// RefreshFinishedShowsForNewSeasons is a recurring task (also ran once at
// startup), for every user with AutomateShowStatuses on:
//
//  1. BackfillFinishedSeasonEpisodes - data repair so every FINISHED season
//     also has all its episodes FINISHED.
//  2. For every show marked FINISHED, check TMDB for a season that started
//     airing after the show was marked FINISHED (see
//     newseason.NewSeasonsSince). If one has aired, the show has continued
//     since being finished, so move it back to PLANNED (and notify) -
//     counterpart to the season->episode/show FINISHED cascade in
//     season.go, for when a show picks back up after being "done".
//     Seasons that are announced but haven't aired yet don't flip the show;
//     those show up as an "upcoming season" card in Up Next instead.
func RefreshFinishedShowsForNewSeasons(
	db *gorm.DB,
	tmdbSvc *tmdb.TMDB,
	activityProvider domain.ActivityAddProvider,
) {
	BackfillFinishedSeasonEpisodes(db, tmdbSvc)

	var shows []entity.Watched
	if res := db.
		Joins("Content").
		Where("watcheds.status = ? AND Content.type = ?", entity.FINISHED, entity.SHOW).
		Preload("WatchedSeasons").
		Find(&shows); res.Error != nil {
		slog.Error("RefreshFinishedShowsForNewSeasons: Failed to query finished shows!", "error", res.Error)
		return
	}
	if len(shows) == 0 {
		return
	}

	usersById, automateByUser, err := automationUsers(db, shows)
	if err != nil {
		slog.Error("RefreshFinishedShowsForNewSeasons: Failed to query users!", "error", err)
		return
	}

	watchedIds := make([]uint, 0, len(shows))
	for _, w := range shows {
		watchedIds = append(watchedIds, w.ID)
	}
	finishedAt, err := newseason.FinishedAtByWatched(db, watchedIds)
	if err != nil {
		slog.Error("RefreshFinishedShowsForNewSeasons: Failed to query finish dates!", "error", err)
		return
	}

	now := time.Now()
	checked := 0
	updated := 0
	for _, w := range shows {
		if !automateByUser[w.UserID] || w.Content == nil {
			continue
		}
		checked++
		showDetails, err := tmdbSvc.ShowDetails(tmdb.ShowDetailsOptions{ID: strconv.Itoa(w.Content.TmdbID)})
		if err != nil {
			slog.Error("RefreshFinishedShowsForNewSeasons: Failed to get show details!",
				"watched_id", w.ID, "error", err)
			continue
		}
		fa, ok := finishedAt[w.ID]
		if !ok {
			fa = w.UpdatedAt
		}
		var aired *newseason.NewSeason
		for _, ns := range newseason.NewSeasonsSince(showDetails, w.WatchedSeasons, fa, now) {
			if ns.Aired {
				aired = &ns
				break
			}
		}
		if aired == nil {
			continue
		}
		if res := db.Model(&entity.Watched{}).Where("id = ?", w.ID).Update("status", entity.PLANNED); res.Error != nil {
			slog.Error("RefreshFinishedShowsForNewSeasons: Failed to update show status!",
				"watched_id", w.ID, "error", res.Error)
			continue
		}
		updated++
		actData, _ := json.Marshal(map[string]any{
			"status": entity.PLANNED,
			"reason": fmt.Sprintf("Season %d became available.", aired.SeasonNumber),
		})
		activityProvider.AddActivity(w.UserID, domain.ActivityAddProps{
			WatchedID: w.ID,
			Type:      entity.STATUS_CHANGED_AUTO,
			Data:      string(actData),
		}, false)

		if u, ok := usersById[w.UserID]; ok &&
			u.NtfyUrl != nil && *u.NtfyUrl != "" &&
			(u.NotifyNewSeasons == nil || *u.NotifyNewSeasons) &&
			!notify.AlreadyNotified(db, w.UserID, w.ID, entity.NotificationTypeNewSeason) {
			msg := fmt.Sprintf("🆕 New season available: %s (Season %d)", w.Content.Title, aired.SeasonNumber)
			if err := notify.SendNtfy(*u.NtfyUrl, msg); err != nil {
				slog.Error("RefreshFinishedShowsForNewSeasons: Failed to send notification!",
					"watched_id", w.ID, "error", err)
			} else {
				notify.MarkNotified(db, w.UserID, w.ID, entity.NotificationTypeNewSeason)
			}
		}
	}
	slog.Info("RefreshFinishedShowsForNewSeasons: Done.", "checked", checked, "updated", updated)
}

// Loads the users owning `shows`, returning them by id plus whether each has
// AutomateShowStatuses enabled (nil = default = on).
func automationUsers(db *gorm.DB, shows []entity.Watched) (map[uint]entity.User, map[uint]bool, error) {
	userIds := make([]uint, 0, len(shows))
	seenUsers := make(map[uint]bool, len(shows))
	for _, w := range shows {
		if !seenUsers[w.UserID] {
			seenUsers[w.UserID] = true
			userIds = append(userIds, w.UserID)
		}
	}
	var users []entity.User
	if res := db.Select("id", "automate_show_statuses", "ntfy_url", "notify_new_seasons").
		Where("id IN ?", userIds).Find(&users); res.Error != nil {
		return nil, nil, res.Error
	}
	automateByUser := make(map[uint]bool, len(users))
	usersById := make(map[uint]entity.User, len(users))
	for _, u := range users {
		automateByUser[u.ID] = u.AutomateShowStatuses == nil || *u.AutomateShowStatuses
		usersById[u.ID] = u
	}
	return usersById, automateByUser, nil
}

// BackfillFinishedSeasonEpisodes makes sure every FINISHED season also has
// all of its episodes FINISHED: episodes without an entry are created, and
// entries that aren't completed yet (PLANNED/WATCHING/HOLD/no status) are
// upgraded. DROPPED episodes are left alone (deliberate choice by the user).
//
// The season->episode cascade in hookSeasonStatusChanged does this live
// whenever a season gets marked FINISHED, but seasons finished before that
// cascade existed (eg from an import) never got their episodes filled. This
// is idempotent and only touches seasons that are actually incomplete, so
// it's safe to run on every startup / task run.
//
// Pure data repair: no activities are added (it would otherwise flood the
// activity log with hundreds of backdated "episode watched" entries).
func BackfillFinishedSeasonEpisodes(db *gorm.DB, tmdbSvc *tmdb.TMDB) {
	var seasons []entity.WatchedSeason
	if res := db.Where("status = ?", entity.FINISHED).Find(&seasons); res.Error != nil {
		slog.Error("BackfillFinishedSeasonEpisodes: Failed to query finished seasons!", "error", res.Error)
		return
	}
	if len(seasons) == 0 {
		return
	}

	// Respect AutomateShowStatuses like the live cascade does.
	userIds := []uint{}
	seenUser := map[uint]bool{}
	for _, s := range seasons {
		if !seenUser[s.UserID] {
			seenUser[s.UserID] = true
			userIds = append(userIds, s.UserID)
		}
	}
	var users []entity.User
	if res := db.Select("id", "automate_show_statuses").Where("id IN ?", userIds).Find(&users); res.Error != nil {
		slog.Error("BackfillFinishedSeasonEpisodes: Failed to query users!", "error", res.Error)
		return
	}
	automate := map[uint]bool{}
	for _, u := range users {
		automate[u.ID] = u.AutomateShowStatuses == nil || *u.AutomateShowStatuses
	}

	// TMDB id per watched entry.
	watchedIds := []uint{}
	seenWatched := map[uint]bool{}
	for _, s := range seasons {
		if !seenWatched[s.WatchedID] {
			seenWatched[s.WatchedID] = true
			watchedIds = append(watchedIds, s.WatchedID)
		}
	}
	var watcheds []entity.Watched
	if res := db.Joins("Content").Where("watcheds.id IN ?", watchedIds).Find(&watcheds); res.Error != nil {
		slog.Error("BackfillFinishedSeasonEpisodes: Failed to query watched items!", "error", res.Error)
		return
	}
	tmdbIdByWatched := map[uint]int{}
	for _, w := range watcheds {
		if w.Content != nil && w.Content.Type == entity.SHOW {
			tmdbIdByWatched[w.ID] = w.Content.TmdbID
		}
	}

	seasonsFixed := 0
	episodesCreated := 0
	episodesUpgraded := 0
	for _, s := range seasons {
		tmdbId, ok := tmdbIdByWatched[s.WatchedID]
		if !ok || !automate[s.UserID] || s.SeasonNumber <= 0 {
			continue
		}
		var existing []entity.WatchedEpisode
		if res := db.Where("watched_id = ? AND season_number = ?", s.WatchedID, s.SeasonNumber).
			Find(&existing); res.Error != nil {
			slog.Error("BackfillFinishedSeasonEpisodes: Failed to query episodes!",
				"watched_id", s.WatchedID, "season", s.SeasonNumber, "error", res.Error)
			continue
		}
		byEp := make(map[int]entity.WatchedEpisode, len(existing))
		for _, e := range existing {
			byEp[e.EpisodeNumber] = e
		}
		sd, err := tmdbSvc.SeasonDetails(strconv.Itoa(tmdbId), strconv.Itoa(s.SeasonNumber))
		if err != nil {
			slog.Error("BackfillFinishedSeasonEpisodes: Failed to get season details!",
				"watched_id", s.WatchedID, "season", s.SeasonNumber, "error", err)
			continue
		}
		fixedHere := false
		for _, ep := range sd.Episodes {
			e, ok := byEp[ep.EpisodeNumber]
			if !ok {
				if res := db.Create(&entity.WatchedEpisode{
					UserID:        s.UserID,
					WatchedID:     s.WatchedID,
					SeasonNumber:  s.SeasonNumber,
					EpisodeNumber: ep.EpisodeNumber,
					Status:        entity.FINISHED,
				}); res.Error != nil {
					slog.Error("BackfillFinishedSeasonEpisodes: Failed to create episode!",
						"watched_id", s.WatchedID, "season", s.SeasonNumber, "episode", ep.EpisodeNumber, "error", res.Error)
					continue
				}
				episodesCreated++
				fixedHere = true
				continue
			}
			if e.Status == entity.FINISHED || e.Status == entity.DROPPED {
				continue
			}
			if res := db.Model(&entity.WatchedEpisode{}).Where("id = ?", e.ID).
				Update("status", entity.FINISHED); res.Error != nil {
				slog.Error("BackfillFinishedSeasonEpisodes: Failed to upgrade episode!",
					"watched_episode_id", e.ID, "error", res.Error)
				continue
			}
			episodesUpgraded++
			fixedHere = true
		}
		if fixedHere {
			seasonsFixed++
		}
	}
	slog.Info("BackfillFinishedSeasonEpisodes: Done.",
		"seasons_checked", len(seasons), "seasons_fixed", seasonsFixed,
		"episodes_created", episodesCreated, "episodes_upgraded", episodesUpgraded)
}
