package handler

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func RegisterHealthRoutes(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func RegisterSignalHandler(lc fx.Lifecycle, shutdowner fx.Shutdowner, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
				sig := <-sigCh
				logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
				_ = shutdowner.Shutdown()
			}()
			return nil
		},
	})
}
