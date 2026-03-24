package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopybara/httpbara"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
)

type newRouterParams struct {
	fx.In

	Handlers []*httpbara.Handler `group:"handlers"`
	Logger   httpbara.Logger
	ZapLog   *zap.Logger
}

// NewRouter assembles the gin router with httpbara handlers and middleware.
func NewRouter(params newRouterParams) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	accessLog, err := httpbara.NewAccessLogMiddleware(params.Logger)
	if err != nil {
		return nil, fmt.Errorf("access log middleware: %w", err)
	}

	taskTrackerMw, err := httpbara.NewTaskTrackerMiddleware(params.Logger, httpbara.NewActiveTaskTracker())
	if err != nil {
		return nil, fmt.Errorf("task tracker middleware: %w", err)
	}

	_, err = httpbara.New(params.Handlers,
		httpbara.WithGinEngine(r),
		httpbara.WithLogger(params.Logger),
		httpbara.WithRootMiddlewares(accessLog, taskTrackerMw),
		httpbara.WithTaskTracker(httpbara.NewActiveTaskTracker()),
	)
	if err != nil {
		return nil, fmt.Errorf("httpbara engine: %w", err)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r, nil
}

// StartServer registers fx lifecycle hooks to start and gracefully stop the HTTPS server.
func StartServer(lc fx.Lifecycle, cfg config.ListenConfig, router *gin.Engine, logger *zap.Logger) {
	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: router,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		ReadHeaderTimeout: 10 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				logger.Info("starting server", zap.String("address", cfg.Address))
				if err := srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey); err != nil && err != http.ErrServerClosed {
					logger.Fatal("server failed", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down server")
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("server shutdown: %w", err)
			}
			logger.Info("server stopped")
			return nil
		},
	})
}
