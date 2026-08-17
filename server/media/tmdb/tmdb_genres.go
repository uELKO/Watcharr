package tmdb

import (
	"errors"
	"log/slog"
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GenresResponse struct {
	Genres []Genre `json:"genres"`
}

// Genres returns the TMDB genre list for "movie" or "tv".
func (t *TMDB) Genres(mediaType string) (GenresResponse, error) {
	resp := new(GenresResponse)
	cacheKey := cache.CreateCacheKey("Genres", mediaType)
	if cache.GetCache(ContentStore, cacheKey, &resp) {
		slog.Debug("Genres: Returning cache.")
		return *resp, nil
	}
	ep := "/genre/movie/list"
	if mediaType == "tv" {
		ep = "/genre/tv/list"
	}
	err := t.req(ep, map[string]string{}, &resp)
	if err != nil {
		slog.Error("Genres: Request failed!", "error", err)
		return GenresResponse{}, errors.New("request failed")
	}
	ContentStore.Set(cacheKey, resp, time.Hour*24)
	return *resp, nil
}
