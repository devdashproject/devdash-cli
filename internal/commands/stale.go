package commands

import (
	"fmt"
	"os"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/output"
	"github.com/spf13/cobra"
)

func newStaleCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stale",
		Short: "In-progress issues with no recent activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			since, _ := cmd.Flags().GetString("since")
			var sinceFilter string
			if since != "" {
				sinceFilter, err = output.FormatSinceISO(since)
				if err != nil {
					return err
				}
			}

			var stale []api.Bead
			for _, b := range beads {
				if b.Status != "in_progress" || b.StaleMinutes <= 0 {
					continue
				}
				if sinceFilter != "" && b.UpdatedAt.Format("2006-01-02T15:04:05.000Z") < sinceFilter {
					continue
				}
				stale = append(stale, b)
			}

			if len(stale) == 0 {
				fmt.Fprintln(os.Stderr, "No stale issues.")
				return nil
			}

			fmt.Printf("Stale in-progress issues (%d):\n\n", len(stale))
			for _, b := range stale {
				fmt.Println(output.FormatStaleLine(b))
			}
			return nil
		},
	}
	cmd.Flags().String("since", "", "Filter by updatedAt (Nh, Nd, Nw, or YYYY-MM-DD)")
	return cmd
}
