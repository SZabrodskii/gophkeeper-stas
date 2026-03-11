package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopybara/httpbara"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
	"github.com/SZabrodskii/gophkeeper-stas/internal/middleware"
)

type newRouterParams struct {
	fx.In

	Handlers []*httpbara.Handler `group:"handlers"`
	Logger   *zap.Logger
}

func NewRouter(params newRouterParams) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(params.Logger))

	_, err := httpbara.New(params.Handlers, httpbara.WithGinEngine(r))
	if err != nil {
		return nil, fmt.Errorf("httpbara engine: %w", err)
	}

	return r, nil
}

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
