package justwatch

import (
	"strconv"
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

const searchQuery = `
query GetSearchTitles(
	$searchTitlesFilter: TitleFilter!
	$country: Country!
	$language: Language!
	$first: Int!
) {
	popularTitles(
		country: $country
		filter: $searchTitlesFilter
		first: $first
		sortBy: POPULAR
		sortRandomSeed: 0
	) {
		edges {
			node {
				objectType
				content(country: $country, language: $language) {
					title
					externalIds {
						tmdbId
					}
					scoring {
						imdbScore
						imdbVotes
						tomatoMeter
						certifiedFresh
					}
				}
			}
		}
	}
}`

// Scoring is JustWatch's aggregated rating data for one title (itself
// sourced from IMDb/Rotten Tomatoes).
type Scoring struct {
	ImdbScore *float64 `json:"imdbScore"`
	// JustWatch returns this as a float (eg 2272987 as 2.272987e+06), even
	// though it's always a whole number of votes.
	ImdbVotes      *float64 `json:"imdbVotes"`
	TomatoMeter    *int     `json:"tomatoMeter"`
	CertifiedFresh *bool    `json:"certifiedFresh"`
}

type SearchEntry struct {
	// "MOVIE" or "SHOW".
	ObjectType string `json:"objectType"`
	Content    struct {
		Title       string `json:"title"`
		ExternalIds struct {
			TmdbID string `json:"tmdbId"`
		} `json:"externalIds"`
		Scoring *Scoring `json:"scoring"`
	} `json:"content"`
}

// Search looks up a title by name on JustWatch, most relevant first.
func (j *JustWatch) Search(title string, country string, count int) ([]SearchEntry, error) {
	var resp struct {
		PopularTitles struct {
			Edges []struct {
				Node SearchEntry `json:"node"`
			} `json:"edges"`
		} `json:"popularTitles"`
	}
	if err := j.req(
		"GetSearchTitles",
		map[string]any{
			"searchTitlesFilter": map[string]any{"searchQuery": title},
			"country":            country,
			"language":           "en",
			"first":              count,
		},
		searchQuery,
		&resp,
	); err != nil {
		return nil, err
	}
	entries := make([]SearchEntry, len(resp.PopularTitles.Edges))
	for i, e := range resp.PopularTitles.Edges {
		entries[i] = e.Node
	}
	return entries, nil
}

// ScoringForTmdbId searches JustWatch for `title` and returns the Scoring
// for whichever result's own TMDB id matches tmdbId, or nil (with no error)
// if none of the first few results match - JustWatch may simply have no
// listing for it, or title-based search didn't surface it.
func (j *JustWatch) ScoringForTmdbId(title string, country string, tmdbId int) (*Scoring, error) {
	cacheKey := cache.CreateCacheKey("JustWatchScoring", title, country, tmdbId)
	cached := new(*Scoring)
	if cache.GetCache(ContentStore, cacheKey, cached) {
		return *cached, nil
	}
	entries, err := j.Search(title, country, 5)
	if err != nil {
		return nil, err
	}
	want := strconv.Itoa(tmdbId)
	var found *Scoring
	for _, e := range entries {
		if e.Content.ExternalIds.TmdbID == want {
			found = e.Content.Scoring
			break
		}
	}
	ContentStore.Set(cacheKey, found, time.Hour*24)
	return found, nil
}
