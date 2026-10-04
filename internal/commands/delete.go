package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newDeleteCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id> [<id>...]",
		Short: "Delete one or more issues",
		Long: `Permanently delete one or more issues.

Accepts one or multiple issue IDs. Use --cascade to also delete all
child issues. When run in a terminal, asks for confirmation for each
issue; use --force to skip it. Non-interactive runs (scripts, agents)
do not prompt.

This action is irreversible. If you want to preserve history, consider
closing the issue instead.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			cascade, _ := cmd.Flags().GetBool("cascade")
			force, _ := cmd.Flags().GetBool("force")
			in, _ := cmd.InOrStdin().(*os.File)
			prompt := !force && in != nil && isTerminal(in)
			var reader *bufio.Reader
			if prompt {
				reader = bufio.NewReader(in)
			}

			beads, err := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
			if err != nil {
				return err
			}

			for _, arg := range args {
				uuid, err := resolve.ID(arg, beads)
				if err != nil {
					return fmt.Errorf("failed to resolve %q: %w", arg, err)
				}

				if prompt && !confirmDelete(reader, beadSubject(beads, uuid), cascade) {
					fmt.Printf("Skipped: %s\n", uuid)
					continue
				}

				path := "/beads/" + uuid + "?projectId=" + pid
				if cascade {
					path += "&cascade=true"
				}

				_, err = d.Client.Delete(path)
				if err != nil {
					return fmt.Errorf("failed to delete %s: %w", uuid, err)
				}

				fmt.Printf("Deleted: %s\n", uuid)
			}
			return nil
		},
	}
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	cmd.Flags().Bool("cascade", false, "Delete children too")
	return cmd
}

func beadSubject(beads []api.Bead, uuid string) string {
	for _, b := range beads {
		if b.ID == uuid {
			return b.Subject
		}
	}
	return uuid
}

// confirmDelete asks y/N; anything but yes (including a read error) declines.
func confirmDelete(r *bufio.Reader, subject string, cascade bool) bool {
	if cascade {
		fmt.Fprintf(os.Stderr, "Delete '%s' and all its children? [y/N] ", subject)
	} else {
		fmt.Fprintf(os.Stderr, "Delete '%s'? [y/N] ", subject)
	}
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
