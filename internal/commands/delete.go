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

Accepts one or multiple issue IDs: full UUIDs work from any directory with no
project configured; short prefixes are resolved in the current project. Use --cascade to also delete all
child issues. When run in a terminal, asks for confirmation for each
issue; use --force to skip it. Non-interactive runs (scripts, agents)
do not prompt.

This action is irreversible. If you want to preserve history, consider
closing the issue instead.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cascade, _ := cmd.Flags().GetBool("cascade")
			force, _ := cmd.Flags().GetBool("force")
			in, _ := cmd.InOrStdin().(*os.File)
			prompt := !force && in != nil && isTerminal(in)
			var reader *bufio.Reader
			if prompt {
				reader = bufio.NewReader(in)
			}

			// Full UUIDs need no project; the project list is fetched (and a
			// project required) only when a prefix/local ID needs resolving.
			var pid string
			var beads []api.Bead
			for _, arg := range args {
				if isFullUUID(strings.TrimSpace(arg)) {
					continue
				}
				var err error
				pid, err = d.requireProject(cmd)
				if err != nil {
					return err
				}
				beads, err = api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)
				if err != nil {
					return err
				}
				break
			}
			if pid == "" {
				if err := d.requireAuth(); err != nil {
					return err
				}
			}

			for _, arg := range args {
				uuid, err := resolve.ID(arg, beads)
				if err != nil {
					return fmt.Errorf("failed to resolve %q: %w", arg, err)
				}

				// A UUID outside the current project is sent without one so the
				// server uses the bead's own project.
				bpid := pid
				if !beadInList(beads, uuid) {
					bpid = ""
				}

				if prompt {
					subject := beadSubject(beads, uuid)
					if subject == uuid {
						if b, err := api.JSON[api.Bead](d.Client.Get("/beads/" + uuid)); err == nil && b.Subject != "" {
							subject = b.Subject
						}
					}
					if !confirmDelete(reader, subject, cascade) {
						fmt.Printf("Skipped: %s\n", uuid)
						continue
					}
				}

				err = d.withBeadProject(uuid, bpid, func(p string) error {
					q := projectQuery(p)
					if cascade {
						if q == "" {
							q = "?cascade=true"
						} else {
							q += "&cascade=true"
						}
					}
					_, err := d.Client.Delete("/beads/" + uuid + q)
					return err
				})
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

func beadInList(beads []api.Bead, uuid string) bool {
	for _, b := range beads {
		if b.ID == uuid {
			return true
		}
	}
	return false
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
