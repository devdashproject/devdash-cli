package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/devdashproject/devdash-cli/internal/api"
)

func TestLoadDefaults(t *testing.T) {
	// Clear env vars
	os.Unsetenv("DD_PROJECT_ID")
	os.Unsetenv("DD_API_URL")
	os.Unsetenv("DD_CONFIG_DIR")
	os.Unsetenv("DD_TOKEN_FILE")
	t.Setenv("DD_CONFIG_DIR", t.TempDir())

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
	t.Setenv("DD_CONFIG_DIR", t.TempDir())

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
	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("repository APIURL = %q, want trusted default %q", cfg.APIURL, DefaultAPIURL)
	}
	if cfg.CloseGate != "commit" {
		t.Errorf("CloseGate = %q, want %q", cfg.CloseGate, "commit")
	}
}

func TestRepositoryAPIURLCannotReceiveToken(t *testing.T) {
	trustedHits, attackerHits := 0, 0
	trusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trustedHits++
		if got := r.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Errorf("trusted endpoint received auth %q", got)
		}
		w.Write([]byte(`{}`))
	}))
	defer trusted.Close()
	attackerHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { attackerHits++ }))
	defer attackerHTTP.Close()
	attackerHTTPS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { attackerHits++ }))
	defer attackerHTTPS.Close()

	for _, attackerURL := range []string{attackerHTTP.URL, attackerHTTPS.URL} {
		t.Run(attackerURL, func(t *testing.T) {
			dir := t.TempDir()
			settingsDir := t.TempDir()
			t.Setenv("DD_CONFIG_DIR", settingsDir)
			t.Setenv("DD_API_URL", "")
			settings, _ := json.Marshal(map[string]string{"api_url": trusted.URL})
			if err := os.WriteFile(filepath.Join(settingsDir, SettingsFileName), settings, 0600); err != nil {
				t.Fatal(err)
			}
			project, _ := json.Marshal(ProjectFile{ProjectID: "repo-project", APIURL: attackerURL})
			if err := os.WriteFile(filepath.Join(dir, ProjectFileName), project, 0644); err != nil {
				t.Fatal(err)
			}
			old, _ := os.Getwd()
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(old)

			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ProjectID != "repo-project" || cfg.APIURL != trusted.URL || cfg.APIURLSource != filepath.Join(settingsDir, SettingsFileName) {
				t.Fatalf("untrusted repo URL affected configuration: %+v", cfg)
			}
			client := api.New(cfg.APIURL, "test-secret", "test")
			if _, err := client.Get("/projects"); err != nil {
				t.Fatal(err)
			}
		})
	}
	if attackerHits != 0 || trustedHits != 2 {
		t.Fatalf("attacker requests = %d, trusted requests = %d", attackerHits, trustedHits)
	}
}

func TestExplicitAPIURLOverride(t *testing.T) {
	t.Setenv("DD_CONFIG_DIR", t.TempDir())
	t.Setenv("DD_API_URL", "https://custom-api.example.com")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIURL != "https://custom-api.example.com" || cfg.APIURLSource != "DD_API_URL environment variable" {
		t.Fatalf("explicit override not distinguished: %+v", cfg)
	}
}

func TestInsecureRemoteOverrideRejected(t *testing.T) {
	t.Setenv("DD_CONFIG_DIR", t.TempDir())
	t.Setenv("DD_API_URL", "http://evil.example")
	if _, err := Load(); err == nil {
		t.Fatal("remote plain HTTP accepted")
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

	// Check permissions (skip on Windows — no Unix permission model)
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(filepath.Join(dir, TokenFileName))
		if info.Mode().Perm() != 0600 {
			t.Errorf("token permissions = %o, want 0600", info.Mode().Perm())
		}
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

func TestLoadTokenFromEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DD_CONFIG_DIR", dir)
	t.Setenv("DD_TOKEN_FILE", filepath.Join(dir, "token"))
	if err := os.WriteFile(filepath.Join(dir, "token"), []byte("dd_fromfile"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(TokenEnvVar, "dd_fromenv")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "dd_fromenv" || cfg.TokenSource != "DEVDASH_TOKEN env var" {
		t.Errorf("env var should win: token=%q source=%q", cfg.Token, cfg.TokenSource)
	}

	t.Setenv(TokenEnvVar, "")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "dd_fromfile" || cfg.TokenSource != filepath.Join(dir, "token") {
		t.Errorf("file fallback: token=%q source=%q", cfg.Token, cfg.TokenSource)
	}
}
