package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteManagedInstructions_NewFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "CLAUDE.md")

	if err := writeManagedInstructions(target, "hello body", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if !strings.Contains(s, managedBlockStart) || !strings.Contains(s, managedBlockEnd) {
		t.Fatalf("missing sentinels: %q", s)
	}
	if !strings.Contains(s, "hello body") {
		t.Fatalf("missing body: %q", s)
	}
}

func TestWriteManagedInstructions_ForceIdempotentPreservesUserContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "CLAUDE.md")

	const userBefore = "# My Project\n\nSome user-written notes.\n\n"
	const userAfter = "\n\n## Other Section\n\nMore user content.\n"

	// Seed with user content + managed block + trailing user content.
	initial := userBefore + managedBlockStart + "\n\nOLD BODY\n" + managedBlockEnd + userAfter
	if err := os.WriteFile(target, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	// First forced run: replace block.
	if err := writeManagedInstructions(target, "NEW BODY", true); err != nil {
		t.Fatalf("first force: %v", err)
	}

	got, _ := os.ReadFile(target)
	s := string(got)
	if !strings.Contains(s, userBefore) {
		t.Errorf("user content before block lost: %q", s)
	}
	if !strings.Contains(s, userAfter) {
		t.Errorf("user content after block lost: %q", s)
	}
	if strings.Contains(s, "OLD BODY") {
		t.Errorf("old body not replaced: %q", s)
	}
	if !strings.Contains(s, "NEW BODY") {
		t.Errorf("new body not written: %q", s)
	}
	if strings.Count(s, managedBlockStart) != 1 || strings.Count(s, managedBlockEnd) != 1 {
		t.Errorf("sentinel count drift: %q", s)
	}

	// Second forced run with same body: byte-for-byte identical.
	first := s
	if err := writeManagedInstructions(target, "NEW BODY", true); err != nil {
		t.Fatalf("second force: %v", err)
	}
	got2, _ := os.ReadFile(target)
	if string(got2) != first {
		t.Errorf("second force not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, string(got2))
	}
}

func TestWriteManagedInstructions_NoForceSkipsWhenMarkersPresent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "CLAUDE.md")

	initial := "user stuff\n" + managedBlockStart + "\n\nOLD BODY\n" + managedBlockEnd + "\nmore user\n"
	if err := os.WriteFile(target, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeManagedInstructions(target, "NEW BODY", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(target)
	if string(got) != initial {
		t.Errorf("file modified without --force:\n--- want ---\n%s\n--- got ---\n%s", initial, string(got))
	}
}

func TestWriteManagedInstructions_NoMarkersAppends(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "CLAUDE.md")

	// File exists but does not mention devdash and has no markers.
	initial := "# Existing project\n\nSome notes.\n"
	if err := os.WriteFile(target, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeManagedInstructions(target, "APPENDED BODY", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(target)
	s := string(got)
	if !strings.HasPrefix(s, initial) {
		t.Errorf("prefix changed: %q", s)
	}
	if !strings.Contains(s, "APPENDED BODY") {
		t.Errorf("body not appended: %q", s)
	}
	if !strings.Contains(s, managedBlockStart) || !strings.Contains(s, managedBlockEnd) {
		t.Errorf("sentinels missing on append: %q", s)
	}
}
