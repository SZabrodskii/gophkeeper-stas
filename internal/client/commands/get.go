package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get entry by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		entry, err := app.API.GetEntry(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		fmt.Printf("ID:      %s\n", entry.ID)
		fmt.Printf("Type:    %s\n", entry.EntryType)
		fmt.Printf("Name:    %s\n", entry.Name)
		fmt.Printf("Created: %s\n", entry.CreatedAt)
		fmt.Printf("Updated: %s\n", entry.UpdatedAt)

		var data map[string]string
		if err := json.Unmarshal(entry.Data, &data); err == nil {
			fmt.Println("Data:")
			for k, v := range data {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
