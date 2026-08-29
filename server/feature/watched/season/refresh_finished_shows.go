package season

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/notify"
	"gorm.io/gorm"
)

// RefreshFinishedShowsForNewSeasons is a recurring task: for every show
// marked FINISHED (across all users with AutomateShowStatuses on), check
// TMDB for a season the user has no watched entry for at all. If one
// exists, the show has continued since being finished, so move it back to
// PLANNED - counterpart to the season->episode/show FINISHED cascade in
// season.go, for when a show picks back up after being "done".
func RefreshFinishedShowsForNewSeasons(
	db *gorm.DB,
	tmdbSvc *tmdb.TMDB,
	activityProvider domain.ActivityAddProvider,
) {
	var shows []entity.Watched
	if res := db.
		Joins("Content").
		Where("status = ? AND Content.type = ?", entity.FINISHED, entity.SHOW).
		Preload("WatchedSeasons").
		Find(&shows); res.Error != nil {
		slog.Error("RefreshFinishedShowsForNewSeasons: Failed to query finished shows!", "error", res.Error)
		return
	}
	if len(shows) == 0 {
		return
	}

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
		slog.Error("RefreshFinishedShowsForNewSeasons: Failed to query users!", "error", res.Error)
		return
	}
	automateByUser := make(map[uint]bool, len(users))
	usersById := make(map[uint]entity.User, len(users))
	for _, u := range users {
		automateByUser[u.ID] = u.AutomateShowStatuses == nil || *u.AutomateShowStatuses
		usersById[u.ID] = u
	}

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
		known := make(map[int]bool, len(w.WatchedSeasons))
		for _, ws := range w.WatchedSeasons {
			known[ws.SeasonNumber] = true
		}
		hasNewSeason := false
		for _, se := range showDetails.Seasons {
			// Skip specials (season 0) and seasons TMDB has no episodes
			// for yet (announced but not released).
			if se.SeasonNumber <= 0 || se.EpisodeCount <= 0 {
				continue
			}
			if !known[se.SeasonNumber] {
				hasNewSeason = true
				break
			}
		}
		if !hasNewSeason {
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
			"reason": "A new season became available.",
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
			msg := fmt.Sprintf("🆕 New season available: %s", w.Content.Title)
			if err := notify.SendNtfy(*u.NtfyUrl, msg); err != nil {
				slog.Error("RefreshFinishedShowsForNewSeasons: Failed to send notification!",
					"watched_id", w.ID, "error", err)
			} else {
				notify.MarkNotified(db, w.UserID, w.ID, entity.NotificationTypeNewSeason)
			}
		}
	}
	slog.Debug("RefreshFinishedShowsForNewSeasons: Done.", "checked", checked, "updated", updated)
}
