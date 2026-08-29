package notify

import (
	"log/slog"

	"github.com/sbondCo/Watcharr/database/entity"
	"gorm.io/gorm"
)

// AlreadyNotified reports whether the user has already been sent this
// notification type for this watched item, so recurring tasks don't spam
// the same reminder every time they run.
func AlreadyNotified(db *gorm.DB, userId uint, watchedId uint, t entity.NotificationType) bool {
	var count int64
	db.Model(&entity.SentNotification{}).
		Where("user_id = ? AND watched_id = ? AND type = ?", userId, watchedId, t).
		Count(&count)
	return count > 0
}

// MarkNotified records that a notification was sent, so AlreadyNotified
// returns true for it from now on.
func MarkNotified(db *gorm.DB, userId uint, watchedId uint, t entity.NotificationType) {
	if res := db.Create(&entity.SentNotification{
		UserID:    userId,
		WatchedID: watchedId,
		Type:      t,
	}); res.Error != nil {
		slog.Error("notify.MarkNotified: Failed to record sent notification!",
			"user_id", userId, "watched_id", watchedId, "type", t, "error", res.Error)
	}
}
