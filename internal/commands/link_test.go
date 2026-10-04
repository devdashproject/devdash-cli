package commands

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	apiPkg "github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
)

// newLinkRepo creates a git repo with no remote and chdirs into it.
func newLinkRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	t.Chdir(dir)
	return dir
}

// withClosedStdin makes os.Stdin return EOF immediately.
func withClosedStdin(t *testing.T) {
	t.Helper()
	r, w, _ := os.Pipe()
	w.Close()
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old; r.Close() })
}

func linkedProjectID(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, config.ProjectFileName))
	if err != nil {
		return ""
	}
	var pf config.ProjectFile
	json.Unmarshal(data, &pf)
	return pf.ProjectID
}

func TestLinkByProjectIDIsNonInteractive(t *testing.T) {
	dir := newLinkRepo(t)
	withClosedStdin(t)
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("link", "proj-0002")
	if err != nil {
		t.Fatalf("link by prefix failed: %v\n%s", err, out)
	}
	if got := linkedProjectID(t, dir); got != "proj-0002-0000-0000-000000000002" {
		t.Errorf("linked %q, want proj-0002...", got)
	}
}

func TestLinkByNameAndProjectFlag(t *testing.T) {
	dir := newLinkRepo(t)
	withClosedStdin(t)
	run := newTestEnv(t, apiPkg.SampleBeads())
	if out, err := run("--project=project-one", "link"); err != nil {
		t.Fatalf("link via --project name failed: %v\n%s", err, out)
	}
	if got := linkedProjectID(t, dir); got != "proj-0001-0000-0000-000000000001" {
		t.Errorf("linked %q, want proj-0001...", got)
	}
}

func TestLinkWithoutInputFailsInsteadOfPickingFirst(t *testing.T) {
	dir := newLinkRepo(t)
	withClosedStdin(t)
	run := newTestEnv(t, apiPkg.SampleBeads())
	_, err := run("link")
	if err == nil || !strings.Contains(err.Error(), "devdash link <project") {
		t.Fatalf("expected no-selection error with hint, got: %v", err)
	}
	if got := linkedProjectID(t, dir); got != "" {
		t.Errorf("must not write .devdash without a choice, linked %q", got)
	}
}

func TestLinkHereWritesCurrentDir(t *testing.T) {
	dir := newLinkRepo(t)
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	t.Chdir(sub)
	withClosedStdin(t)
	run := newTestEnv(t, apiPkg.SampleBeads())
	if out, err := run("link", "test-project-id", "--here"); err != nil {
		t.Fatalf("link --here failed: %v\n%s", err, out)
	}
	if linkedProjectID(t, sub) != "test-project-id" || linkedProjectID(t, dir) != "" {
		t.Errorf("--here should write only %s/.devdash", sub)
	}
}

func TestLinkNoRemoteDoesNotInventOne(t *testing.T) {
	dir := newLinkRepo(t)
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	t.Chdir(sub)
	r, w, _ := os.Pipe()
	w.Write([]byte("1\n1\n"))
	w.Close()
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
	run := newTestEnv(t, apiPkg.SampleBeads())
	out, err := run("link")
	if err != nil {
		t.Fatalf("interactive link failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "github.com/") {
		t.Errorf("should not show a GitHub remote for a repo without one:\n%s", out)
	}
	if !strings.Contains(out, "no GitHub remote") {
		t.Errorf("should say there is no GitHub remote:\n%s", out)
	}
}

func TestSamePathResolvesSymlinks(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks unsupported")
	}
	if !samePath(real, link) {
		t.Errorf("samePath(%q, %q) should be true", real, link)
	}
}

func TestReadChoiceFrom(t *testing.T) {
	if c, err := readChoiceFrom(bufio.NewReader(strings.NewReader("x\n2\n")), 2); err != nil || c != 2 {
		t.Errorf("got %d, %v; want 2 after one invalid line", c, err)
	}
	if _, err := readChoiceFrom(bufio.NewReader(strings.NewReader("")), 2); err == nil {
		t.Error("EOF should be an error, not option 1")
	}
}
