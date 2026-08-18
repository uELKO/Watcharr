package season

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WatchedSeasonAddRequest struct {
	WatchedID       uint                 `json:"watchedId"`
	SeasonNumber    int                  `json:"seasonNumber"`
	Status          entity.WatchedStatus `json:"status"`
	Rating          int8                 `json:"rating" binding:"max=10"`
	AddActivity     entity.ActivityType  `json:"-"`
	AddActivityDate time.Time            `json:"-"`
	// Data to add to activity if the season is created.
	// Combined with data we already add.
	AddActivityData map[string]interface{} `json:"-"`
}

type WatchedSeasonAddResponse struct {
	WatchedSeasons []entity.WatchedSeason `json:"watchedSeasons"`
	AddedActivity  entity.Activity        `json:"addedActivity"`
	// Response from the season->episode cascade hook (see hookSeasonStatusChanged).
	SeasonStatusChangedHookResponse SeasonStatusChangedHookResponse `json:"seasonStatusChangedHookResponse,omitempty"`
}

type SeasonStatusChangedHookResponse struct {
	// Watched episodes that were added/updated by the cascade, so the
	// client can update its state without needing a full refresh.
	WatchedEpisodes []entity.WatchedEpisode `json:"watchedEpisodes,omitempty"`
	// Set if every season of the show is now FINISHED/DROPPED and we
	// updated the show's own status to match.
	NewShowStatus entity.WatchedStatus `json:"newShowStatus,omitempty"`
	Errors        []string             `json:"errors,omitempty"`
}

type UserProvider interface {
	UserGetSettings(userId uint) (entity.UserSettings, error)
}

// Implemented by episode.Service. Can't import that package directly (it
// already imports this one, for WatchedSeasonProvider), so this only uses
// primitive/entity types rather than episode's own request/response structs.
type WatchedEpisodeProvider interface {
	// Sets a single episode's status without running the normal
	// episode-status-changed automation (which would redundantly try to
	// update the season/show status we're already explicitly setting here).
	// Used only for the season->episode cascade below.
	SetEpisodeStatusForSeasonCascade(userId uint, watchedId uint, seasonNumber int, episodeNumber int, status entity.WatchedStatus, addActivity entity.ActivityType) (entity.WatchedEpisode, error)
	// Episode numbers in a season that already have a watched entry (any
	// status), so the cascade only backfills ones the user hasn't touched.
	GetWatchedEpisodeNumbersInSeason(userId uint, watchedId uint, seasonNumber int) ([]int, error)
}

type Service struct {
	db               *gorm.DB
	activityProvider domain.ActivityAddProvider
	tmdb             *tmdb.TMDB
	userProvider     UserProvider
	// Set after construction via SetWatchedEpisodeProvider, since
	// episode.Service depends on this service and so can't exist yet
	// when this service is constructed.
	wep WatchedEpisodeProvider
}

func NewService(
	db *gorm.DB,
	activityProvider domain.ActivityAddProvider,
	tmdb *tmdb.TMDB,
	userProvider UserProvider,
) *Service {
	return &Service{
		db:               db,
		activityProvider: activityProvider,
		tmdb:             tmdb,
		userProvider:     userProvider,
	}
}

// See the `wep` field comment on Service.
func (s *Service) SetWatchedEpisodeProvider(wep WatchedEpisodeProvider) {
	s.wep = wep
}

// Add/edit a watched season.
func (s *Service) AddWatchedSeason(userId uint, ar WatchedSeasonAddRequest) (WatchedSeasonAddResponse, error) {
	slog.Debug("Adding watched season item", "userId", userId, "watchedID", ar.WatchedID, "season", ar.SeasonNumber)
	// 1. Make sure watched item exists and it is the correct type (TV)
	var w entity.Watched
	if resp := s.db.
		Where("id = ? AND user_id = ?", ar.WatchedID, userId).
		Preload("Content").
		Preload("WatchedSeasons").
		Find(&w); resp.Error != nil {
		slog.Error("Failed when adding a watched season", "error", "failed to get watched item from db")
		return WatchedSeasonAddResponse{}, errors.New("failed when retrieving watched item")
	}
	if w.ID == 0 {
		slog.Error("Failed when adding a watched season", "error", "watched item does not exist in db")
		return WatchedSeasonAddResponse{}, errors.New("can't add a watched season for a show that doesnt have a status itself")
	}
	if w.Content.Type != entity.SHOW {
		return WatchedSeasonAddResponse{}, errors.New("can't add watched season for non show content")
	}
	found := false
	updated := false
	for i, ws := range w.WatchedSeasons {
		if ws.SeasonNumber == ar.SeasonNumber {
			slog.Debug("Existing watched season item found, updating existing")
			found = true
			if ar.Status != "" && ar.Status != w.WatchedSeasons[i].Status {
				w.WatchedSeasons[i].Status = ar.Status
				updated = true
			}
			if ar.Rating != 0 && ar.Rating != w.WatchedSeasons[i].Rating {
				w.WatchedSeasons[i].Rating = ar.Rating
				updated = true
			}
			break
		}
	}
	var addedActivity entity.Activity
	if !found {
		slog.Debug("Existing watched season not found, adding as new entry")
		w.WatchedSeasons = append(w.WatchedSeasons, entity.WatchedSeason{
			UserID:       userId,
			WatchedID:    ar.WatchedID,
			SeasonNumber: ar.SeasonNumber,
			Status:       ar.Status,
			Rating:       ar.Rating,
		})
	}
	if resp := s.db.Save(&w.WatchedSeasons); resp.Error != nil {
		slog.Debug("Failed to save watched season item in db", "error", resp.Error)
		return WatchedSeasonAddResponse{}, errors.New("failed to save")
	}
	// Add activity
	if found {
		// Only add change activity if we actually updated a value
		// (changing value to same value doesn't count).
		if updated {
			if ar.Status != "" {
				json, _ := json.Marshal(map[string]interface{}{"season": ar.SeasonNumber, "status": ar.Status})
				addedActivity, _ = s.activityProvider.AddActivity(
					userId,
					domain.ActivityAddProps{
						WatchedID: w.ID,
						Type:      entity.SEASON_STATUS_CHANGED,
						Data:      string(json),
					},
					false,
				)
			}
			if ar.Rating != 0 {
				json, _ := json.Marshal(map[string]interface{}{"season": ar.SeasonNumber, "rating": ar.Rating})
				addedActivity, _ = s.activityProvider.AddActivity(
					userId,
					domain.ActivityAddProps{
						WatchedID: w.ID,
						Type:      entity.SEASON_RATING_CHANGED,
						Data:      string(json),
					},
					false,
				)
			}
		}
	} else {
		actData := map[string]interface{}{"season": ar.SeasonNumber, "status": ar.Status, "rating": ar.Rating}
		if len(ar.AddActivityData) > 0 {
			for k, v := range ar.AddActivityData {
				if _, ok := ar.AddActivityData[k]; ok {
					actData[k] = v
				}
			}
		}
		json, _ := json.Marshal(actData)
		act := domain.ActivityAddProps{WatchedID: w.ID, Type: entity.SEASON_ADDED, Data: string(json)}
		if ar.AddActivity != "" {
			act.Type = ar.AddActivity
		}
		if !ar.AddActivityDate.IsZero() {
			act.CustomDate = &ar.AddActivityDate
		}
		addedActivity, _ = s.activityProvider.AddActivity(userId, act, false)
	}
	resp := WatchedSeasonAddResponse{
		WatchedSeasons: w.WatchedSeasons,
		AddedActivity:  addedActivity,
	}
	// If the season was just set to FINISHED (created that way, or changed
	// to it), backfill any episodes in it that don't have a status yet.
	if ar.Status == entity.FINISHED && (!found || updated) {
		resp.SeasonStatusChangedHookResponse =
			s.hookSeasonStatusChanged(userId, w.ID, w.Content.TmdbID, ar.SeasonNumber, w.Status)
	}
	return resp, nil
}

// Called after a season has been explicitly marked FINISHED. Backfills any
// episode in it that doesn't have a watched entry yet as FINISHED too, so
// "mark whole season watched" doesn't leave individual episodes still
// toggleable/inconsistent with the season you just finished. Episodes the
// user already set a status for (eg one they marked DROPPED) are left alone.
// `showStatus` is the show's status *before* this call, so we know whether
// it's safe to auto-advance it (eg never override a deliberate DROPPED).
func (s *Service) hookSeasonStatusChanged(userId uint, watchedId uint, showTmdbId int, seasonNum int, showStatus entity.WatchedStatus) SeasonStatusChangedHookResponse {
	hookResponse := SeasonStatusChangedHookResponse{}
	if s.wep == nil {
		slog.Error("hookSeasonStatusChanged: No WatchedEpisodeProvider set, cannot continue.")
		hookResponse.Errors = append(hookResponse.Errors, "no episode provider configured")
		return hookResponse
	}
	userSettings, err := s.userProvider.UserGetSettings(userId)
	if err != nil {
		slog.Error("hookSeasonStatusChanged: Failed to get user settings! Hook will continue.", "error", err)
	} else if !*userSettings.AutomateShowStatuses {
		slog.Debug("hookSeasonStatusChanged: User has AutomateShowStatuses disabled. Skipping hook.", "user_id", userId)
		return hookResponse
	}
	seasonDetails, err := s.tmdb.SeasonDetails(strconv.Itoa(showTmdbId), strconv.Itoa(seasonNum))
	if err != nil {
		slog.Error("hookSeasonStatusChanged: Failed to get season details!", "error", err)
		hookResponse.Errors = append(hookResponse.Errors, "failed to get season details")
		return hookResponse
	}
	existing, err := s.wep.GetWatchedEpisodeNumbersInSeason(userId, watchedId, seasonNum)
	if err != nil {
		slog.Error("hookSeasonStatusChanged: Failed to get existing watched episodes!", "error", err)
		hookResponse.Errors = append(hookResponse.Errors, "failed to get existing watched episodes")
		return hookResponse
	}
	existingSet := make(map[int]bool, len(existing))
	for _, e := range existing {
		existingSet[e] = true
	}
	for _, ep := range seasonDetails.Episodes {
		if existingSet[ep.EpisodeNumber] {
			continue
		}
		we, err := s.wep.SetEpisodeStatusForSeasonCascade(
			userId, watchedId, seasonNum, ep.EpisodeNumber,
			entity.FINISHED, entity.EPISODE_ADDED_AUTO,
		)
		if err != nil {
			slog.Error("hookSeasonStatusChanged: Failed to set episode status!",
				"episode", ep.EpisodeNumber, "error", err)
			hookResponse.Errors = append(hookResponse.Errors,
				fmt.Sprintf("failed to set episode %d status", ep.EpisodeNumber))
			continue
		}
		hookResponse.WatchedEpisodes = append(hookResponse.WatchedEpisodes, we)
	}
	slog.Debug(fmt.Sprintf(
		"hookSeasonStatusChanged: Backfilled episodes for season %d as finished.", seasonNum))

	// If every season of the show is now done, the show itself is done too.
	// Never override a show the user has explicitly DROPPED though - that's
	// a deliberate decision, not something automation should second-guess.
	if showStatus == entity.DROPPED {
		return hookResponse
	}
	allFinished, err := s.allSeasonsFinished(userId, watchedId, showTmdbId)
	if err != nil {
		slog.Error("hookSeasonStatusChanged: Failed to check if all seasons are finished!", "error", err)
		hookResponse.Errors = append(hookResponse.Errors, "failed to check if all seasons are finished")
		return hookResponse
	}
	if allFinished {
		if res := s.db.Model(&entity.Watched{}).Where("id = ?", watchedId).Update("status", entity.FINISHED); res.Error != nil {
			slog.Error("hookSeasonStatusChanged: Failed to update show status to finished!", "error", res.Error)
			hookResponse.Errors = append(hookResponse.Errors, "failed to update show status to finished")
			return hookResponse
		}
		hookResponse.NewShowStatus = entity.FINISHED
		json, _ := json.Marshal(map[string]interface{}{
			"status": entity.FINISHED,
			"reason": fmt.Sprintf("Season %d was the last remaining season.", seasonNum),
		})
		s.activityProvider.AddActivity(
			userId,
			domain.ActivityAddProps{
				WatchedID: watchedId,
				Type:      entity.STATUS_CHANGED_AUTO,
				Data:      string(json),
			},
			false,
		)
	}
	return hookResponse
}

// True if every real season (excludes specials and seasons TMDB has no
// episodes for yet) of the show has a FINISHED or DROPPED watched entry.
func (s *Service) allSeasonsFinished(userId uint, watchedId uint, showTmdbId int) (bool, error) {
	showDetails, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{ID: strconv.Itoa(showTmdbId)})
	if err != nil {
		return false, err
	}
	var watchedSeasons []entity.WatchedSeason
	if res := s.db.Where("watched_id = ? AND user_id = ?", watchedId, userId).Find(&watchedSeasons); res.Error != nil {
		return false, res.Error
	}
	done := make(map[int]bool, len(watchedSeasons))
	for _, ws := range watchedSeasons {
		if ws.Status == entity.FINISHED || ws.Status == entity.DROPPED {
			done[ws.SeasonNumber] = true
		}
	}
	for _, se := range showDetails.Seasons {
		if se.SeasonNumber <= 0 || se.EpisodeCount <= 0 {
			continue
		}
		if !done[se.SeasonNumber] {
			return false, nil
		}
	}
	return true, nil
}

// Remove a watched season
func (s *Service) RmWatchedSeason(userId uint, seasonId uint) (entity.Activity, error) {
	slog.Debug("rmWatchedSeason called", "user_id", userId, "season_id", seasonId)
	var watchedSeason entity.WatchedSeason
	resp := s.db.
		Clauses(clause.Returning{}).
		Model(&entity.WatchedSeason{}).
		Unscoped().
		Where("id = ? AND user_id = ?", seasonId, userId).
		Delete(&watchedSeason)
	if resp.Error != nil {
		slog.Error("Failed when removing a watched season", "error", resp.Error)
		return entity.Activity{}, errors.New("failed when removing watched season")
	}
	if resp.RowsAffected == 0 {
		slog.Error("Failed when removing a watched season", "error", "zero rows affected")
		return entity.Activity{}, errors.New("wasn't removed from db.. may not exist")
	}
	slog.Debug("rmWatchedSeason, deleted row", "row", watchedSeason)
	if watchedSeason.ID != 0 {
		json, _ := json.Marshal(map[string]interface{}{
			"season": watchedSeason.SeasonNumber,
			"status": watchedSeason.Status,
			"rating": watchedSeason.Rating,
		})
		addedActivity, _ := s.activityProvider.AddActivity(
			userId,
			domain.ActivityAddProps{
				WatchedID: watchedSeason.WatchedID,
				Type:      entity.SEASON_REMOVED,
				Data:      string(json),
			},
			false,
		)
		return addedActivity, nil
	}
	return entity.Activity{}, errors.New("removed, but failed to add activity entry")
}

func (s *Service) GetWatchedSeason(userId uint, watchedId uint, seasonNumber int) (*entity.WatchedSeason, error) {
	var ws *entity.WatchedSeason
	if res := s.db.Model(&entity.WatchedSeason{}).Where("watched_id = ? AND season_number = ? AND user_id = ?", watchedId, seasonNumber, userId).Take(&ws); res.Error != nil {
		slog.Error("getWatchedSeason: Failed to get:", "error", res.Error.Error())
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return &entity.WatchedSeason{}, errors.New("failed to get watched season")
	}
	return ws, nil
}
