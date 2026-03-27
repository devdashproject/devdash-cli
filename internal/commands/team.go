package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(teamCmd)
}

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "List project team members",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := requireProject()
		if err != nil {
			return err
		}

		data, err := client.Get("/projects/" + pid + "/members?format=compact")
		if err != nil {
			return err
		}

		var members []api.TeamMember
		if err := json.Unmarshal(data, &members); err != nil {
			return fmt.Errorf("failed to parse members: %w", err)
		}

		// Cap at 20 members (matches Bash)
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
