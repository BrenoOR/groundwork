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

func makeScanHandler(t *testing.T, allowedRoots []string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	t.Helper()
	sb, err := guardrails.NewSandbox(allowedRoots)
	if err != nil {
		t.Fatal(err)
	}
	lim := guardrails.NewRateLimiter(100, 100)
	limits := guardrails.DefaultSizeLimits()
	auditor := guardrails.NewAuditor(os.Stderr)
	return tools.HandleScan(sb, lim, limits, auditor)
}

func callScanTool(t *testing.T, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), projectPath string) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"project_path": projectPath}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	return result
}

func TestScan_detectsPythonS3(t *testing.T) {
	// Use the existing example project.
	examplePath := filepath.Join("..", "..", "..", "examples", "python_api")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Skip("example project not found")
	}

	abs, _ := filepath.Abs(examplePath)
	handler := makeScanHandler(t, []string{abs})
	result := callScanTool(t, handler, abs)

	if isError(result) {
		t.Fatalf("scan failed: %v", result.Content)
	}

	text := mcp.GetTextFromContent(result.Content[0])
	var sr tools.ScanResult
	if err := json.Unmarshal([]byte(text), &sr); err != nil {
		t.Fatalf("parse result: %v", err)
	}

	if sr.Language != "python" {
		t.Errorf("expected language python, got %q", sr.Language)
	}
	if len(sr.Services) == 0 {
		t.Error("expected at least one service detected")
	}
	found := false
	for _, s := range sr.Services {
		if s == "s3" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected s3 in services, got %v", sr.Services)
	}
}

func TestScan_blocksOutsideRoot(t *testing.T) {
	root := t.TempDir()
	handler := makeScanHandler(t, []string{root})

	result := callScanTool(t, handler, "/etc")
	if !isError(result) {
		t.Error("expected scan of /etc to be blocked")
	}
}

func TestScan_missingProjectPath(t *testing.T) {
	root := t.TempDir()
	handler := makeScanHandler(t, []string{root})

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !isError(result) {
		t.Error("expected error for missing project_path")
	}
}
