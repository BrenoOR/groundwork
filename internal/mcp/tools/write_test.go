package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
	"github.com/groundwork-dev/groundwork/internal/mcp/tools"
)

func makeWriteHandlerWithRoot(t *testing.T) (func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), string) {
	t.Helper()
	root := t.TempDir()
	sb, _ := guardrails.NewSandbox([]string{root})
	lim := guardrails.NewRateLimiter(100, 100)
	cv := guardrails.NewContentValidator(false, 512*1024)
	auditor := guardrails.NewAuditor(os.Stderr)
	return tools.HandleWriteFiles(sb, lim, cv, auditor), root
}

func callWriteTool(t *testing.T, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	return result
}

func isError(result *mcp.CallToolResult) bool {
	return result != nil && result.IsError
}

func TestWriteFiles_requiresConfirmed(t *testing.T) {
	handler, root := makeWriteHandlerWithRoot(t)
	result := callWriteTool(t, handler, map[string]any{
		"base_path": root,
		"files":     []any{map[string]any{"path": "main.tf", "content": "# ok"}},
		"confirmed": false,
	})
	if !isError(result) {
		t.Error("expected error when confirmed=false")
	}
}

func TestWriteFiles_writesFiles(t *testing.T) {
	handler, root := makeWriteHandlerWithRoot(t)
	result := callWriteTool(t, handler, map[string]any{
		"base_path": root,
		"files": []any{
			map[string]any{"path": "terragrunt.hcl", "content": "# root"},
			map[string]any{"path": "modules/s3/main.tf", "content": "# s3"},
		},
		"confirmed": true,
	})
	if isError(result) {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	// Verify files exist.
	for _, rel := range []string{"terragrunt.hcl", "modules/s3/main.tf"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("expected file %q to exist: %v", rel, err)
		}
	}

	// Verify response JSON contains files_written.
	text := mcp.GetTextFromContent(result.Content[0])
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if _, ok := resp["files_written"]; !ok {
		t.Error("expected files_written in response")
	}
}

func TestWriteFiles_blocksPathTraversal(t *testing.T) {
	handler, root := makeWriteHandlerWithRoot(t)
	result := callWriteTool(t, handler, map[string]any{
		"base_path": root,
		"files":     []any{map[string]any{"path": "../../etc/passwd", "content": "evil"}},
		"confirmed": true,
	})
	if !isError(result) {
		t.Error("expected path traversal to be blocked")
	}
}

func TestWriteFiles_blocksDisallowedExtension(t *testing.T) {
	handler, root := makeWriteHandlerWithRoot(t)
	result := callWriteTool(t, handler, map[string]any{
		"base_path": root,
		"files":     []any{map[string]any{"path": "script.sh", "content": "echo hi"}},
		"confirmed": true,
	})
	if !isError(result) {
		t.Error("expected .sh extension to be blocked")
	}
}

func TestWriteFiles_blocksLocalExec(t *testing.T) {
	handler, root := makeWriteHandlerWithRoot(t)
	content := `provisioner "local-exec" { command = "rm -rf /" }`
	result := callWriteTool(t, handler, map[string]any{
		"base_path": root,
		"files":     []any{map[string]any{"path": "main.tf", "content": content}},
		"confirmed": true,
	})
	if !isError(result) {
		t.Error("expected local-exec to be blocked")
	}
}

func TestWriteFiles_blocksOutsideRoot(t *testing.T) {
	handler, _ := makeWriteHandlerWithRoot(t)
	result := callWriteTool(t, handler, map[string]any{
		"base_path": "/tmp/not-allowed",
		"files":     []any{map[string]any{"path": "main.tf", "content": "# ok"}},
		"confirmed": true,
	})
	if !isError(result) {
		t.Error("expected write outside allowed root to be blocked")
	}
}
