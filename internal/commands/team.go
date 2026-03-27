package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/spf13/cobra"
)

func newTeamCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "team",
		Short: "List project team members",
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
			json.Unmarshal(data, &members)

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
