package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "registration of user",
	RunE: func(cmd *cobra.Command, args []string) error {
		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")

		resp, err := app.API.Register(cmd.Context(), login, password)
		if err != nil {
			return err
		}

		if err := app.Keyring.Set(resp.Token); err != nil {
			return fmt.Errorf("save token: %w", err)
		}

		fmt.Println("Registered successfully")
		return nil
	},
}

func init() {
	registerCmd.Flags().StringP("login", "l", "", "login")
	registerCmd.Flags().StringP("password", "p", "", "password")
	_ = registerCmd.MarkFlagRequired("login")
	_ = registerCmd.MarkFlagRequired("password")
	rootCmd.AddCommand(registerCmd)
}
