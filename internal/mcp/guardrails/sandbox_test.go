package guardrails_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
)

func TestSandbox_CheckRead_allowed(t *testing.T) {
	root := t.TempDir()
	sb, err := guardrails.NewSandbox([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if err := sb.CheckRead(root); err != nil {
		t.Errorf("expected read on root to be allowed: %v", err)
	}
	sub := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := sb.CheckRead(sub); err != nil {
		t.Errorf("expected read on subdir to be allowed: %v", err)
	}
}

func TestSandbox_CheckRead_blocked(t *testing.T) {
	root := t.TempDir()
	sb, err := guardrails.NewSandbox([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Dir(root)
	if err := sb.CheckRead(outside); err == nil {
		t.Error("expected read outside allowed root to be blocked")
	}
}

func TestSandbox_CheckRelative_pathTraversal(t *testing.T) {
	root := t.TempDir()
	sb, err := guardrails.NewSandbox([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	traversals := []string{
		"../../etc/passwd",
		"../outside",
		"a/../../outside",
	}
	for _, rel := range traversals {
		if err := sb.CheckRelative(root, rel); err == nil {
			t.Errorf("expected %q to be blocked as path traversal", rel)
		}
	}
}

func TestSandbox_CheckRelative_valid(t *testing.T) {
	root := t.TempDir()
	sb, err := guardrails.NewSandbox([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	valid := []string{
		"modules/s3/main.tf",
		"terragrunt.hcl",
		"a/b/c/d.tf",
	}
	for _, rel := range valid {
		if err := sb.CheckRelative(root, rel); err != nil {
			t.Errorf("expected %q to be valid: %v", rel, err)
		}
	}
}

func TestSandbox_CheckRelative_controlChars(t *testing.T) {
	root := t.TempDir()
	sb, err := guardrails.NewSandbox([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if err := sb.CheckRelative(root, "file\x00name.tf"); err == nil {
		t.Error("expected path with null byte to be blocked")
	}
}

func TestNewSandbox_emptyRoots(t *testing.T) {
	// Empty allowed-roots means nothing is accessible.
	sb, err := guardrails.NewSandbox([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if err := sb.CheckRead("/tmp"); err == nil {
		t.Error("expected any path to be blocked with empty roots")
	}
}
