package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/eduardolat/pgbackweb/internal/config"
	"github.com/eduardolat/pgbackweb/internal/cron"
	"github.com/eduardolat/pgbackweb/internal/database"
	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/integration"
	"github.com/eduardolat/pgbackweb/internal/logger"
	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/eduardolat/pgbackweb/internal/util/pathutil"
	"github.com/eduardolat/pgbackweb/internal/view"
	"github.com/labstack/echo/v4"
)

func main() {
	env, err := config.GetEnv()
	if err != nil {
		logger.FatalError("error getting environment variables", logger.KV{"error": err})
	}

	pathutil.SetPathPrefix(env.PBW_PATH_PREFIX)

	cr, err := cron.New()
	if err != nil {
		logger.FatalError("error initializing cron scheduler", logger.KV{"error": err})
	}
	cr.Start()
	defer func() {
		if err := cr.Shutdown(); err != nil {
			logger.Error("error shutting down cron scheduler", logger.KV{"error": err})
		}
	}()

	db := database.Connect(env)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("error closing database connection", logger.KV{"error": err})
		}
	}()
	dbgen := dbgen.New(db)

	ints := integration.New()
	servs := service.New(env, dbgen, cr, ints)
	initSchedule(cr, servs)

	app := echo.New()
	app.HideBanner = true
	app.HidePort = true
	app.IPExtractor = newIPExtractor(env.PBW_TRUSTED_PROXIES)
	view.MountRouter(app, servs)

	address := fmt.Sprintf("%s:%s", env.PBW_LISTEN_HOST, env.PBW_LISTEN_PORT)
	localURL := fmt.Sprintf("http://localhost:%s%s", env.PBW_LISTEN_PORT, pathutil.GetPathPrefix())
	logger.Info("server started at "+localURL, logger.KV{
		"listenHost": env.PBW_LISTEN_HOST,
		"listenPort": env.PBW_LISTEN_PORT,
	})
	if err := app.Start(address); err != nil {
		logger.FatalError("error starting server", logger.KV{"error": err})
	}
}

// newIPExtractor returns how Echo finds the client IP (used by the login rate
// limit and the logs). By default Echo trusts the left-most X-Forwarded-For
// value, which any client can set. Here the header is only used when the
// request comes from one of the trusted proxies.
func newIPExtractor(trustedProxies []string) echo.IPExtractor {
	if len(trustedProxies) == 0 {
		return echo.ExtractIPDirect()
	}

	opts := []echo.TrustOption{
		echo.TrustLoopback(false),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(false),
	}
	for _, p := range trustedProxies {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.Contains(p, "/") {
			if ip := net.ParseIP(p); ip != nil && ip.To4() != nil {
				p += "/32"
			} else {
				p += "/128"
			}
		}
		_, ipNet, err := net.ParseCIDR(p)
		if err != nil {
			logger.FatalError("invalid PBW_TRUSTED_PROXIES entry", logger.KV{
				"entry": p, "error": err,
			})
		}
		opts = append(opts, echo.TrustIPRange(ipNet))
	}

	return echo.ExtractIPFromXFFHeader(opts...)
}
