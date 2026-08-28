package discover

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/media/justwatch"
	"github.com/sbondCo/Watcharr/media/tmdb"
)

// ChartItem is one entry of a provider's Top 10 chart.
type ChartItem struct {
	domain.Media
	// Position in this provider's Top 10, 1-indexed.
	Rank int `json:"rank"`
	// Rank movement vs JustWatch's own last-known rank: "up", "down",
	// "same", or "" if unknown.
	Movement string `json:"movement,omitempty"`
}

// ProviderChart is the ranked Top 10 of movies+shows currently popular on
// one streaming provider.
type ProviderChart struct {
	// JustWatch's provider short name (eg "nfx" for Netflix).
	Provider string      `json:"provider"`
	Items    []ChartItem `json:"items"`
}

const chartSize = 10

// ChartProviders returns every streaming provider JustWatch knows about in
// the given region, for the Charts page's provider picker.
func (s *Service) ChartProviders(region string) ([]justwatch.Package, error) {
	return s.justwatch.Providers(region)
}

// Charts returns a Top 10 chart for each given provider (JustWatch short
// names), movies and shows merged, ranked by JustWatch's own "popular on
// this provider" ordering, with real rank-movement data from JustWatch
// (see media/justwatch). A provider that fails is skipped rather than
// failing the whole request - JustWatch's API is unofficial/undocumented
// and can misbehave.
func (s *Service) Charts(providerShortNames []string, region string) ([]ProviderChart, error) {
	charts := make([]ProviderChart, 0, len(providerShortNames))
	for _, p := range providerShortNames {
		items, err := s.chartForProvider(p, region)
		if err != nil {
			slog.Warn("Charts: chartForProvider failed, skipping", "provider", p, "error", err)
			continue
		}
		charts = append(charts, ProviderChart{Provider: p, Items: items})
	}
	if len(charts) == 0 && len(providerShortNames) > 0 {
		return nil, errors.New("failed to build any charts")
	}
	return charts, nil
}

func (s *Service) chartForProvider(providerShortName string, region string) ([]ChartItem, error) {
	entries, err := s.justwatch.Popular(region, providerShortName, chartSize)
	if err != nil {
		return nil, err
	}
	items := make([]ChartItem, 0, len(entries))
	for i, e := range entries {
		tmdbId, err := strconv.Atoi(e.Content.ExternalIds.TmdbID)
		if err != nil || tmdbId == 0 {
			// Some JustWatch entries (eg exclusives with no TMDB match)
			// have no TMDB id - skip, we have nowhere to link them to.
			continue
		}
		media, err := s.mediaForTmdbId(tmdbId, e.ObjectType, region)
		if err != nil {
			slog.Warn("chartForProvider: TMDB lookup failed, skipping", "tmdbId", tmdbId, "error", err)
			continue
		}
		movement := ""
		if len(e.StreamingCharts.Edges) > 0 {
			switch e.StreamingCharts.Edges[0].StreamingChartInfo.Trend {
			case "UP":
				movement = "up"
			case "DOWN":
				movement = "down"
			case "STABLE":
				movement = "same"
			}
		}
		items = append(items, ChartItem{
			Media:    media,
			Rank:     i + 1,
			Movement: movement,
		})
	}
	return items, nil
}

// mediaForTmdbId fetches full TMDB details for an id JustWatch gave us, so
// chart cards use the exact same poster/rating/etc conventions as the rest
// of the app (and benefit from the same TMDB response caching) instead of
// JustWatch's own image CDN.
func (s *Service) mediaForTmdbId(tmdbId int, objectType string, region string) (domain.Media, error) {
	id := strconv.Itoa(tmdbId)
	if objectType == "SHOW" {
		d, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{ID: id, Country: region})
		if err != nil {
			return domain.Media{}, err
		}
		return d.AsMedia(), nil
	}
	d, err := s.tmdb.MovieDetails(tmdb.MovieDetailsOptions{ID: id, Country: region})
	if err != nil {
		return domain.Media{}, err
	}
	return d.AsMedia(), nil
}
