package handler

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/gopybara/httpbara"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type healthHandlerRoutes struct {
	Health httpbara.Route `route:"GET /health"`
}

// HealthHandler serves the /health liveness probe.
type HealthHandler struct {
	healthHandlerRoutes
}

// NewHealthHandler creates a HealthHandler and registers its routes.
func NewHealthHandler() (FxHandler, error) {
	h := &HealthHandler{}
	return asFxHandler(httpbara.AsHandler(h))
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RegisterSignalHandler listens for OS signals and triggers graceful shutdown.
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
