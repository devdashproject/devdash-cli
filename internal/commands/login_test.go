package commands

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devdashproject/devdash-cli/internal/config"
)

// runLogin runs `login` against a server that accepts only goodToken.
func runLogin(t *testing.T, goodToken string, stdin string, args ...string) (string, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+goodToken {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	t.Setenv("DD_TOKEN_FILE", filepath.Join(dir, "token"))
	t.Setenv(config.TokenEnvVar, "")
	deps := &Deps{Cfg: &config.Config{APIURL: server.URL, ConfigDir: dir}}

	rootCmd := NewRootCmd(deps)
	rootCmd.SetArgs(append([]string{"login"}, args...))
	rootCmd.SetIn(strings.NewReader(stdin))
	var errBuf bytes.Buffer
	rootCmd.SetErr(&errBuf)
	rootCmd.SetOut(&errBuf)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := rootCmd.Execute()
	w.Close()
	os.Stdout = oldStdout
	var out bytes.Buffer
	out.ReadFrom(r)
	return out.String() + errBuf.String(), err
}

func savedToken(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(os.Getenv("DD_TOKEN_FILE"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func TestLoginWithTokenFlag(t *testing.T) {
	out, err := runLogin(t, "dd_good123456", "", "--token=dd_good123456")
	if err != nil {
		t.Fatalf("login --token failed: %v\n%s", err, out)
	}
	if got := savedToken(t); got != "dd_good123456" {
		t.Errorf("saved token = %q", got)
	}
	if !strings.Contains(out, "token verified") {
		t.Errorf("expected verification message, got: %s", out)
	}
}

func TestLoginWithTokenStdin(t *testing.T) {
	out, err := runLogin(t, "dd_stdin123456", "dd_stdin123456\n", "--with-token")
	if err != nil {
		t.Fatalf("login --with-token failed: %v\n%s", err, out)
	}
	if got := savedToken(t); got != "dd_stdin123456" {
		t.Errorf("saved token = %q", got)
	}
	if strings.Contains(out, "dd_stdin123456") {
		t.Errorf("token echoed in output: %s", out)
	}
}

func TestLoginWithTokenEmptyStdin(t *testing.T) {
	_, err := runLogin(t, "dd_x", "", "--with-token")
	if err == nil || !strings.Contains(err.Error(), "no token read from stdin") {
		t.Errorf("expected empty-stdin error, got: %v", err)
	}
}

func TestLoginWithRejectedToken(t *testing.T) {
	_, err := runLogin(t, "dd_good123456", "", "--token=dd_bad1234567")
	if err == nil || !strings.Contains(err.Error(), "token rejected") {
		t.Fatalf("expected rejection, got: %v", err)
	}
	if got := savedToken(t); got != "" {
		t.Errorf("rejected token was saved: %q", got)
	}
}

func TestLoginPositionalTokenIsMasked(t *testing.T) {
	out, err := runLogin(t, "dd_x", "", "dd_abcdef0123456789")
	if err == nil {
		t.Fatal("expected error for positional argument")
	}
	if strings.Contains(err.Error()+out, "dd_abcdef0123456789") {
		t.Errorf("token not masked: %v %s", err, out)
	}
	if !strings.Contains(err.Error(), "--token") {
		t.Errorf("error should suggest --token: %v", err)
	}
}

func TestLoginHelpMentionsToken(t *testing.T) {
	out, err := runLogin(t, "dd_x", "", "--help")
	if err != nil {
		t.Fatalf("login --help failed: %v", err)
	}
	for _, want := range []string{"--token", "--with-token", "DEVDASH_TOKEN"} {
		if !strings.Contains(out, want) {
			t.Errorf("login --help missing %q", want)
		}
	}
}
