package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize data with server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		since, err := app.Keyring.GetLastSync()
		if err != nil {
			since = time.Time{}
		}

		resp, err := app.API.Sync(cmd.Context(), since)
		if err != nil {
			return err
		}

		if len(resp.Entries) == 0 {
			fmt.Println("No updates")
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTYPE\tNAME\tUPDATED")
			for _, e := range resp.Entries {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.ID, e.EntryType, e.Name, e.UpdatedAt)
			}
			w.Flush()

			for _, e := range resp.Entries {
				var data map[string]string
				if json.Unmarshal(e.Data, &data) == nil {
					fmt.Printf("\n--- %s (%s) ---\n", e.Name, e.EntryType)
					for k, v := range data {
						fmt.Printf("  %s: %s\n", k, v)
					}
				}
			}
		}

		serverTime, err := time.Parse(time.RFC3339, resp.ServerTime)
		if err != nil {
			return fmt.Errorf("parse server time: %w", err)
		}

		if err := app.Keyring.SetLastSync(serverTime); err != nil {
			return fmt.Errorf("save last sync: %w", err)
		}

		fmt.Printf("\nSynced at %s\n", resp.ServerTime)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
