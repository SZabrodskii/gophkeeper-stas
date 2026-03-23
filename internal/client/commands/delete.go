package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		if err := app.API.DeleteEntry(cmd.Context(), args[0]); err != nil {
			return err
		}

		fmt.Println("Deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
