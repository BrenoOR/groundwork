package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/groundwork-dev/groundwork/internal/engine"
	"github.com/groundwork-dev/groundwork/internal/registry"
	"github.com/groundwork-dev/groundwork/pkg/model"
	awsplugin "github.com/groundwork-dev/groundwork/plugins/aws"
)

var testBackend = model.BackendConfig{
	Bucket:    "test-tf-state",
	Region:    "us-east-1",
	LockTable: "test-tf-state-lock",
	Encrypt:   true,
}

func newRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	reg := registry.New()
	p, err := awsplugin.New()
	if err != nil {
		t.Fatalf("awsplugin.New: %v", err)
	}
	if err := reg.Register(p); err != nil {
		t.Fatalf("registry.Register: %v", err)
	}
	return reg
}

func scannerFixture(t *testing.T, name string) string {
	t.Helper()
	// Reuse the scanner testdata fixtures.
	path, err := filepath.Abs(filepath.Join("..", "scanner", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEngine_run_goProject(t *testing.T) {
	out := t.TempDir()
	cfg := engine.Config{
		InputDir:  scannerFixture(t, "go_project"),
		OutputDir: out,
		Backend:   testBackend,
	}

	e := engine.New(cfg, newRegistry(t))
	if err := e.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// s3 is imported in the go_project fixture.
	s3Module := filepath.Join(out, "modules", "s3", "terragrunt.hcl")
	if _, err := os.Stat(s3Module); err != nil {
		t.Errorf("expected s3 module to be generated, got: %v", err)
	}
}

func TestEngine_run_noAWS(t *testing.T) {
	// node_project fixture has no AWS SDK usage.
	input := t.TempDir()
	if err := os.WriteFile(filepath.Join(input, "index.js"), []byte(`console.log("hello")`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(input, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := t.TempDir()
	cfg := engine.Config{InputDir: input, OutputDir: out, Backend: testBackend}

	e := engine.New(cfg, newRegistry(t))
	if err := e.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Nothing should be written when no AWS services are detected.
	entries, _ := os.ReadDir(filepath.Join(out, "modules"))
	if len(entries) != 0 {
		t.Errorf("expected no modules, got %d", len(entries))
	}
}

func TestEngine_run_dryRun(t *testing.T) {
	out := t.TempDir()
	cfg := engine.Config{
		InputDir:  scannerFixture(t, "go_project"),
		OutputDir: out,
		DryRun:    true,
		Backend:   testBackend,
	}

	e := engine.New(cfg, newRegistry(t))
	if err := e.Run(); err != nil {
		t.Fatalf("Run (dry-run): %v", err)
	}

	// Dry-run must not write any files.
	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		t.Errorf("dry-run wrote files: %v", entries)
	}
}

func TestEngine_run_invalidInput(t *testing.T) {
	cfg := engine.Config{
		InputDir:  "/nonexistent/path",
		OutputDir: t.TempDir(),
		Backend:   testBackend,
	}
	e := engine.New(cfg, newRegistry(t))
	if err := e.Run(); err == nil {
		t.Fatal("expected error for invalid input path, got nil")
	}
}