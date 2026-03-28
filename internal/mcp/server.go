// Package mcp implements the groundwork MCP server.
//
// The server exposes two tools:
//   - groundwork_scan: scans a project and returns detected AWS services
//   - groundwork_write_files: writes AI-generated Terraform/Terragrunt files to disk
//
// And two resources:
//   - groundwork://services: canonical service → Terraform resource type mapping
//   - groundwork://template/{service}: reference .tf template for a given service
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
	"github.com/groundwork-dev/groundwork/internal/mcp/tools"
)

// Config holds the server configuration derived from CLI flags.
type Config struct {
	AllowedRoots          []string
	ReadOnly              bool
	MaxFiles              int
	MaxProjectBytes       int64
	MaxFileBytes          int64
	RatePerSec            float64
	Burst                 float64
	AllowExecProvisioners bool
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{
		MaxFiles:        guardrails.DefaultMaxFiles,
		MaxProjectBytes: guardrails.DefaultMaxProjectBytes,
		MaxFileBytes:    guardrails.DefaultMaxFileBytes,
		RatePerSec:      guardrails.DefaultRatePerSec,
		Burst:           guardrails.DefaultBurst,
	}
}

// New builds and returns a configured MCPServer.
// Returns an error if any AllowedRoot cannot be resolved.
func New(cfg Config) (*server.MCPServer, error) {
	sb, err := guardrails.NewSandbox(cfg.AllowedRoots)
	if err != nil {
		return nil, fmt.Errorf("mcp server: %w", err)
	}

	limits := guardrails.SizeLimits{
		MaxFiles:        cfg.MaxFiles,
		MaxProjectBytes: cfg.MaxProjectBytes,
		MaxFileBytes:    cfg.MaxFileBytes,
	}

	lim := guardrails.NewRateLimiter(cfg.RatePerSec, cfg.Burst)
	cv := guardrails.NewContentValidator(cfg.AllowExecProvisioners, cfg.MaxFileBytes)
	auditor := guardrails.NewAuditor(nil) // writes to stderr

	s := server.NewMCPServer(
		"groundwork-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	// --- Tools ---

	s.AddTool(tools.ScanTool(), tools.HandleScan(sb, lim, limits, auditor))

	if !cfg.ReadOnly {
		s.AddTool(tools.WriteFilesTool(), tools.HandleWriteFiles(sb, lim, cv, auditor))
	}

	// --- Resources ---

	addResources(s)

	return s, nil
}

// addResources registers the read-only MCP resources.
func addResources(s *server.MCPServer) {
	// groundwork://services — canonical service → resource type map.
	s.AddResource(
		mcp.NewResource(
			"groundwork://services",
			"AWS service → Terraform resource type mapping",
			mcp.WithMIMEType("application/json"),
		),
		handleServicesResource,
	)

	// groundwork://template/{service} — reference .tf template for each service.
	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"groundwork://template/{service}",
			"Reference Terraform template for a given AWS service",
			mcp.WithTemplateDescription(
				"Returns the reference .tf template for a service (e.g. s3, dynamodb, lambda). "+
					"The AI may use this as a starting point or ignore it entirely.",
			),
		),
		handleTemplateResource,
	)
}

// handleServicesResource returns the service → resource type mapping as JSON.
func handleServicesResource(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	const body = `{
  "s3":             "aws_s3_bucket",
  "dynamodb":       "aws_dynamodb_table",
  "sqs":            "aws_sqs_queue",
  "sns":            "aws_sns_topic",
  "lambda":         "aws_lambda_function",
  "rds":            "aws_db_instance",
  "secretsmanager": "aws_secretsmanager_secret"
}`
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "groundwork://services",
			MIMEType: "application/json",
			Text:     body,
		},
	}, nil
}

// handleTemplateResource reads the embedded .tf template for the requested service.
func handleTemplateResource(_ context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	raw := req.Params.Arguments["service"]
	service, _ := raw.(string)
	if service == "" {
		return nil, fmt.Errorf("service path parameter is required")
	}

	content, err := loadTemplateContent(service)
	if err != nil {
		return nil, err
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/plain",
			Text:     content,
		},
	}, nil
}
