package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
)

// FileSpec represents a single file to be written.
type FileSpec struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// writeRequest is the parsed input for groundwork_write_files.
type writeRequest struct {
	BasePath  string     `json:"base_path"`
	Files     []FileSpec `json:"files"`
	Confirmed bool       `json:"confirmed"`
}

// WriteFilesTool returns the MCP tool definition for groundwork_write_files.
func WriteFilesTool() mcp.Tool {
	return mcp.NewTool("groundwork_write_files",
		mcp.WithDescription(
			"Writes Terraform/Terragrunt files to disk. "+
				"The AI provides complete file contents; this tool persists them with safety guardrails. "+
				"Requires confirmed=true to actually write — call without confirmed first to validate inputs.",
		),
		mcp.WithString("base_path",
			mcp.Required(),
			mcp.Description("Absolute path of the output directory (e.g. /abs/myapp/infra)"),
		),
		mcp.WithArray("files",
			mcp.Required(),
			mcp.Description("List of files to write, each with a relative path and full content"),
			func(s map[string]any) {
				s["items"] = map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path":    map[string]any{"type": "string", "description": "Relative path within base_path (e.g. modules/s3/main.tf)"},
						"content": map[string]any{"type": "string", "description": "Complete file content"},
					},
					"required": []string{"path", "content"},
				}
				s["minItems"] = 1
				s["maxItems"] = 50
			},
		),
		mcp.WithBoolean("confirmed",
			mcp.Description("Set to true to actually write files. Omit or set false to validate only."),
			mcp.DefaultBool(false),
		),
	)
}

// HandleWriteFiles returns the handler function for groundwork_write_files.
func HandleWriteFiles(
	sb *guardrails.Sandbox,
	lim *guardrails.RateLimiter,
	cv *guardrails.ContentValidator,
	auditor *guardrails.Auditor,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		if !lim.Allow() {
			auditor.Log(guardrails.AuditEntry{Tool: "groundwork_write_files", Result: "blocked", Reason: "rate limit"})
			return mcp.NewToolResultError(guardrails.ErrRateLimited().Error()), nil
		}

		var input writeRequest
		if err := req.BindArguments(&input); err != nil {
			return mcp.NewToolResultErrorf("invalid input: %v", err), nil
		}

		if input.BasePath == "" {
			return mcp.NewToolResultError("base_path is required"), nil
		}
		if len(input.Files) == 0 {
			return mcp.NewToolResultError("files must not be empty"), nil
		}

		// Sandbox check on base_path.
		if err := sb.CheckWrite(input.BasePath); err != nil {
			auditor.Log(guardrails.AuditEntry{Tool: "groundwork_write_files", Result: "blocked", Reason: err.Error()})
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Validate all files before touching the filesystem.
		for _, f := range input.Files {
			if err := sb.CheckRelative(input.BasePath, f.Path); err != nil {
				auditor.Log(guardrails.AuditEntry{Tool: "groundwork_write_files", Result: "blocked", Reason: err.Error()})
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := cv.Validate(f.Path, f.Content); err != nil {
				auditor.Log(guardrails.AuditEntry{Tool: "groundwork_write_files", Result: "blocked", Reason: err.Error()})
				return mcp.NewToolResultError(err.Error()), nil
			}
		}

		// Require explicit confirmation before writing.
		if !input.Confirmed {
			return mcp.NewToolResultError(
				"write operations require confirmed=true. " +
					"Review the files listed above, then call groundwork_write_files again with confirmed=true.",
			), nil
		}

		// Write files atomically — all-or-nothing on first error.
		var written []string
		for _, f := range input.Files {
			dest := filepath.Join(input.BasePath, filepath.FromSlash(f.Path))
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return mcp.NewToolResultErrorf("create directory for %q: %v", f.Path, err), nil
			}
			if err := os.WriteFile(dest, []byte(f.Content), 0o644); err != nil {
				return mcp.NewToolResultErrorf("write %q: %v", f.Path, err), nil
			}
			written = append(written, f.Path)
		}

		result := map[string]any{"files_written": written}
		b, _ := json.MarshalIndent(result, "", "  ")

		auditor.Log(guardrails.AuditEntry{
			Tool:       "groundwork_write_files",
			Result:     "ok",
			FilesCount: len(written),
			DurationMs: time.Since(start).Milliseconds(),
		})

		return mcp.NewToolResultText(string(b)), nil
	}
}
