package commands

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newShowCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Full issue detail",
		Long: `Display the full detail for a single issue as pretty-printed JSON.

The output includes all fields: status, priority, type, description,
dependencies, parent reference, timestamps, and any other metadata stored
on the issue. Accepts short ID prefixes — the shortest unique prefix is
enough to identify the issue.

Useful for inspecting an issue's complete state or piping structured data
to other tools like jq.`,
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

			// Pretty-print the raw API response so new server fields appear automatically
			var buf bytes.Buffer
			if err := json.Indent(&buf, data, "", "  "); err != nil {
				return fmt.Errorf("failed to format response: %w", err)
			}
			fmt.Println(buf.String())
			return nil
		},
	}
}
