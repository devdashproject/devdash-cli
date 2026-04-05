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
		Long: `Look up a bead by its full 36-character UUID and print the raw JSON response.

Unlike most commands, find is not scoped to the current project — it searches
across all projects your account has access to. This makes it useful for
resolving cross-project references or inspecting a bead when you only have
its UUID.

Requires a full UUID (with dashes); short IDs are not accepted.`,
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
			_ = json.Unmarshal(data, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
}
