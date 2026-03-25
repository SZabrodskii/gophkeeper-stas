package config

import "github.com/caarlos0/env/v11"

// ClientConfig holds CLI client settings parsed from environment variables.
type ClientConfig struct {
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:"https://localhost:8443"`
	TLSInsecure   bool   `env:"TLS_INSECURE"   envDefault:"false"`
}

// NewClientConfig parses client environment variables into ClientConfig.
func NewClientConfig() (*ClientConfig, error) {
	var cfg ClientConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
