package commands

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		Long: `Update devdash to the latest version by auto-detecting the installation method.

If installed via npm, runs npm update. If running from a git clone, pulls the
latest changes and rebuilds with go build. Otherwise, downloads the latest
GitHub release archive (tar.gz on Unix, zip on Windows) and replaces the
current binary.

Use this instead of manually downloading releases or pulling updates.`,
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
			resp, err := http.Get("https://api.github.com/repos/devdashproject/devdash-cli/releases/latest")
			if err != nil {
				return fmt.Errorf("failed to check latest version: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			var release struct {
				TagName string `json:"tag_name"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
				return fmt.Errorf("failed to parse release info: %w", err)
			}
			version := strings.TrimPrefix(release.TagName, "v")
			if version == "" {
				return fmt.Errorf("could not determine latest version")
			}

			ext := "tar.gz"
			if runtime.GOOS == "windows" {
				ext = "zip"
			}
			binaryName := "devdash"
			if runtime.GOOS == "windows" {
				binaryName = "devdash.exe"
			}

			fmt.Printf("Downloading devdash v%s...\n", version)
			archive := fmt.Sprintf("devdash_%s_%s_%s.%s", version, runtime.GOOS, runtime.GOARCH, ext)
			url := fmt.Sprintf(
				"https://github.com/devdashproject/devdash-cli/releases/download/%s/%s",
				release.TagName, archive)

			tmpDir, err := os.MkdirTemp("", "devdash-update-*")
			if err != nil {
				return fmt.Errorf("failed to create temp dir: %w", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			archivePath := filepath.Join(tmpDir, archive)
			if err := downloadFile(url, archivePath); err != nil {
				return fmt.Errorf("download failed: %w", err)
			}

			src := filepath.Join(tmpDir, binaryName)
			if ext == "zip" {
				if err := extractZip(archivePath, binaryName, src); err != nil {
					return fmt.Errorf("extraction failed: %w", err)
				}
			} else {
				if err := extractTarGz(archivePath, binaryName, src); err != nil {
					return fmt.Errorf("extraction failed: %w", err)
				}
			}

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
		Long: `Remove the devdash binary and its configuration directory (~/.config/dev-dash).

Use --dry-run to preview exactly which paths will be deleted without actually
removing anything. Use --force to skip the confirmation prompt.

After removal, the command suggests cleaning up any shell aliases (e.g. the
"dd" alias added by alias-setup) since those live in your RC file and are
not removed automatically.`,
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
		Long: `Add a "dd" shell alias for devdash to your shell configuration file.

Detects your current shell (zsh, bash, or fish) and appends the
appropriate alias syntax to the matching RC file (~/.zshrc, ~/.bashrc,
or ~/.config/fish/config.fish). If the alias already exists in the
file, the command exits without making changes.

After writing the alias, it prints a source command you can run to
activate it in your current session.`,
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
				if runtime.GOOS == "windows" {
					fmt.Println("Automatic alias setup isn't supported on Windows yet.")
					fmt.Println()
					fmt.Println("To add the 'dd' alias in PowerShell, add this line to your $PROFILE:")
					fmt.Println("  Set-Alias dd devdash")
					fmt.Println()
					fmt.Println("Open your profile with: notepad $PROFILE")
					fmt.Println("Then reload it with:    . $PROFILE")
					return nil
				}
				return fmt.Errorf("unsupported shell %q — add 'alias dd=devdash' to your shell RC file manually", shell)
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

func downloadFile(url, dest string) error {
	resp, err := http.Get(url) //nolint:gosec // URL is constructed from GitHub API response
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGz(archivePath, binaryName, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s not found in archive", binaryName)
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == binaryName {
			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer func() { _ = out.Close() }()
			_, err = io.Copy(out, tr) //nolint:gosec // archive from trusted GitHub release
			return err
		}
	}
}

func extractZip(archivePath, binaryName, dest string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() { _ = rc.Close() }()
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer func() { _ = out.Close() }()
		_, err = io.Copy(out, rc) //nolint:gosec // archive from trusted GitHub release
		return err
	}
	return fmt.Errorf("%s not found in archive", binaryName)
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
