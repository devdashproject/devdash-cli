package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultAPIURL      = "https://dev-dash-server-production.up.railway.app"
	DefaultFrontendURL = "https://dev-dash-blue.vercel.app"
	DefaultCloseGate   = "push"
	DefaultConfigDir   = ".config/dev-dash"
	TokenFileName      = "token"
	ProjectFileName    = ".devdash"
)

// ProjectFile represents the .devdash file in a repo root.
type ProjectFile struct {
	ProjectID   string `json:"project_id"`
	APIURL      string `json:"api_url,omitempty"`
	FrontendURL string `json:"frontend_url,omitempty"`
	CloseGate   string `json:"close_gate,omitempty"`
}

// Config holds resolved configuration from all sources.
type Config struct {
	ProjectID   string
	APIURL      string
	FrontendURL string
	CloseGate   string
	Token       string
	ConfigDir   string
}

// Load resolves configuration from env vars, .devdash file, and defaults.
func Load() (*Config, error) {
	cfg := &Config{
		APIURL:      DefaultAPIURL,
		FrontendURL: DefaultFrontendURL,
		CloseGate:   DefaultCloseGate,
	}

	// Config directory
	cfg.ConfigDir = os.Getenv("DD_CONFIG_DIR")
	if cfg.ConfigDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		cfg.ConfigDir = filepath.Join(home, DefaultConfigDir)
	}

	// Load .devdash project file (walk up from cwd to find it)
	if pf, err := findProjectFile(); err == nil {
		if pf.ProjectID != "" {
			cfg.ProjectID = pf.ProjectID
		}
		if pf.APIURL != "" {
			cfg.APIURL = pf.APIURL
		}
		if pf.FrontendURL != "" {
			cfg.FrontendURL = pf.FrontendURL
		}
		if pf.CloseGate != "" {
			cfg.CloseGate = pf.CloseGate
		}
	}

	// Environment variable overrides (highest priority)
	if v := os.Getenv("DD_PROJECT_ID"); v != "" {
		cfg.ProjectID = v
	}
	if v := os.Getenv("DD_API_URL"); v != "" {
		cfg.APIURL = v
	}

	// Load token
	token, err := loadToken(cfg.ConfigDir)
	if err == nil {
		cfg.Token = token
	}

	return cfg, nil
}

// TokenFilePath returns the path to the token file.
func (c *Config) TokenFilePath() string {
	if v := os.Getenv("DD_TOKEN_FILE"); v != "" {
		return v
	}
	return filepath.Join(c.ConfigDir, TokenFileName)
}

// SaveToken writes the token to the config directory with secure permissions.
func (c *Config) SaveToken(token string) error {
	if err := os.MkdirAll(c.ConfigDir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	path := c.TokenFilePath()
	return os.WriteFile(path, []byte(token), 0600)
}

// RequireToken returns the token or an error if not authenticated.
func (c *Config) RequireToken() (string, error) {
	if c.Token == "" {
		return "", fmt.Errorf("not authenticated — run 'devdash login' first")
	}
	return c.Token, nil
}

// RequireProjectID returns the project ID or an error if not configured.
func (c *Config) RequireProjectID() (string, error) {
	if c.ProjectID == "" {
		return "", fmt.Errorf("no project configured — run 'devdash link' or set DD_PROJECT_ID")
	}
	return c.ProjectID, nil
}

func loadToken(configDir string) (string, error) {
	path := os.Getenv("DD_TOKEN_FILE")
	if path == "" {
		path = filepath.Join(configDir, TokenFileName)
	}

	// Check permissions
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		// Fix permissions silently
		_ = os.Chmod(path, 0600)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func findProjectFile() (*ProjectFile, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		path := filepath.Join(dir, ProjectFileName)
		data, err := os.ReadFile(path)
		if err == nil {
			var pf ProjectFile
			if err := json.Unmarshal(data, &pf); err != nil {
				return nil, fmt.Errorf("invalid %s: %w", path, err)
			}
			return &pf, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, fmt.Errorf("%s not found", ProjectFileName)
}
