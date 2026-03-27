package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear env vars
	os.Unsetenv("DD_PROJECT_ID")
	os.Unsetenv("DD_API_URL")
	os.Unsetenv("DD_CONFIG_DIR")
	os.Unsetenv("DD_TOKEN_FILE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, DefaultAPIURL)
	}
	if cfg.FrontendURL != DefaultFrontendURL {
		t.Errorf("FrontendURL = %q, want %q", cfg.FrontendURL, DefaultFrontendURL)
	}
	if cfg.CloseGate != DefaultCloseGate {
		t.Errorf("CloseGate = %q, want %q", cfg.CloseGate, DefaultCloseGate)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("DD_PROJECT_ID", "test-project-id")
	t.Setenv("DD_API_URL", "http://localhost:3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.ProjectID != "test-project-id" {
		t.Errorf("ProjectID = %q, want %q", cfg.ProjectID, "test-project-id")
	}
	if cfg.APIURL != "http://localhost:3000" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://localhost:3000")
	}
}

func TestLoadProjectFile(t *testing.T) {
	dir := t.TempDir()

	pf := ProjectFile{
		ProjectID:   "from-file",
		APIURL:      "https://custom-api.example.com",
		FrontendURL: "https://custom-frontend.example.com",
		CloseGate:   "commit",
	}
	data, _ := json.Marshal(pf)
	os.WriteFile(filepath.Join(dir, ProjectFileName), data, 0644)

	// Change to temp dir so findProjectFile finds it
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.Unsetenv("DD_PROJECT_ID")
	os.Unsetenv("DD_API_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.ProjectID != "from-file" {
		t.Errorf("ProjectID = %q, want %q", cfg.ProjectID, "from-file")
	}
	if cfg.APIURL != "https://custom-api.example.com" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "https://custom-api.example.com")
	}
	if cfg.CloseGate != "commit" {
		t.Errorf("CloseGate = %q, want %q", cfg.CloseGate, "commit")
	}
}

func TestEnvOverridesProjectFile(t *testing.T) {
	dir := t.TempDir()
	pf := ProjectFile{ProjectID: "from-file"}
	data, _ := json.Marshal(pf)
	os.WriteFile(filepath.Join(dir, ProjectFileName), data, 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	t.Setenv("DD_PROJECT_ID", "from-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.ProjectID != "from-env" {
		t.Errorf("ProjectID = %q, want %q (env should override file)", cfg.ProjectID, "from-env")
	}
}

func TestSaveToken(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ConfigDir: dir}

	if err := cfg.SaveToken("test-token-123"); err != nil {
		t.Fatalf("SaveToken() failed: %v", err)
	}

	// Read it back
	data, err := os.ReadFile(filepath.Join(dir, TokenFileName))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "test-token-123" {
		t.Errorf("token = %q, want %q", string(data), "test-token-123")
	}

	// Check permissions
	info, _ := os.Stat(filepath.Join(dir, TokenFileName))
	if info.Mode().Perm() != 0600 {
		t.Errorf("token permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestRequireToken(t *testing.T) {
	cfg := &Config{Token: ""}
	if _, err := cfg.RequireToken(); err == nil {
		t.Error("RequireToken() should fail with empty token")
	}

	cfg.Token = "valid"
	token, err := cfg.RequireToken()
	if err != nil {
		t.Errorf("RequireToken() failed: %v", err)
	}
	if token != "valid" {
		t.Errorf("token = %q, want %q", token, "valid")
	}
}

func TestRequireProjectID(t *testing.T) {
	cfg := &Config{ProjectID: ""}
	if _, err := cfg.RequireProjectID(); err == nil {
		t.Error("RequireProjectID() should fail with empty project ID")
	}

	cfg.ProjectID = "proj-123"
	pid, err := cfg.RequireProjectID()
	if err != nil {
		t.Errorf("RequireProjectID() failed: %v", err)
	}
	if pid != "proj-123" {
		t.Errorf("projectID = %q, want %q", pid, "proj-123")
	}
}
