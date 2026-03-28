package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	gwmcp "github.com/groundwork-dev/groundwork/internal/mcp"
	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
	"github.com/groundwork-dev/groundwork/internal/mcp/tools"
)

// newTestServer builds a full groundwork MCP server via mcptest,
// using the given allowed root. Returns the test server and a cancel func.
func newTestServer(t *testing.T, cfg gwmcp.Config) (*mcptest.Server, context.CancelFunc) {
	t.Helper()

	sb, err := guardrails.NewSandbox(cfg.AllowedRoots)
	if err != nil {
		t.Fatal(err)
	}

	limits := guardrails.SizeLimits{
		MaxFiles:        cfg.MaxFiles,
		MaxProjectBytes: cfg.MaxProjectBytes,
		MaxFileBytes:    cfg.MaxFileBytes,
	}
	lim := guardrails.NewRateLimiter(cfg.RatePerSec, cfg.Burst)
	cv := guardrails.NewContentValidator(cfg.AllowExecProvisioners, cfg.MaxFileBytes)
	auditor := guardrails.NewAuditor(os.Stderr)

	ts := mcptest.NewUnstartedServer(t)
	ts.AddTool(tools.ScanTool(), tools.HandleScan(sb, lim, limits, auditor))
	if !cfg.ReadOnly {
		ts.AddTool(tools.WriteFilesTool(), tools.HandleWriteFiles(sb, lim, cv, auditor))
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := ts.Start(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		ts.Close()
	})

	return ts, cancel
}

func callTool(t *testing.T, ts *mcptest.Server, toolName string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = args
	result, err := ts.Client().CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool %q: %v", toolName, err)
	}
	return result
}

// TestIntegration_ScanAndWrite tests the full scan → write flow via stdio MCP protocol.
func TestIntegration_ScanAndWrite(t *testing.T) {
	root := t.TempDir()

	// Create a tiny python project with S3 usage.
	// requirements.txt is required for the scanner to recognise Python.
	projectDir := filepath.Join(root, "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "requirements.txt"), []byte("boto3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "main.py"), []byte("import boto3\ns3 = boto3.client('s3')\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(root, "infra")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := gwmcp.DefaultConfig()
	cfg.AllowedRoots = []string{root}
	cfg.RatePerSec = 100
	cfg.Burst = 100

	ts, cancel := newTestServer(t, cfg)
	defer cancel()

	// Step 1: groundwork_scan
	scanResult := callTool(t, ts, "groundwork_scan", map[string]any{
		"project_path": projectDir,
	})
	if scanResult.IsError {
		t.Fatalf("scan failed: %v", scanResult.Content)
	}

	text := mcp.GetTextFromContent(scanResult.Content[0])
	var sr tools.ScanResult
	if err := json.Unmarshal([]byte(text), &sr); err != nil {
		t.Fatalf("parse scan result: %v", err)
	}
	if sr.Language != "python" {
		t.Errorf("expected python, got %q", sr.Language)
	}
	found := false
	for _, s := range sr.Services {
		if s == "s3" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected s3 service, got %v", sr.Services)
	}

	// Step 2: groundwork_write_files (confirmed=true)
	writeResult := callTool(t, ts, "groundwork_write_files", map[string]any{
		"base_path": outDir,
		"files": []any{
			map[string]any{"path": "terragrunt.hcl", "content": "# root"},
			map[string]any{"path": "modules/s3/main.tf", "content": "# s3 module"},
		},
		"confirmed": true,
	})
	if writeResult.IsError {
		t.Fatalf("write failed: %v", writeResult.Content)
	}

	// Verify files exist on disk.
	for _, rel := range []string{"terragrunt.hcl", "modules/s3/main.tf"} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Errorf("expected file %q: %v", rel, err)
		}
	}

	// Verify the response.
	writeText := mcp.GetTextFromContent(writeResult.Content[0])
	var wr map[string]any
	if err := json.Unmarshal([]byte(writeText), &wr); err != nil {
		t.Fatalf("parse write result: %v", err)
	}
	if _, ok := wr["files_written"]; !ok {
		t.Error("expected files_written in response")
	}
}

// TestIntegration_ReadOnly verifies that groundwork_write_files is not available
// when the server is started in read-only mode.
func TestIntegration_ReadOnly(t *testing.T) {
	root := t.TempDir()

	cfg := gwmcp.DefaultConfig()
	cfg.AllowedRoots = []string{root}
	cfg.ReadOnly = true
	cfg.RatePerSec = 100
	cfg.Burst = 100

	ts, cancel := newTestServer(t, cfg)
	defer cancel()

	// List tools and verify groundwork_write_files is absent.
	listReq := mcp.ListToolsRequest{}
	toolsResult, err := ts.Client().ListTools(context.Background(), listReq)
	if err != nil {
		t.Fatal(err)
	}

	for _, tool := range toolsResult.Tools {
		if tool.Name == "groundwork_write_files" {
			t.Error("groundwork_write_files should not be registered in read-only mode")
		}
	}

	// groundwork_scan should still be available.
	scanFound := false
	for _, tool := range toolsResult.Tools {
		if tool.Name == "groundwork_scan" {
			scanFound = true
		}
	}
	if !scanFound {
		t.Error("groundwork_scan should be available in read-only mode")
	}
}

// TestIntegration_FileLimit verifies that scanning a project that exceeds MaxFiles
// returns a warning in the scan result.
func TestIntegration_FileLimit(t *testing.T) {
	root := t.TempDir()

	// Create a project with 5 Python files.
	projectDir := filepath.Join(root, "bigapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "requirements.txt"), []byte("boto3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		name := filepath.Join(projectDir, strings.Repeat("a", i+1)+".py")
		content := "import boto3\ns3 = boto3.client('s3')\n"
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Set MaxFiles to 2 — should trigger a warning/truncation.
	cfg := gwmcp.DefaultConfig()
	cfg.AllowedRoots = []string{root}
	cfg.MaxFiles = 2
	cfg.RatePerSec = 100
	cfg.Burst = 100

	ts, cancel := newTestServer(t, cfg)
	defer cancel()

	scanResult := callTool(t, ts, "groundwork_scan", map[string]any{
		"project_path": projectDir,
	})
	if scanResult.IsError {
		t.Fatalf("scan failed: %v", scanResult.Content)
	}

	text := mcp.GetTextFromContent(scanResult.Content[0])
	var sr tools.ScanResult
	if err := json.Unmarshal([]byte(text), &sr); err != nil {
		t.Fatalf("parse scan result: %v", err)
	}

	// The scan should have been limited.
	if sr.FilesScanned > 2 {
		t.Errorf("expected at most 2 files scanned, got %d", sr.FilesScanned)
	}

	// There should be a warning about the file limit.
	if len(sr.Warnings) == 0 {
		t.Error("expected a warning about the file limit, got none")
	}

	hasLimitWarning := false
	for _, w := range sr.Warnings {
		if strings.Contains(strings.ToLower(w), "limit") || strings.Contains(strings.ToLower(w), "truncat") || strings.Contains(strings.ToLower(w), "max") {
			hasLimitWarning = true
		}
	}
	if !hasLimitWarning {
		t.Errorf("expected a file-limit warning, got: %v", sr.Warnings)
	}
}

// TestIntegration_ToolsAvailable verifies the full tool list in normal mode.
func TestIntegration_ToolsAvailable(t *testing.T) {
	root := t.TempDir()

	cfg := gwmcp.DefaultConfig()
	cfg.AllowedRoots = []string{root}
	cfg.RatePerSec = 100
	cfg.Burst = 100

	ts, cancel := newTestServer(t, cfg)
	defer cancel()

	listReq := mcp.ListToolsRequest{}
	toolsResult, err := ts.Client().ListTools(context.Background(), listReq)
	if err != nil {
		t.Fatal(err)
	}

	names := make(map[string]bool)
	for _, tool := range toolsResult.Tools {
		names[tool.Name] = true
	}

	for _, expected := range []string{"groundwork_scan", "groundwork_write_files"} {
		if !names[expected] {
			t.Errorf("expected tool %q to be registered", expected)
		}
	}
}

