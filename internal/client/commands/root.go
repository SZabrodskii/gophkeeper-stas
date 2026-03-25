package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SZabrodskii/gophkeeper-stas/internal/client/api"
	"github.com/SZabrodskii/gophkeeper-stas/internal/client/keyring"
	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
	"github.com/SZabrodskii/gophkeeper-stas/pkg/buildinfo"
)

// App holds shared dependencies for all CLI commands.
type App struct {
	Config  *config.ClientConfig
	API     *api.HTTPClient
	Keyring keyring.TokenStore
}

var app App

var rootCmd = &cobra.Command{
	Use:     "gophkeeper",
	Short:   "GophKeeper — менеджер паролей и секретов",
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", buildinfo.Version, buildinfo.Commit, buildinfo.Date),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.NewClientConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if s, _ := cmd.Flags().GetString("server"); s != "" {
			cfg.ServerAddress = s
		}
		if ins, _ := cmd.Flags().GetBool("insecure"); ins {
			cfg.TLSInsecure = true
		}

		app.Config = cfg
		app.API = api.NewHTTPClient(cfg)
		app.Keyring = keyring.New()
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().String("server", "", "server address (overrides SERVER_ADDRESS)")
	rootCmd.PersistentFlags().Bool("insecure", false, "skip TLS verification")
}

// Execute runs the root cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
