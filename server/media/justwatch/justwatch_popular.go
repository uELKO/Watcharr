package justwatch

import (
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

const popularQuery = `
query GetPopularTitles(
	$popularTitlesFilter: TitleFilter
	$country: Country!
	$language: Language!
	$first: Int!
	$offset: Int = 0
) {
	popularTitles(
		country: $country
		filter: $popularTitlesFilter
		first: $first
		sortBy: POPULAR
		sortRandomSeed: 0
		offset: $offset
	) {
		edges {
			node {
				objectType
				content(country: $country, language: $language) {
					title
					externalIds {
						tmdbId
					}
				}
				streamingCharts(country: $country) {
					edges {
						streamingChartInfo {
							rank
							trend
							trendDifference
						}
					}
				}
			}
		}
	}
}`

// PopularEntry is one title from a Popular query result.
type PopularEntry struct {
	// "MOVIE" or "SHOW".
	ObjectType string `json:"objectType"`
	Content    struct {
		Title       string `json:"title"`
		ExternalIds struct {
			// TMDB numeric ID, as a string (JustWatch returns it that way).
			TmdbID string `json:"tmdbId"`
		} `json:"externalIds"`
	} `json:"content"`
	StreamingCharts struct {
		Edges []struct {
			StreamingChartInfo struct {
				// Rank in the requested country overall (not scoped to the
				// requested provider) - useful as a rough popularity signal,
				// but a title's position in the returned list is what
				// reflects "popular on this specific provider".
				Rank int `json:"rank"`
				// "UP", "DOWN", or "STABLE".
				Trend           string `json:"trend"`
				TrendDifference int    `json:"trendDifference"`
			} `json:"streamingChartInfo"`
		} `json:"edges"`
	} `json:"streamingCharts"`
}

// Popular returns up to `count` titles JustWatch currently considers
// popular, restricted to one provider's catalog (so the returned order is
// effectively "what's popular on this provider" in the given country).
// packageShortName is JustWatch's own 3-letter provider code (e.g. "nfx"
// for Netflix), from Providers.
func (j *JustWatch) Popular(country string, packageShortName string, count int) ([]PopularEntry, error) {
	cacheKey := cache.CreateCacheKey("JustWatchPopular", country, packageShortName, count)
	cached := new([]PopularEntry)
	if cache.GetCache(ContentStore, cacheKey, cached) {
		return *cached, nil
	}
	var resp struct {
		PopularTitles struct {
			Edges []struct {
				Node PopularEntry `json:"node"`
			} `json:"edges"`
		} `json:"popularTitles"`
	}
	if err := j.req(
		"GetPopularTitles",
		map[string]any{
			"popularTitlesFilter": map[string]any{
				"packages":    []string{packageShortName},
				"objectTypes": []string{"MOVIE", "SHOW"},
			},
			"country":  country,
			"language": "en",
			"first":    count,
			"offset":   0,
		},
		popularQuery,
		&resp,
	); err != nil {
		return nil, err
	}
	entries := make([]PopularEntry, len(resp.PopularTitles.Edges))
	for i, e := range resp.PopularTitles.Edges {
		entries[i] = e.Node
	}
	// Shorter TTL than other justwatch caches - chart rank/trend is the
	// point of this data, so it shouldn't go too stale.
	ContentStore.Set(cacheKey, &entries, time.Hour)
	return entries, nil
}
