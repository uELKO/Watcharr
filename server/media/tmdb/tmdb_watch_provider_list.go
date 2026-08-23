package tmdb

import (
	"errors"
	"log/slog"
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

// WatchProviderListItem is our own (camelCase) shape, returned to the
// frontend. Kept separate from the raw TMDB response shape below it.
type WatchProviderListItem struct {
	ProviderID   int    `json:"providerId"`
	ProviderName string `json:"providerName"`
	LogoPath     string `json:"logoPath"`
}

// Raw TMDB response shape (snake_case, as TMDB sends it).
type tmdbWatchProviderListItem struct {
	ProviderID   int    `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	LogoPath     string `json:"logo_path"`
}

type tmdbWatchProviderListResponse struct {
	Results []tmdbWatchProviderListItem `json:"results"`
}

// WatchProviderList returns the streaming services TMDB knows about for
// "movie" or "tv" in a region, for a provider-picker UI (as opposed to
// Regions(), which lists the regions themselves).
func (t *TMDB) WatchProviderList(mediaType string, region string) ([]WatchProviderListItem, error) {
	cacheKey := cache.CreateCacheKey("WatchProviderList", mediaType, region)
	cached := new([]WatchProviderListItem)
	if cache.GetCache(ContentStore, cacheKey, &cached) {
		slog.Debug("WatchProviderList: Returning cache.")
		return *cached, nil
	}
	ep := "/watch/providers/movie"
	if mediaType == "tv" {
		ep = "/watch/providers/tv"
	}
	resp := new(tmdbWatchProviderListResponse)
	err := t.req(ep, map[string]string{"watch_region": region}, &resp)
	if err != nil {
		slog.Error("WatchProviderList: Request failed!", "error", err)
		return nil, errors.New("request failed")
	}
	out := make([]WatchProviderListItem, 0, len(resp.Results))
	for _, r := range resp.Results {
		out = append(out, WatchProviderListItem{
			ProviderID:   r.ProviderID,
			ProviderName: r.ProviderName,
			LogoPath:     r.LogoPath,
		})
	}
	ContentStore.Set(cacheKey, &out, time.Hour*24)
	return out, nil
}
