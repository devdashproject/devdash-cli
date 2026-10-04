package commands

import (
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newShowCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Full issue detail (JSON; --pretty for a readable view)",
		Long: `Display the full detail for a single issue as pretty-printed JSON.

The output includes all fields: status, priority, type, description,
dependencies, parent reference, timestamps, and any other metadata stored
on the issue. Accepts short ID prefixes — the shortest unique prefix is
enough to identify the issue.

Useful for inspecting an issue's complete state or piping structured data
to other tools like jq. Add --pretty for a short human-readable view.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			uuid, err := resolve.IDWithFetch(args[0], d.Client, pid)
			if err != nil {
				return err
			}

			data, err := d.Client.Get("/beads/" + uuid + "?projectId=" + pid)
			if err != nil {
				return err
			}

			// JSON by default (the raw API response, so new server fields appear automatically)
			pretty, _ := cmd.Flags().GetBool("pretty")
			printOutput(data, pretty, prettyBead)
			if pretty {
				if beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid); err == nil {
					fmt.Print(prettyChildren(beads, uuid))
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("pretty", false, "Human-readable view instead of JSON")
	return cmd
}
