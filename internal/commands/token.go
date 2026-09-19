package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// defaultTokenName generates a reasonable token name when the caller doesn't
// provide one, so "token create" works with no arguments.
func defaultTokenName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "cli"
	}
	return fmt.Sprintf("%s-%s", host, time.Now().Format("2006-01-02-150405"))
}

func newTokenCmd(d *Deps) *cobra.Command {
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Manage API tokens",
		Long: `Manage API tokens for authenticating with the devdash API.

Subcommands let you create, list, and revoke tokens. Use "token create <name>"
to generate a new named token, "token list" to show all active tokens, and
"token revoke <id>" to revoke one by its ID.

Tokens are scoped to your user account and grant the same access as your
session. Treat them like passwords.`,
	}

	tokenCmd.AddCommand(&cobra.Command{
		Use: "create [name]", Short: "Create a new API token", Args: cobra.MaximumNArgs(1),
		Long: `Create a new API token.

Naming is optional — with no argument, a name is generated from your
hostname and the current time ("token create <name>" to pick your own).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			name := defaultTokenName()
			if len(args) > 0 && args[0] != "" {
				name = args[0]
			}
			data, err := d.Client.Post("/auth/tokens", map[string]string{"name": name})
			if err != nil {
				return err
			}
			var raw json.RawMessage
			_ = json.Unmarshal(data, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	})

	tokenCmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List API tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			data, err := d.Client.Get("/auth/tokens")
			if err != nil {
				return err
			}
			var raw json.RawMessage
			_ = json.Unmarshal(data, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	})

	tokenCmd.AddCommand(&cobra.Command{
		Use: "revoke <id>", Short: "Revoke an API token", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			_, err := d.Client.Delete("/auth/tokens/" + args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Revoked token: %s\n", args[0])
			return nil
		},
	})

	return tokenCmd
}
