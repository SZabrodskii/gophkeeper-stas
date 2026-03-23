package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to an existing account",
	RunE: func(cmd *cobra.Command, args []string) error {
		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")

		resp, err := app.API.Login(cmd.Context(), login, password)
		if err != nil {
			return err
		}

		if err := app.Keyring.Set(resp.Token); err != nil {
			return fmt.Errorf("save token: %w", err)
		}

		fmt.Println("Logged in successfully")
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("login", "l", "", "login")
	loginCmd.Flags().StringP("password", "p", "", "password")
	_ = loginCmd.MarkFlagRequired("login")
	_ = loginCmd.MarkFlagRequired("password")
	rootCmd.AddCommand(loginCmd)
}
