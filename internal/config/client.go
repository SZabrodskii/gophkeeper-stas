package config

import "github.com/caarlos0/env/v11"

type ClientConfig struct {
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:"https://localhost:8443"`
	TLSInsecure   bool   `env:"TLS_INSECURE"   envDefault:"false"`
}

func NewClientConfig() (*ClientConfig, error) {
	var cfg ClientConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
