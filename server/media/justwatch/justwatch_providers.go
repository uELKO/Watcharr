package justwatch

import (
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

const providersQuery = `
query GetProviders($country: Country!) {
	packages(country: $country, platform: WEB, includeAddons: true) {
		packageId
		clearName
		shortName
		icon(profile: S100, format: PNG)
	}
}`

// Package is one streaming provider as JustWatch knows it.
type Package struct {
	PackageID int    `json:"packageId"`
	ClearName string `json:"clearName"`
	ShortName string `json:"shortName"`
	// Full icon URL (the raw API response only gives a relative path).
	Icon string `json:"icon"`
}

// Providers returns every provider JustWatch has offers for in the given
// country (ISO 3166-1 alpha-2, e.g. "DE", "US").
func (j *JustWatch) Providers(country string) ([]Package, error) {
	cacheKey := cache.CreateCacheKey("JustWatchProviders", country)
	cached := new([]Package)
	if cache.GetCache(ContentStore, cacheKey, cached) {
		return *cached, nil
	}
	var resp struct {
		Packages []Package `json:"packages"`
	}
	if err := j.req(
		"GetProviders",
		map[string]any{"country": country},
		providersQuery,
		&resp,
	); err != nil {
		return nil, err
	}
	for i := range resp.Packages {
		if resp.Packages[i].Icon != "" {
			resp.Packages[i].Icon = "https://images.justwatch.com" + resp.Packages[i].Icon
		}
	}
	ContentStore.Set(cacheKey, &resp.Packages, time.Hour*24)
	return resp.Packages, nil
}
