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
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		// Multi doesn't support genre filtering (genre lists differ per
		// movie/tv, and the frontend only offers genre picking for those
		// specific types), so always pass an empty genres value here.
		err = s.discoverMultiTrending(tmdb.TrendingTypeAll, "", meta, &resp)
	case domain.DiscoverFilterInTheatres:
		// Multi doesn't support genre filtering (genre lists differ per
		// movie/tv, and the frontend only offers genre picking for those
		// specific types), so always pass an empty genres value here.
		err = s.discoverMovieInTheatres("", meta, &resp)
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
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		err = s.discoverMultiTrending(tmdb.TrendingTypeMovie, r.Genres, meta, &resp)
	case domain.DiscoverFilterInTheatres:
		err = s.discoverMovieInTheatres(r.Genres, meta, &resp)
	case domain.DiscoverFilterUpcoming:
		err = s.discoverMovieUpcoming(r.Genres, meta, &resp)
	case domain.DiscoverFilterPopular:
		err = s.discoverMoviePopular(r.Genres, meta, &resp)
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
	switch r.Filter {
	case domain.DiscoverFilterTrending:
		err = s.discoverMultiTrending(tmdb.TrendingTypeShow, r.Genres, meta, &resp)
	case domain.DiscoverFilterUpcoming:
		err = s.discoverTvUpcoming(r.Genres, meta, &resp)
	case domain.DiscoverFilterPopular:
		err = s.discoverTvPopular(r.Genres, meta, &resp)
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
		// People have no genres, always pass an empty genres value here.
		err = s.discoverMultiTrending(tmdb.TrendingTypePerson, "", meta, &resp)
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
// TMDB's /trending endpoint has no genre query param (unlike /discover), so
// unlike the other discover*  functions, genre filtering here is done by
// filtering the already-fetched results ourselves using the genre_ids TMDB
// includes on every trending result. Same caveat as the "hide watched"
// poster filter: TotalResults/TotalPages still reflect the unfiltered
// count, since filtering happens after paging.
func (s *Service) discoverMultiTrending(
	t tmdb.TrendingType,
	genres string,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.Trending(t, meta.PageParams.Page, meta.Region)
	if err != nil {
		slog.Error("discoverMulti: Failed to search tmdb!", "error", err)
		return errors.New("content request failed")
	}
	wantedGenres := parseGenreIds(genres)
	for _, v := range tmdbRes.Results {
		if len(wantedGenres) > 0 && !anyGenreMatches(v.GenreIds, wantedGenres) {
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
	genres string,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverMovies(
		tmdb.DiscoverOptions{
			ReleaseDateMin:  time.Now().AddDate(0, 0, -40),
			ReleaseDateMax:  time.Now().AddDate(0, 0, 2),
			WithReleaseType: "2|3",
			WithGenres:      genres,
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
	genres string,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverMovies(
		tmdb.DiscoverOptions{
			ReleaseDateMin:  time.Now(),
			ReleaseDateMax:  time.Now().AddDate(0, 1, 0),
			WithReleaseType: "2|3",
			WithGenres:      genres,
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
	genres string,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverMovies(
		tmdb.DiscoverOptions{WithGenres: genres},
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
	genres string,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverShows(
		tmdb.DiscoverOptions{
			ReleaseDateMin:  time.Now(),
			ReleaseDateMax:  time.Now().AddDate(0, 1, 0),
			WithReleaseType: "2|3",
			WithGenres:      genres,
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
	genres string,
	meta domain.DiscoverRequestMeta,
	resp *domain.DiscoverResponse,
) error {
	tmdbRes, err := s.tmdb.DiscoverShows(
		tmdb.DiscoverOptions{WithGenres: genres},
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
