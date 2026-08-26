package entity

import (
	"time"

	"github.com/sbondCo/Watcharr/database/dbmodel"
)

// ProviderChartSnapshot is one entry (one movie/show at one rank) of a
// provider's Top 10 chart on one day. A recurring task snapshots the
// current chart daily for every provider that's ever been requested via
// the Charts page, so today's ranking can be compared against ~7 days ago
// to show rank movement (up/down).
type ProviderChartSnapshot struct {
	dbmodel.GormModelNoDel
	ProviderID   int         `gorm:"index:idx_provider_snapshot_date;not null"`
	SnapshotDate time.Time   `gorm:"index:idx_provider_snapshot_date;not null"` // truncated to day (UTC midnight)
	TmdbID       int         `gorm:"not null"`
	ContentType  ContentType `gorm:"not null"`
	Rank         int         `gorm:"not null"`
}
