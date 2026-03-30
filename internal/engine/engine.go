package engine

import (
	"fmt"
	"io"
	"os"

	"github.com/groundwork-dev/groundwork/internal/generator/terragrunt"
	"github.com/groundwork-dev/groundwork/internal/registry"
	"github.com/groundwork-dev/groundwork/internal/scanner"
	"github.com/groundwork-dev/groundwork/pkg/model"
)

// Config holds the parameters for a single engine run.
type Config struct {
	InputDir  string
	OutputDir string
	DryRun    bool
	Backend   model.BackendConfig
}

// Engine orchestrates the full pipeline: scan → detect → map → generate.
type Engine struct {
	cfg      Config
	scanner  *scanner.Scanner
	registry *registry.Registry
	gen      *terragrunt.Generator
	stderr   io.Writer
}

// New builds an Engine from the given config and pre-populated registry.
// The caller is responsible for registering plugins before calling Run.
func New(cfg Config, reg *registry.Registry) *Engine {
	return &Engine{
		cfg:      cfg,
		scanner:  scanner.New(cfg.InputDir),
		registry: reg,
		gen:      terragrunt.New(cfg.OutputDir),
		stderr:   os.Stderr,
	}
}

// Run executes the full pipeline.
//
//  1. Scan: walk InputDir and collect source files.
//  2. Detect + Map: run all registered plugins and collect ResourceSpecs.
//  3. Generate: render Terragrunt files into OutputDir (skipped on DryRun).
func (e *Engine) Run() error {
	fmt.Fprintf(e.stderr, "groundwork: input=%q output=%q dry-run=%v\n", e.cfg.InputDir, e.cfg.OutputDir, e.cfg.DryRun)
	fmt.Fprintf(e.stderr, "groundwork: state bucket=%q region=%q lock-table=%q encrypt=%v\n",
		e.cfg.Backend.Bucket, e.cfg.Backend.Region, e.cfg.Backend.LockTable, e.cfg.Backend.Encrypt)

	// 1. Scan
	files, err := e.scanner.Scan()
	if err != nil {
		return fmt.Errorf("engine: scan %q: %w", e.cfg.InputDir, err)
	}
	fmt.Fprintf(e.stderr, "groundwork: scanned %d file(s) from %q\n", len(files), e.cfg.InputDir)

	// 2. Detect + Map
	specs, err := e.registry.RunAll(files)
	if err != nil {
		// RunAll uses errors.Join — partial results are still valid.
		fmt.Fprintf(e.stderr, "groundwork: plugin warning: %v\n", err)
	}
	fmt.Fprintf(e.stderr, "groundwork: %d resource spec(s) collected\n", len(specs))

	if len(specs) == 0 {
		fmt.Fprintln(e.stderr, "groundwork: no AWS resources detected — nothing to generate")
		return nil
	}

	// 3. Generate (or dry-run preview)
	if e.cfg.DryRun {
		return e.printDryRun(specs)
	}
	for _, s := range specs {
		fmt.Fprintf(e.stderr, "groundwork: generating module %q (%s)\n", s.Name, s.Type)
	}
	if err := e.gen.Generate(specs, e.cfg.Backend); err != nil {
		return fmt.Errorf("engine: generate: %w", err)
	}
	fmt.Fprintf(e.stderr, "groundwork: output written to %q\n", e.cfg.OutputDir)
	return nil
}

// printDryRun writes a human-readable summary of what would be generated.
func (e *Engine) printDryRun(specs []model.ResourceSpec) error {
	fmt.Fprintln(e.stderr, "groundwork: dry-run — no files written")
	fmt.Fprintf(e.stderr, "  %s/terragrunt.hcl  (root remote_state)\n", e.cfg.OutputDir)
	for _, s := range specs {
		fmt.Fprintf(e.stderr, "  %s/modules/%s/terragrunt.hcl  (%s)\n",
			e.cfg.OutputDir, s.Name, s.Type)
	}
	return nil
}