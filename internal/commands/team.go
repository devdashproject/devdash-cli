package commands

import (
	"encoding/json"
	"fmt"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/spf13/cobra"
)

func newTeamCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "team",
		Short: "List project team members",
		Long: `List project team members (up to 20).

For each active member, prints their display name, @username, email
address, and role. Pending invitations are shown separately with just
the email and a "pending" label.

Useful when you need to find a teammate's email for --owner assignment
or want to confirm who has access to the project.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			data, err := d.Client.Get("/projects/" + pid + "/members?format=compact")
			if err != nil {
				return err
			}

			var members []api.TeamMember
			_ = json.Unmarshal(data, &members)

			if len(members) > 20 {
				members = members[:20]
			}

			for _, m := range members {
				if m.Status == "pending" {
					fmt.Printf("%s   pending\n", m.Email)
				} else {
					name := m.Name
					if name == "" {
						name = m.Email
					}
					username := ""
					if m.Username != "" {
						username = fmt.Sprintf("@%s", m.Username)
					}
					fmt.Printf("%s    %s    %s    %s\n", name, username, m.Email, m.Role)
				}
			}
			return nil
		},
	}
}
