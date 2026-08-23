package domain

import (
	"github.com/go-playground/validator/v10"
	"github.com/sbondCo/Watcharr/util"
)

type DiscoverFilter string

const (
	// Generic "What's Trending?" (all)
	DiscoverFilterTrending DiscoverFilter = "trending"
	// Popular stuff (basically trending but over months instead of just a day).
	DiscoverFilterPopular DiscoverFilter = "popular"
	// Upcoming content (all).
	DiscoverFilterUpcoming DiscoverFilter = "upcoming"
	// What's streaming (movies/tv).
	DiscoverFilterStreaming DiscoverFilter = "streaming"
	// What's in theatres (movies).
	DiscoverFilterInTheatres DiscoverFilter = "intheatres"
)

type DiscoverRequest struct {
	// The type of content we want to discover.
	// Reusing the SearchType enum here, but if this needs to diverge,
	// then make our own enum in this file.
	Type SearchType `form:"type" binding:"validsearchtype"`
	// A main filter.
	// Not every `Type` of discover will support all Filters (service funcs
	// will error individually based on what they support).
	Filter DiscoverFilter `form:"filter" binding:"validdiscoverfilter"`
	// Optional: comma separated TMDB genre ids to filter by. Only applies to
	// filters backed by /discover/{movie,tv} (Popular/Upcoming/In Theatres);
	// TMDB's /trending endpoint doesn't support genre filtering, so this is
	// ignored for DiscoverFilterTrending.
	Genres string `form:"genres"`
	// Optional: pipe separated TMDB watch provider ids to filter by. Same
	// scope restriction as Genres, but unlike genres there's no per-item
	// provider data on trending results to filter with client-side either,
	// so this is unconditionally ignored for DiscoverFilterTrending.
	Providers string `form:"providers"`
	// Optional: exact release year to filter by. Like Genres, this works
	// for DiscoverFilterTrending too via a server-side post-filter (trending
	// results already include a release/first-air date).
	Year int `form:"year"`
	// Optional: minimum average rating (0-10). Same as Year re: Trending.
	MinRating float64 `form:"minRating"`
}

// Extra data that we provide to the Discover service func.
type DiscoverRequestMeta struct {
	PageParams util.PaginationParams
	Region     string
}

type DiscoverResponse struct {
	util.PaginationResponse[Media, util.None]
}

var ValidDiscoverFilter validator.Func = func(fl validator.FieldLevel) bool {
	st, ok := fl.Field().Interface().(DiscoverFilter)
	if ok {
		switch st {
		case DiscoverFilterTrending,
			DiscoverFilterPopular,
			DiscoverFilterUpcoming,
			DiscoverFilterStreaming,
			DiscoverFilterInTheatres:
			return true
		}
	}
	return false
}
