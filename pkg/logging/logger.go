package logging

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level string
}

func NewLogger(cfg Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level.SetLevel(level)

	return zapCfg.Build()
}

var Module = fx.Module("logging",
	fx.Provide(NewLogger),
	fx.Invoke(func(lc fx.Lifecycle, logger *zap.Logger) {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				_ = logger.Sync()
				return nil
			},
		})
	}),
)
