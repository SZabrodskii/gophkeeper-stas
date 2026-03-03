package config

import (
	"github.com/caarlos0/env/v11"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Address       string `env:"ADDRESS"         envDefault:":8443"`
	DatabaseDSN   string `env:"DATABASE_DSN,required"`
	JWTSecret     string `env:"JWT_SECRET,required"`
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`
	TLSCert       string `env:"TLS_CERT,required"`
	TLSKey        string `env:"TLS_KEY,required"`
	MaxBinarySize int64  `env:"MAX_BINARY_SIZE" envDefault:"10485760"`
	LogLevel      string `env:"LOG_LEVEL"       envDefault:"info"`
}

func NewServerConfig() (*ServerConfig, error) {
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

var Module = fx.Module("config",
	fx.Provide(NewServerConfig),
)
