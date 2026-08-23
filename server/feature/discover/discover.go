package discover

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"gorm.io/gorm"
)

type Service struct {
	db   *gorm.DB
	cfg  *config.ServerConfig
	tmdb *tmdb.TMDB
}

func NewService(
	db *gorm.DB,
	cfg *config.ServerConfig,
	tmdb *tmdb.TMDB,
) *Service {
	return &Service{
		db,
		cfg,
		tmdb,
	}
}

// Bundles the optional refinement filters (genre/provider/year/rating) so
// they don't have to be threaded through every discover* function as
// separate positional params. Not every filter applies to every mode -
// callers zero out what doesn't apply (eg genres/providers are cleared for
// DiscoverMulti, since those lists are per movie/tv).
type discoverFilters struct {
	genres    string
	providers string
	year      int
	minRating float64
}

func filtersFromRequest(r domain.DiscoverRequest) discoverFilters {
	return discoverFilters{
		genres:    r.Genres,
		providers: r.Providers,
		year:      r.Year,
		minRating: r.MinRating,
	}
}

// `Limit` is not supported.
func (s *Service) Discover(
	// User request
	r domain.DiscoverRequest,
	// Extra data
	meta domain.DiscoverRequestMeta,
) (domain.DiscoverResponse, error) {
	resp := domain.DiscoverResponse{}

	switch r.Type {
	case domain.SearchTypeMulti:
		return s.DiscoverMulti(r, meta)
	case domain.SearchTypeShow:
		return s.DiscoverTv(r, meta)
	case domain.SearchTypePerson:
		return s.DiscoverPeople(r, meta)
	case domain.SearchTypeMovie:
		return s.DiscoverMovie(r, meta)
	case domain.SearchTypeGame:
		return s.DiscoverGame(r, meta)
	}
	return resp, nil
}

// Discover Multi. Just for tmdb.
func (s *Service) DiscoverMulti(
	r domain.DiscoverRequest,
	meta domain.DiscoverRequestMeta,
) (domain.DiscoverResponse, error) {
	resp := domain.DiscoverResponse{}
	var err error
	// Multi doesn't support genre/provider filtering (those lists differ per
	// movie/tv, and the frontend only offers picking for those specific
	// types), but year/rating apply fine to anything.
	f := filtersFromRequest(r)
	f.genres = ""
	f.providers = ""
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		err = s.discoverMultiTrending(tmdb.TrendingTypeAll, f, meta, &resp)
	case domain.DiscoverFilterInTheatres:
		err = s.discoverMovieInTheatres(f, meta, &resp)
	default:
		slog.Error("DiscoverMulti: Unsupported filter.")
		return resp, errors.New("unsupported filter")
	}
	return resp, err
}

// Discover movies.
func (s *Service) DiscoverMovie(
	r domain.DiscoverRequest,
	meta domain.DiscoverRequestMeta,
) (domain.DiscoverResponse, error) {
	resp := domain.DiscoverResponse{}
	var err error
	f := filtersFromRequest(r)
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		err = s.discoverMultiTrending(tmdb.TrendingTypeMovie, f, meta, &resp)
	case domain.DiscoverFilterInTheatres:
		err = s.discoverMovieInTheatres(f, meta, &resp)
	case domain.DiscoverFilterUpcoming:
		err = s.discoverMovieUpcoming(f, meta, &resp)
	case domain.DiscoverFilterPopular:
		err = s.discoverMoviePopular(f, meta, &resp)
	default:
		slog.Error("DiscoverMovie: Unsupported filter.")
		return resp, errors.New("unsupported filter")
	}
	return resp, err
}

// Discover shows.
func (s *Service) DiscoverTv(
	r domain.DiscoverRequest,
	meta domain.DiscoverRequestMeta,
) (domain.DiscoverResponse, error) {
	resp := domain.DiscoverResponse{}
	var err error
	f := filtersFromRequest(r)
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		err = s.discoverMultiTrending(tmdb.TrendingTypeShow, f, meta, &resp)
	case domain.DiscoverFilterUpcoming:
		err = s.discoverTvUpcoming(f, meta, &resp)
	case domain.DiscoverFilterPopular:
		err = s.discoverTvPopular(f, meta, &resp)
	default:
		slog.Error("DiscoverMovie: Unsupported filter.")
		return resp, errors.New("unsupported filter")
	}
	return resp, err
}

// Discover people.
func (s *Service) DiscoverPeople(
	r domain.DiscoverRequest,
	meta domain.DiscoverRequestMeta,
) (domain.DiscoverResponse, error) {
	resp := domain.DiscoverResponse{}
	var err error
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		// People have no genres/providers/year/rating.
		err = s.discoverMultiTrending(tmdb.TrendingTypePerson, discoverFilters{}, meta, &resp)
	case domain.DiscoverFilterPopular:
		err = s.discoverPeoplePopular(meta, &resp)
	default:
		slog.Error("DiscoverMulti: Unsupported filter.")
		return resp, errors.New("unsupported filter")
	}
	return resp, err
}

// Discover games.
func (s *Service) DiscoverGame(
	r domain.DiscoverRequest,
	meta domain.DiscoverRequestMeta,
) (domain.DiscoverResponse, error) {
	resp := domain.DiscoverResponse{}
	var err error
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		err = s.discoverGameTrending(&resp)
	case domain.DiscoverFilterUpcoming:
		err = s.discoverGameUpcoming(&resp)
	default:
		slog.Error("DiscoverGame: Unsupported filter.")
		return resp, errors.New("unsupported filter")
	}
	return resp, err
}

// Discover anything that is trending on TMDB (including combined).
//
// TMDB's /trending endpoint has no genre/provider/year/rating query params
// (unlike /discover), so unlike the other discover* functions, filtering
// here is done by filtering the already-fetched results ourselves, using
// the genre_ids/release date/vote_average TMDB includes on every trending
// result. providers is the one filter that can't be done this way (trending
// results carry no per-item watch-provider data at all), so it's always
// empty by the time it gets here (callers clear it). Same caveat as the
// "hide watched" poster filter: TotalResults/TotalPages still reflect the
// unfiltered count, since filtering happens after paging.
func (s *Service) discoverMultiTrending(
	t tmdb.TrendingType,
	f discoverFilters,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.Trending(t, meta.PageParams.Page, meta.Region)
	if err != nil {
		slog.Error("discoverMulti: Failed to search tmdb!", "error", err)
		return errors.New("content request failed")
	}
	wantedGenres := parseGenreIds(f.genres)
	for _, v := range tmdbRes.Results {
		if len(wantedGenres) > 0 && !anyGenreMatches(v.GenreIds, wantedGenres) {
			continue
		}
		if f.year > 0 {
			d := v.ReleaseDate
			if d == "" {
				d = v.FirstAirDate
			}
			// "This year or newer" - same semantics as the release_date.gte
			// param used for the non-trending discover* functions.
			releaseYear, err := strconv.Atoi(strings.SplitN(d, "-", 2)[0])
			if err != nil || releaseYear < f.year {
				continue
			}
		}
		if f.minRating > 0 && float64(v.VoteAverage) < f.minRating {
			continue
		}
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = tmdbRes.Page
	resp.TotalPages = tmdbRes.TotalPages
	resp.TotalResults = int64(tmdbRes.TotalResults)
	return nil
}

// Parses our pipe separated genre id string (eg "28|12") into ints,
// skipping anything that doesn't parse.
func parseGenreIds(genres string) []int {
	if genres == "" {
		return nil
	}
	parts := strings.Split(genres, "|")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		if id, err := strconv.Atoi(p); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// True if any of `have` is present in `wanted` (ie. an OR match).
func anyGenreMatches(have []int, wanted []int) bool {
	for _, h := range have {
		for _, w := range wanted {
			if h == w {
				return true
			}
		}
	}
	return false
}

func (s *Service) discoverMovieInTheatres(
	f discoverFilters,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverMovies(
		tmdb.DiscoverOptions{
			ReleaseDateMin:     time.Now().AddDate(0, 0, -40),
			ReleaseDateMax:     time.Now().AddDate(0, 0, 2),
			WithReleaseType:    "2|3",
			WithGenres:         f.genres,
			WithWatchProviders: f.providers,
			Year:               f.year,
			MinRating:          f.minRating,
		},
		meta.PageParams.Page,
		meta.Region,
	)
	if err != nil {
		slog.Error("discoverMovieInTheatres: Failed to search tmdb!",
			"error", err)
		return errors.New("content request failed")
	}
	for _, v := range tmdbRes.Results {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = tmdbRes.Page
	resp.TotalPages = tmdbRes.TotalPages
	resp.TotalResults = int64(tmdbRes.TotalResults)
	return nil
}

func (s *Service) discoverMovieUpcoming(
	f discoverFilters,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverMovies(
		tmdb.DiscoverOptions{
			ReleaseDateMin:     time.Now(),
			ReleaseDateMax:     time.Now().AddDate(0, 1, 0),
			WithReleaseType:    "2|3",
			WithGenres:         f.genres,
			WithWatchProviders: f.providers,
			Year:               f.year,
			MinRating:          f.minRating,
		},
		meta.PageParams.Page,
		meta.Region,
	)
	if err != nil {
		slog.Error("discoverMovieUpcoming: Failed to search tmdb!",
			"error", err)
		return errors.New("content request failed")
	}
	for _, v := range tmdbRes.Results {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = tmdbRes.Page
	resp.TotalPages = tmdbRes.TotalPages
	resp.TotalResults = int64(tmdbRes.TotalResults)
	return nil
}

func (s *Service) discoverMoviePopular(
	f discoverFilters,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverMovies(
		tmdb.DiscoverOptions{
			WithGenres:         f.genres,
			WithWatchProviders: f.providers,
			Year:               f.year,
			MinRating:          f.minRating,
		},
		meta.PageParams.Page,
		meta.Region,
	)
	if err != nil {
		slog.Error("discoverMoviePopular: Failed to search tmdb!",
			"error", err)
		return errors.New("content request failed")
	}
	for _, v := range tmdbRes.Results {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = tmdbRes.Page
	resp.TotalPages = tmdbRes.TotalPages
	resp.TotalResults = int64(tmdbRes.TotalResults)
	return nil
}

func (s *Service) discoverTvUpcoming(
	f discoverFilters,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverShows(
		tmdb.DiscoverOptions{
			ReleaseDateMin:     time.Now(),
			ReleaseDateMax:     time.Now().AddDate(0, 1, 0),
			WithReleaseType:    "2|3",
			WithGenres:         f.genres,
			WithWatchProviders: f.providers,
			Year:               f.year,
			MinRating:          f.minRating,
		},
		meta.PageParams.Page,
		meta.Region,
	)
	if err != nil {
		slog.Error("discoverTvUpcoming: Failed to search tmdb!",
			"error", err)
		return errors.New("content request failed")
	}
	for _, v := range tmdbRes.Results {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = tmdbRes.Page
	resp.TotalPages = tmdbRes.TotalPages
	resp.TotalResults = int64(tmdbRes.TotalResults)
	return nil
}

func (s *Service) discoverTvPopular(
	f discoverFilters,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverShows(
		tmdb.DiscoverOptions{
			WithGenres:         f.genres,
			WithWatchProviders: f.providers,
			Year:               f.year,
			MinRating:          f.minRating,
		},
		meta.PageParams.Page,
		meta.Region,
	)
	if err != nil {
		slog.Error("discoverTvPopular: Failed to search tmdb!",
			"error", err)
		return errors.New("content request failed")
	}
	for _, v := range tmdbRes.Results {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = tmdbRes.Page
	resp.TotalPages = tmdbRes.TotalPages
	resp.TotalResults = int64(tmdbRes.TotalResults)
	return nil
}

func (s *Service) discoverPeoplePopular(
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.PopularPeople(
		meta.PageParams.Page,
	)
	if err != nil {
		slog.Error("discoverPeoplePopular: Failed to search tmdb!",
			"error", err)
		return errors.New("content request failed")
	}
	for _, v := range tmdbRes.Results {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = tmdbRes.Page
	resp.TotalPages = tmdbRes.TotalPages
	resp.TotalResults = int64(tmdbRes.TotalResults)
	return nil
}

func (s *Service) discoverGameTrending(
	resp *domain.DiscoverResponse,
) error {
	igdbRes, err := s.cfg.TWITCH.Trending()
	if err != nil {
		slog.Error("discoverGameTrending: Failed to search igdb!", "error", err)
		return errors.New("content request failed")
	}
	for _, v := range igdbRes {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = 1
	resp.TotalPages = 1
	resp.TotalResults = int64(len(igdbRes))
	return nil
}

func (s *Service) discoverGameUpcoming(
	resp *domain.DiscoverResponse,
) error {
	igdbRes, err := s.cfg.TWITCH.Upcoming()
	if err != nil {
		slog.Error("discoverGameUpcoming: Failed to search igdb!", "error", err)
		return errors.New("content request failed")
	}
	for _, v := range igdbRes {
		resp.Results = append(
			resp.Results,
			v.AsMedia(),
		)
	}
	resp.Page = 1
	resp.TotalPages = 1
	resp.TotalResults = int64(len(igdbRes))
	return nil
}
