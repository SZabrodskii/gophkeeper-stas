package logging

import (
	"context"

	"github.com/gopybara/httpbara"
	"github.com/gopybara/httpbara/pkg/httpbarazap"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds the logging configuration.
type Config struct {
	Level string
}

// NewLogger creates a production zap logger at the configured level.
func NewLogger(cfg Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level.SetLevel(level)

	return zapCfg.Build()
}

// NewHttpbaraLogger wraps a zap logger as an httpbara-compatible logger.
func NewHttpbaraLogger(logger *zap.Logger) httpbara.Logger {
	return httpbarazap.New(logger)
}

// Module provides the zap logger and httpbara logger adapter via fx DI.
var Module = fx.Module("logging",
	fx.Provide(NewLogger, NewHttpbaraLogger),
	fx.Invoke(func(lc fx.Lifecycle, logger *zap.Logger) {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				_ = logger.Sync()
				return nil
			},
		})
	}),
)
