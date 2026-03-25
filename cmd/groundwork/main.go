package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/groundwork-dev/groundwork/internal/engine"
	"github.com/groundwork-dev/groundwork/internal/registry"
	"github.com/groundwork-dev/groundwork/pkg/model"
	awsplugin "github.com/groundwork-dev/groundwork/plugins/aws"
)

const usage = `groundwork — detect AWS SDK usage and generate Terragrunt scaffolding.

Usage:
  groundwork [flags]

Flags:
  --input              Path to the project to analyse          (default: ".")
  --output             Directory where Terragrunt files will be written (default: "./output")
  --provider           Cloud provider plugin to use           (default: "aws")
  --state-bucket       S3 bucket name for Terraform state     (required)
  --state-region       AWS region of the state bucket         (default: "us-east-1")
  --state-lock-table   DynamoDB table name for state locking  (default: "<bucket>-lock")
  --state-encrypt      Enable SSE-S3 encryption on state      (default: true)
  --dry-run            Print what would be generated without writing any files

Examples:
  groundwork --input ./my-project --output ./infra --state-bucket my-tf-state
  groundwork --input ./my-project --dry-run --state-bucket my-tf-state
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("groundwork", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	input := fs.String("input", ".", "path to the project to analyse")
	output := fs.String("output", "./output", "directory for Terragrunt output")
	provider := fs.String("provider", "aws", "cloud provider plugin")
	dryRun := fs.Bool("dry-run", false, "print what would be generated without writing files")

	stateBucket := fs.String("state-bucket", "", "S3 bucket for Terraform state (required)")
	stateRegion := fs.String("state-region", "us-east-1", "AWS region of the state bucket")
	stateLockTable := fs.String("state-lock-table", "", "DynamoDB table for state locking (default: <bucket>-lock)")
	stateEncrypt := fs.Bool("state-encrypt", true, "enable SSE-S3 encryption on state files")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *stateBucket == "" {
		return fmt.Errorf("--state-bucket is required")
	}

	lockTable := *stateLockTable
	if lockTable == "" {
		lockTable = *stateBucket + "-lock"
	}

	reg, err := buildRegistry(*provider)
	if err != nil {
		return err
	}

	cfg := engine.Config{
		InputDir:  *input,
		OutputDir: *output,
		DryRun:    *dryRun,
		Backend: model.BackendConfig{
			Bucket:    *stateBucket,
			Region:    *stateRegion,
			LockTable: lockTable,
			Encrypt:   *stateEncrypt,
		},
	}

	return engine.New(cfg, reg).Run()
}

// buildRegistry creates a registry with the plugins for the requested provider.
func buildRegistry(provider string) (*registry.Registry, error) {
	reg := registry.New()

	switch provider {
	case "aws":
		p, err := awsplugin.New()
		if err != nil {
			return nil, fmt.Errorf("init aws plugin: %w", err)
		}
		if err := reg.Register(p); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown provider %q — supported: aws", provider)
	}

	return reg, nil
}
