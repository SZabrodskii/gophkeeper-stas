package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list of entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		entries, err := app.API.ListEntries(cmd.Context())
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("No entries found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tNAME\tUPDATED")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.ID, e.EntryType, e.Name, e.UpdatedAt)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
