package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	mcpserver "github.com/groundwork-dev/groundwork/internal/mcp"
)

const usage = `groundwork-mcp — MCP server for AI-driven Terraform/Terragrunt generation.

The AI scans your project, generates Terraform code from its own knowledge,
and writes the files through this server's guardrails.

Usage:
  groundwork-mcp [flags]

Flags:
  --allowed-root          Directory the server is allowed to read/write (repeatable, required)
  --readonly              Disable write operations (groundwork_write_files is not registered)
  --transport             Transport mode: stdio (default) or sse
  --port                  Port for SSE transport (default: 8080)
  --max-files             Maximum number of files to scan per project (default: 10000)
  --max-project-bytes     Maximum total project size in bytes (default: 104857600)
  --max-file-bytes        Maximum size of a single written file in bytes (default: 524288)
  --rate-limit            Maximum tool calls per second (default: 10)
  --allow-exec-provisioners  Allow local-exec/remote-exec provisioner blocks in written files

Examples:
  # stdio mode for Claude Desktop / Claude Code
  groundwork-mcp --allowed-root /home/user/projects

  # SSE mode with extra restrictions
  groundwork-mcp --transport sse --port 8080 --allowed-root /workspace --readonly

  # Allow exec provisioners (opt-in)
  groundwork-mcp --allowed-root /home/user/projects --allow-exec-provisioners
`

// multiFlag allows a flag to be specified multiple times.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ", ") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("groundwork-mcp", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	var allowedRoots multiFlag
	fs.Var(&allowedRoots, "allowed-root", "directory the server may read/write (repeatable)")

	readOnly := fs.Bool("readonly", false, "disable write operations")
	transport := fs.String("transport", "stdio", "transport mode: stdio or sse")
	port := fs.Int("port", 8080, "port for SSE transport")
	maxFiles := fs.Int("max-files", 10_000, "maximum files to scan per project")
	maxProjectBytes := fs.Int64("max-project-bytes", 100*1024*1024, "maximum total project size in bytes")
	maxFileBytes := fs.Int64("max-file-bytes", 512*1024, "maximum single file size in bytes")
	rateLimit := fs.Float64("rate-limit", 10.0, "maximum tool calls per second")
	allowExec := fs.Bool("allow-exec-provisioners", false, "allow local-exec/remote-exec provisioner blocks")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if len(allowedRoots) == 0 {
		return fmt.Errorf("--allowed-root is required (specify at least one directory)")
	}

	cfg := mcpserver.Config{
		AllowedRoots:          []string(allowedRoots),
		ReadOnly:              *readOnly,
		MaxFiles:              *maxFiles,
		MaxProjectBytes:       *maxProjectBytes,
		MaxFileBytes:          *maxFileBytes,
		RatePerSec:            *rateLimit,
		Burst:                 *rateLimit * 2,
		AllowExecProvisioners: *allowExec,
	}

	s, err := mcpserver.New(cfg)
	if err != nil {
		return fmt.Errorf("init server: %w", err)
	}

	switch *transport {
	case "stdio":
		return server.ServeStdio(s)
	case "sse":
		sse := server.NewSSEServer(s, server.WithBaseURL(fmt.Sprintf("http://localhost:%d", *port)))
		fmt.Fprintf(os.Stderr, "groundwork-mcp: listening on :%d (SSE)\n", *port)
		return sse.Start(fmt.Sprintf(":%d", *port))
	default:
		return fmt.Errorf("unknown transport %q — supported: stdio, sse", *transport)
	}
}
