package config

import (
	"github.com/caarlos0/env/v11"
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/pkg/logging"
)

// ServerConfig — полная конфигурация приложения, парсится из env.
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

// DBConfig — конфигурация подключения к базе данных.
type DBConfig struct {
	DSN string
}

// AuthConfig — конфигурация аутентификации.
type AuthConfig struct {
	JWTSecret     string
	EncryptionKey string
}

// ListenConfig — конфигурация HTTP-сервера.
type ListenConfig struct {
	Address string
	TLSCert string
	TLSKey  string
}

// ConfigOut распаковывает ServerConfig в типизированные суб-конфиги через fx.Out.
type ConfigOut struct {
	fx.Out

	Full    *ServerConfig
	DB      DBConfig
	Auth    AuthConfig
	Listen  ListenConfig
	Logging logging.Config
}

func NewServerConfig() (ConfigOut, error) {
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return ConfigOut{}, err
	}
	return ConfigOut{
		Full: &cfg,
		DB: DBConfig{
			DSN: cfg.DatabaseDSN,
		},
		Auth: AuthConfig{
			JWTSecret:     cfg.JWTSecret,
			EncryptionKey: cfg.EncryptionKey,
		},
		Listen: ListenConfig{
			Address: cfg.Address,
			TLSCert: cfg.TLSCert,
			TLSKey:  cfg.TLSKey,
		},
		Logging: logging.Config{Level: cfg.LogLevel},
	}, nil
}

var Module = fx.Module("config",
	fx.Provide(NewServerConfig),
)
