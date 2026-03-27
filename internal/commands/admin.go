package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newAdminCmd(d *Deps) *cobra.Command {
	adminCmd := &cobra.Command{Use: "admin", Short: "Admin commands (requires ADMIN_SECRET)"}

	resetCmd := &cobra.Command{
		Use: "reset-user <user-id>", Short: "Reset a user's data", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secret := getAdminSecret()
			if secret == "" {
				return fmt.Errorf("admin secret not found — set ADMIN_SECRET env var or create ~/.config/dev-dash/admin-secret")
			}
			if d.Cfg == nil {
				return fmt.Errorf("configuration not loaded")
			}

			url := d.Cfg.APIURL + "/api/admin/reset-user/" + args[0]
			req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-admin-secret", secret)

			resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}
			var raw json.RawMessage
			json.Unmarshal(body, &raw)
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
	resetCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")
	adminCmd.AddCommand(resetCmd)

	return adminCmd
}

func getAdminSecret() string {
	if s := os.Getenv("ADMIN_SECRET"); s != "" {
		return s
	}
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".config", "dev-dash", "admin-secret"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
