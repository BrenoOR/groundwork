package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
	"github.com/groundwork-dev/groundwork/internal/registry"
	"github.com/groundwork-dev/groundwork/internal/scanner"
	"github.com/groundwork-dev/groundwork/pkg/model"
	awsplugin "github.com/groundwork-dev/groundwork/plugins/aws"
)

// ScanResult is the structured output of groundwork_scan.
type ScanResult struct {
	Language     string         `json:"language"`
	FilesScanned int            `json:"files_scanned"`
	Services     []string       `json:"services"`
	Resources    []ResourceInfo `json:"resources"`
	Warnings     []string       `json:"warnings"`
}

// ResourceInfo describes a detected cloud resource.
type ResourceInfo struct {
	Provider string `json:"provider"`
	Type     string `json:"type"`
	Name     string `json:"name"`
}

// ScanTool returns the MCP tool definition for groundwork_scan.
func ScanTool() mcp.Tool {
	return mcp.NewTool("groundwork_scan",
		mcp.WithDescription(
			"Scans a project directory and returns the AWS services detected in the source code. "+
				"Also reports the detected programming language, which is useful context for generating "+
				"idiomatic Terraform (e.g. correct Lambda runtime). Read-only operation.",
		),
		mcp.WithString("project_path",
			mcp.Required(),
			mcp.Description("Absolute path to the project directory to scan"),
		),
		mcp.WithString("provider",
			mcp.Description("Cloud provider plugin to use (default: aws)"),
			mcp.DefaultString("aws"),
			mcp.Enum("aws"),
		),
	)
}

// HandleScan returns the handler function for groundwork_scan.
func HandleScan(
	sb *guardrails.Sandbox,
	lim *guardrails.RateLimiter,
	limits guardrails.SizeLimits,
	auditor *guardrails.Auditor,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		if !lim.Allow() {
			auditor.Log(guardrails.AuditEntry{Tool: "groundwork_scan", Result: "blocked", Reason: "rate limit"})
			return mcp.NewToolResultError(guardrails.ErrRateLimited().Error()), nil
		}

		projectPath, err := req.RequireString("project_path")
		if err != nil {
			return mcp.NewToolResultErrorf("project_path is required: %v", err), nil
		}

		if err := sb.CheckRead(projectPath); err != nil {
			auditor.Log(guardrails.AuditEntry{Tool: "groundwork_scan", Result: "blocked", Reason: err.Error()})
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Build registry with the requested provider (only "aws" supported today).
		p, err := awsplugin.New()
		if err != nil {
			return mcp.NewToolResultErrorf("init aws plugin: %v", err), nil
		}
		reg := registry.New()
		if err := reg.Register(p); err != nil {
			return mcp.NewToolResultErrorf("register plugin: %v", err), nil
		}

		// Scan the project.
		sc := scanner.New(projectPath)
		files, err := sc.Scan()
		if err != nil {
			return mcp.NewToolResultErrorf("scan %q: %v", projectPath, err), nil
		}

		var warnings []string

		// Enforce total project size limit before truncating by file count.
		if limits.MaxProjectBytes > 0 {
			var totalBytes int64
			for _, f := range files {
				totalBytes += int64(len(f.Content))
			}
			if totalBytes > limits.MaxProjectBytes {
				auditor.Log(guardrails.AuditEntry{
					Tool:   "groundwork_scan",
					Result: "blocked",
					Reason: fmt.Sprintf("project size %d bytes exceeds max-project-bytes limit of %d", totalBytes, limits.MaxProjectBytes),
				})
				return mcp.NewToolResultErrorf(
					"project size %d bytes exceeds --max-project-bytes limit of %d bytes; use --max-project-bytes to raise the limit",
					totalBytes, limits.MaxProjectBytes,
				), nil
			}
		}

		if len(files) > limits.MaxFiles {
			warnings = append(warnings, fmt.Sprintf(
				"file limit reached (%d max); only the first %d files were analysed",
				limits.MaxFiles, limits.MaxFiles,
			))
			files = files[:limits.MaxFiles]
		}

		// Detect / Map.
		specs, pluginErr := reg.RunAll(files)
		if pluginErr != nil {
			warnings = append(warnings, fmt.Sprintf("plugin warning: %v", pluginErr))
		}

		result := ScanResult{
			Language:     detectLanguage(files),
			FilesScanned: len(files),
			Services:     uniqueNames(specs),
			Resources:    toResourceInfos(specs),
			Warnings:     warnings,
		}

		b, _ := json.MarshalIndent(result, "", "  ")

		auditor.Log(guardrails.AuditEntry{
			Tool:       "groundwork_scan",
			Result:     "ok",
			FilesCount: len(files),
			DurationMs: time.Since(start).Milliseconds(),
		})

		return mcp.NewToolResultText(string(b)), nil
	}
}

// detectLanguage returns the most common non-unknown language found in files.
func detectLanguage(files []model.SourceFile) string {
	counts := map[string]int{}
	for _, f := range files {
		if f.Language != "unknown" && f.Language != "" {
			counts[f.Language]++
		}
	}
	best, bestN := "unknown", 0
	for lang, n := range counts {
		if n > bestN {
			best, bestN = lang, n
		}
	}
	return best
}

// uniqueNames returns service names from specs without duplicates, preserving order.
func uniqueNames(specs []model.ResourceSpec) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range specs {
		if !seen[s.Name] {
			seen[s.Name] = true
			out = append(out, s.Name)
		}
	}
	return out
}

// toResourceInfos converts ResourceSpecs to the JSON-serialisable form.
func toResourceInfos(specs []model.ResourceSpec) []ResourceInfo {
	out := make([]ResourceInfo, len(specs))
	for i, s := range specs {
		out[i] = ResourceInfo{Provider: s.Provider, Type: s.Type, Name: s.Name}
	}
	return out
}
