package discover

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/feature/watched/addedtocontent"
	"github.com/sbondCo/Watcharr/router"
	"github.com/sbondCo/Watcharr/util"
)

type WatchedProvider interface {
	GetWatchedItemBySupportedMediaId(userId uint, id uint, t util.SupportedMedia) (entity.Watched, error)
	GetWatchedItemsBySupportedMediaIds(userId uint, c []addedtocontent.IdToTypePair) ([]entity.Watched, error)
}

type Router struct {
	br              *router.BaseRouter
	service         *Service
	watchedProvider WatchedProvider
}

func NewRouter(br *router.BaseRouter, service *Service, watchedProvider WatchedProvider) *Router {
	return &Router{
		br,
		service,
		watchedProvider,
	}
}

func (r *Router) AddRoutes() {
	discover := r.br.Router.Group("/discover").Use(authmiddleware.AuthRequired(r.br.DB, r.br.Cfg))

	// Master discovery
	discover.GET("", router.WhereaboutsRequired(r.br.Cfg), router.PaginatedRequest(true), r.GetDiscover)
	// Per-provider Top 10 charts (movies+shows merged, ranked by popularity)
	discover.GET("/charts", router.WhereaboutsRequired(r.br.Cfg), r.GetCharts)
}

// NOTE: The handler functions use `copier` to copy values from the response
// structs into a new one that includes the user "Watched" data.
// This was done to avoid adding Watched data to the response structs, as they
// are cached in our in-mem cache, which could cause references to pollute the cache
// resulting in user data being leaked to others.
// We are doing to to explicitly not let that case happen.

func (r *Router) GetDiscover(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	pp := c.MustGet("paginationParams").(util.PaginationParams)
	req := domain.DiscoverRequest{
		// Defaults...
		Type:   domain.SearchTypeMulti,
		Filter: domain.DiscoverFilterTrending,
	}
	if err := c.ShouldBind(&req); err != nil {
		slog.Error("GetDiscover: ShouldBind for request params failed!", "error", err)
		c.JSON(
			http.StatusBadRequest,
			router.ErrorResponse{
				Error: "failed to get request parameters or they are invalid",
			},
		)
		return
	}
	resp, err := r.service.Discover(req, domain.DiscoverRequestMeta{
		PageParams: pp,
		Region:     c.MustGet("userCountry").(string),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	ww := domain.DiscoverResponse{}
	if err := copier.Copy(&ww, &resp); err != nil {
		slog.Error("GetDiscover: Failed to copy", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to prepare response"},
		)
		return
	}
	if err := addedtocontent.AddList(
		r.watchedProvider,
		userId,
		ww.Results,
		func(i int, w *entity.Watched) {
			ww.Results[i].Watched = domain.NewWatchedDtoForLists(w)
		},
	); err != nil {
		slog.Error("GetDiscover: Failed to add watched to content!", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	c.JSON(http.StatusOK, ww)
}

func (r *Router) GetCharts(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	providersQ := c.Query("providers")
	if providersQ == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "providers query param required"})
		return
	}
	charts, err := r.service.Charts(strings.Split(providersQ, "|"), c.MustGet("userCountry").(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	for i := range charts {
		items := charts[i].Items
		if err := addedtocontent.AddList(
			r.watchedProvider,
			userId,
			items,
			func(j int, w *entity.Watched) {
				items[j].Watched = domain.NewWatchedDtoForLists(w)
			},
		); err != nil {
			slog.Error("GetCharts: Failed to add watched to content!", "error", err)
			c.JSON(
				http.StatusInternalServerError,
				router.ErrorResponse{Error: "failed to add watched data to response"},
			)
			return
		}
	}
	c.JSON(http.StatusOK, charts)
}
