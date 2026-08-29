package entity

import "github.com/sbondCo/Watcharr/database/dbmodel"

type NotificationType string

const (
	// A PLANNED movie/show released today.
	NotificationTypeRelease NotificationType = "release"
	// A previously-watched show got a new season (the FINISHED -> PLANNED
	// automation in season.RefreshFinishedShowsForNewSeasons).
	NotificationTypeNewSeason NotificationType = "newseason"
)

// SentNotification records that we've already notified a user about one
// event for one watched item, so the daily notification task doesn't spam
// the same reminder every time it runs.
type SentNotification struct {
	dbmodel.GormModelNoDel
	UserID    uint             `gorm:"uniqueIndex:idx_sent_notification;not null"`
	WatchedID uint             `gorm:"uniqueIndex:idx_sent_notification;not null"`
	Type      NotificationType `gorm:"uniqueIndex:idx_sent_notification;not null"`
}
