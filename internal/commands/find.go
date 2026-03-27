package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newFindCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "find <uuid>",
		Short: "Look up a bead by full UUID across all projects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}

			uuid := args[0]
			if len(uuid) != 36 || strings.Count(uuid, "-") != 4 {
				return fmt.Errorf("find requires a full UUID (36 characters with dashes)")
			}

			data, err := d.Client.Get("/beads/" + uuid)
			if err != nil {
				return err
			}

			var raw json.RawMessage
			json.Unmarshal(data, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
}
