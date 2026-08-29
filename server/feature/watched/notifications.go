package watched

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/notify"
	"gorm.io/gorm"
)

// SendReleaseNotifications is a recurring task: for every PLANNED movie/show
// releasing today, notify its owner via ntfy (if they've configured a topic
// URL and have release notifications enabled). Each (user, watched item) is
// only ever notified once - see entity.SentNotification.
func SendReleaseNotifications(db *gorm.DB) {
	dayStart := time.Now().UTC().Truncate(24 * time.Hour)
	dayEnd := dayStart.AddDate(0, 0, 1)

	var w []entity.Watched
	if res := db.
		Joins("Content").
		Where(
			"watcheds.status = ? AND Content.release_date >= ? AND Content.release_date < ?",
			entity.PLANNED, dayStart, dayEnd,
		).
		Find(&w); res.Error != nil {
		slog.Error("SendReleaseNotifications: Failed to query releases!", "error", res.Error)
		return
	}
	if len(w) == 0 {
		return
	}

	users := usersById(db, watchedUserIds(w))
	sent := 0
	for _, item := range w {
		if item.Content == nil {
			continue
		}
		u, ok := users[item.UserID]
		if !ok || u.NtfyUrl == nil || *u.NtfyUrl == "" {
			continue
		}
		if u.NotifyReleases != nil && !*u.NotifyReleases {
			continue
		}
		if notify.AlreadyNotified(db, item.UserID, item.ID, entity.NotificationTypeRelease) {
			continue
		}
		kind := "Show"
		if item.Content.Type == entity.MOVIE {
			kind = "Movie"
		}
		msg := fmt.Sprintf("🎬 %s out today: %s", kind, item.Content.Title)
		if err := notify.SendNtfy(*u.NtfyUrl, msg); err != nil {
			slog.Error("SendReleaseNotifications: Failed to send notification!",
				"user_id", item.UserID, "watched_id", item.ID, "error", err)
			continue
		}
		notify.MarkNotified(db, item.UserID, item.ID, entity.NotificationTypeRelease)
		sent++
	}
	slog.Debug("SendReleaseNotifications: Done.", "candidates", len(w), "sent", sent)
}

func watchedUserIds(w []entity.Watched) []uint {
	ids := make([]uint, 0, len(w))
	seen := make(map[uint]bool, len(w))
	for _, item := range w {
		if !seen[item.UserID] {
			seen[item.UserID] = true
			ids = append(ids, item.UserID)
		}
	}
	return ids
}

func usersById(db *gorm.DB, ids []uint) map[uint]entity.User {
	var users []entity.User
	m := make(map[uint]entity.User, len(ids))
	if len(ids) == 0 {
		return m
	}
	if res := db.Select("id", "ntfy_url", "notify_releases", "notify_new_seasons").
		Where("id IN ?", ids).Find(&users); res.Error != nil {
		slog.Error("usersById: Failed to query users!", "error", res.Error)
		return m
	}
	for _, u := range users {
		m[u.ID] = u
	}
	return m
}
