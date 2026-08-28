package tmdb

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

func (t *TMDB) DiscoverMovies(
	o DiscoverOptions,
	pageNum int,
	region string,
) (DiscoverMovies, error) {
	resp := new(DiscoverMovies)
	reqParams := map[string]string{
		"page":   strconv.Itoa(pageNum),
		"region": region,
	}
	if o.WithWatchProviders != "" {
		reqParams["watch_region"] = region
	}
	t.applyDiscoverOptionsToMap(true, o, reqParams)
	cacheKey := cache.CreateCacheKey(
		"DiscoverMovies",
		pageNum,
		reqParams)
	if cache.GetCache(ContentStore, cacheKey, &resp) {
		slog.Debug("DiscoverMovies: Returning cache.")
		return *resp, nil
	}
	err := t.req("/discover/movie", reqParams, &resp)
	if err != nil {
		slog.Error("DiscoverMovies: Request failed!", "error", err)
		return DiscoverMovies{}, errors.New("request failed")
	}
	ContentStore.Set(cacheKey, resp, time.Hour*24)
	return *resp, nil
}

func (t *TMDB) DiscoverShows(
	o DiscoverOptions,
	pageNum int,
	region string,
) (DiscoverShows, error) {
	resp := new(DiscoverShows)
	reqParams := map[string]string{
		"page":   strconv.Itoa(pageNum),
		"region": region,
	}
	if o.WithWatchProviders != "" {
		reqParams["watch_region"] = region
	}
	t.applyDiscoverOptionsToMap(false, o, reqParams)
	cacheKey := cache.CreateCacheKey(
		"DiscoverShows",
		pageNum,
		reqParams)
	if cache.GetCache(ContentStore, cacheKey, &resp) {
		slog.Debug("DiscoverShows: Returning cache.")
		return *resp, nil
	}
	err := t.req("/discover/tv", reqParams, &resp)
	if err != nil {
		slog.Error("DiscoverShows: Request failed!", "error", err)
		return DiscoverShows{}, errors.New("request failed")
	}
	ContentStore.Set(cacheKey, resp, time.Hour*24)
	return *resp, nil
}

func (t *TMDB) applyDiscoverOptionsToMap(
	// Some properties are named differently for sorting the same thing as far
	// as we care, so we need to differenciate to name them properly.
	forMovie bool,
	o DiscoverOptions,
	m map[string]string,
) {
	releaseDateMinKey := "release_date.gte"
	releaseDateMaxKey := "release_date.lte"
	withReleaseTypeKey := "with_release_type"
	if !forMovie {
		// Replace with names for equivalent tv filters
		releaseDateMinKey = "first_air_date.gte"
		releaseDateMaxKey = "first_air_date.lte"
		withReleaseTypeKey = "with_type"
	}
	// o.Year ("from this year on") and o.ReleaseDateMin (eg the Upcoming/In
	// Theatres modes' own fixed date range) both want to set the same *.gte
	// param - use whichever is more restrictive (later date) rather than
	// letting one silently overwrite the other.
	releaseDateMin := o.ReleaseDateMin
	if o.Year > 0 {
		yearStart := time.Date(o.Year, time.January, 1, 0, 0, 0, 0, time.UTC)
		if releaseDateMin.IsZero() || yearStart.After(releaseDateMin) {
			releaseDateMin = yearStart
		}
	}
	if !releaseDateMin.IsZero() {
		m[releaseDateMinKey] = releaseDateMin.Format("2006-01-02")
	}
	if !o.ReleaseDateMax.IsZero() {
		m[releaseDateMaxKey] = o.ReleaseDateMax.Format("2006-01-02")
	}
	if o.WithReleaseType != "" {
		m[withReleaseTypeKey] = o.WithReleaseType
	}
	if o.WithGenres != "" {
		// Same param name for both /discover/movie and /discover/tv.
		m["with_genres"] = o.WithGenres
	}
	if o.WithoutGenres != "" {
		m["without_genres"] = o.WithoutGenres
	}
	if o.WithWatchProviders != "" {
		// Same param name for both. watch_region (required for this to take
		// effect) is added by the caller (DiscoverMovies/DiscoverShows).
		m["with_watch_providers"] = o.WithWatchProviders
	}
	if o.MinRating > 0 {
		m["vote_average.gte"] = strconv.FormatFloat(o.MinRating, 'f', -1, 64)
	}
}
