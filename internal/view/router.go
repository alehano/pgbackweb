package view

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/eduardolat/pgbackweb/internal/logger"
	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/eduardolat/pgbackweb/internal/util/pathutil"
	"github.com/eduardolat/pgbackweb/internal/view/api"
	"github.com/eduardolat/pgbackweb/internal/view/middleware"
	"github.com/eduardolat/pgbackweb/internal/view/static"
	"github.com/eduardolat/pgbackweb/internal/view/web"
	"github.com/labstack/echo/v4"
)

func MountRouter(app *echo.Echo, servs *service.Service) {
	mids := middleware.New(servs)

	// Register global middlewares
	app.Use(mids.NoIndex)

	// Reject cross-site state-changing requests (CSRF). It uses the
	// Sec-Fetch-Site header, or compares Origin with Host for older
	// browsers. GET, HEAD and OPTIONS requests are not affected.
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "cross-origin request rejected", http.StatusForbidden)
	}))
	app.Use(echo.WrapMiddleware(csrf.Handler))

	// Create the base group with the path prefix (if any)
	baseGroup := app.Group(pathutil.GetPathPrefix())

	browserCache := mids.NewBrowserCacheMiddleware(
		middleware.BrowserCacheMiddlewareConfig{
			CacheDuration: time.Hour * 24 * 30,
			ExcludedFiles: []string{"/robots.txt"},
		},
	)

	// Mount static files
	staticFS, err := fs.Sub(static.StaticFs, ".")
	if err != nil {
		logger.FatalError("failed to create static filesystem", logger.KV{"error": err})
	}

	staticGroup := baseGroup.Group("", browserCache)
	staticGroup.StaticFS("/", staticFS)

	apiGroup := baseGroup.Group("/api")
	api.MountRouter(apiGroup, mids, servs)

	webGroup := baseGroup.Group("", mids.InjectReqctx)
	web.MountRouter(webGroup, mids, servs)
}
