package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/resolve"
	"github.com/spf13/cobra"
)

func newShowCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Full issue detail",
		Args:  cobra.ExactArgs(1),
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

			var bead api.Bead
			if err := json.Unmarshal(data, &bead); err != nil {
				return fmt.Errorf("failed to parse bead: %w", err)
			}

			out, err := json.MarshalIndent(bead, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}
}
