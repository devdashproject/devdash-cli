package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func newSelfUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update devdash to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("cannot determine executable path: %w", err)
			}
			exe, _ = filepath.EvalSymlinks(exe)

			if isNPMInstall(exe) {
				fmt.Println("Updating via npm...")
				c := exec.Command("npm", "update", "-g", "@devdashproject/devdash-cli")
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				return c.Run()
			}

			if isGitInstall(exe) {
				dir := filepath.Dir(filepath.Dir(exe))
				fmt.Printf("Updating via git pull in %s...\n", dir)
				c := exec.Command("git", "-C", dir, "pull", "origin", "main")
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					return err
				}
				fmt.Println("Rebuilding...")
				build := exec.Command("go", "build", "-o", exe, "./cmd/devdash")
				build.Dir = dir
				build.Stdout = os.Stdout
				build.Stderr = os.Stderr
				return build.Run()
			}

			// Fetch latest version from GitHub API
			fmt.Println("Fetching latest version...")
			out, err := exec.Command("curl", "-fsSL",
				"https://api.github.com/repos/jasonmassey/devdash-cli-go/releases/latest").Output()
			if err != nil {
				return fmt.Errorf("failed to check latest version: %w", err)
			}
			var release struct {
				TagName string `json:"tag_name"`
			}
			if err := json.Unmarshal(out, &release); err != nil {
				return fmt.Errorf("failed to parse release info: %w", err)
			}
			version := strings.TrimPrefix(release.TagName, "v")
			if version == "" {
				return fmt.Errorf("could not determine latest version")
			}

			fmt.Printf("Downloading devdash v%s...\n", version)
			archive := fmt.Sprintf("devdash_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
			url := fmt.Sprintf(
				"https://github.com/jasonmassey/devdash-cli-go/releases/download/%s/%s",
				release.TagName, archive)

			tmpDir, err := os.MkdirTemp("", "devdash-update-*")
			if err != nil {
				return fmt.Errorf("failed to create temp dir: %w", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			archivePath := filepath.Join(tmpDir, archive)
			dl := exec.Command("curl", "-fsSL", "-o", archivePath, url)
			dl.Stderr = os.Stderr
			if err := dl.Run(); err != nil {
				return fmt.Errorf("download failed: %w", err)
			}

			extract := exec.Command("tar", "-xzf", archivePath, "-C", tmpDir)
			if err := extract.Run(); err != nil {
				return fmt.Errorf("extraction failed: %w", err)
			}

			src := filepath.Join(tmpDir, "devdash")
			if err := copyFile(src, exe); err != nil {
				return fmt.Errorf("failed to install binary: %w", err)
			}
			_ = os.Chmod(exe, 0755)
			fmt.Printf("Updated to devdash v%s\n", version)
			return nil
		},
	}
}

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove devdash and its configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			exe, _ := os.Executable()
			home, _ := os.UserHomeDir()
			configDir := filepath.Join(home, ".config", "dev-dash")
			targets := []string{exe, configDir}

			if dryRun {
				fmt.Println("Would remove:")
				for _, t := range targets {
					fmt.Printf("  %s\n", t)
				}
				return nil
			}

			for _, t := range targets {
				if err := os.RemoveAll(t); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", t, err)
				} else {
					fmt.Printf("Removed: %s\n", t)
				}
			}
			fmt.Println("\nDevdash has been uninstalled. You may also want to remove any shell aliases.")
			return nil
		},
	}
	cmd.Flags().Bool("dry-run", false, "Preview what will be removed")
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	return cmd
}

func newAliasSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "alias-setup",
		Short: "Add 'dd' alias to your shell RC file",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			shell := os.Getenv("SHELL")
			var rcFile string
			switch {
			case strings.Contains(shell, "zsh"):
				rcFile = filepath.Join(home, ".zshrc")
			case strings.Contains(shell, "bash"):
				rcFile = filepath.Join(home, ".bashrc")
			case strings.Contains(shell, "fish"):
				rcFile = filepath.Join(home, ".config", "fish", "config.fish")
			default:
				return fmt.Errorf("unsupported shell: %s", shell)
			}

			aliasLine := "alias dd='devdash'"
			if strings.Contains(shell, "fish") {
				aliasLine = "alias dd devdash"
			}

			data, _ := os.ReadFile(rcFile)
			if strings.Contains(string(data), aliasLine) {
				fmt.Printf("Alias already exists in %s\n", rcFile)
				return nil
			}

			f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("cannot write to %s: %w", rcFile, err)
			}
			defer func() { _ = f.Close() }()
			_, _ = fmt.Fprintf(f, "\n# DevDash alias\n%s\n", aliasLine)
			fmt.Printf("Added alias to %s\nRun: source %s\n", rcFile, rcFile)
			return nil
		},
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

func isNPMInstall(exe string) bool {
	return strings.Contains(exe, "node_modules") || strings.Contains(exe, "npm")
}

func isGitInstall(exe string) bool {
	dir := filepath.Dir(filepath.Dir(exe))
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}
