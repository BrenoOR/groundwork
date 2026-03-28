package guardrails_test

import (
	"strings"
	"testing"

	"github.com/groundwork-dev/groundwork/internal/mcp/guardrails"
)

func TestContentValidator_allowedExtensions(t *testing.T) {
	cv := guardrails.NewContentValidator(false, 512*1024)

	allowed := []string{"main.tf", "root.hcl", "vars.tfvars", "config.json", "values.yaml", "values.yml"}
	for _, f := range allowed {
		if err := cv.Validate(f, "# content"); err != nil {
			t.Errorf("expected %q to be allowed: %v", f, err)
		}
	}
}

func TestContentValidator_disallowedExtension(t *testing.T) {
	cv := guardrails.NewContentValidator(false, 512*1024)

	disallowed := []string{"main.sh", "script.py", "Makefile", "README.md"}
	for _, f := range disallowed {
		if err := cv.Validate(f, "content"); err == nil {
			t.Errorf("expected %q to be blocked", f)
		}
	}
}

func TestContentValidator_binaryContent(t *testing.T) {
	cv := guardrails.NewContentValidator(false, 512*1024)
	if err := cv.Validate("main.tf", "text\x00binary"); err == nil {
		t.Error("expected binary content to be blocked")
	}
}

func TestContentValidator_execProvisionerBlocked(t *testing.T) {
	cv := guardrails.NewContentValidator(false, 512*1024)
	content := `resource "null_resource" "x" {
  provisioner "local-exec" {
    command = "echo hi"
  }
}`
	if err := cv.Validate("main.tf", content); err == nil {
		t.Error("expected local-exec provisioner to be blocked by default")
	}
}

func TestContentValidator_execProvisionerAllowed(t *testing.T) {
	cv := guardrails.NewContentValidator(true, 512*1024)
	content := `resource "null_resource" "x" {
  provisioner "local-exec" {
    command = "echo hi"
  }
}`
	if err := cv.Validate("main.tf", content); err != nil {
		t.Errorf("expected local-exec to be allowed with flag set: %v", err)
	}
}

func TestContentValidator_remoteExecBlocked(t *testing.T) {
	cv := guardrails.NewContentValidator(false, 512*1024)
	content := `provisioner "remote-exec" { inline = ["echo hi"] }`
	if err := cv.Validate("main.tf", content); err == nil {
		t.Error("expected remote-exec to be blocked")
	}
}

func TestContentValidator_fileTooLarge(t *testing.T) {
	cv := guardrails.NewContentValidator(false, 10)
	if err := cv.Validate("main.tf", strings.Repeat("x", 11)); err == nil {
		t.Error("expected oversized file to be blocked")
	}
}

func TestContentValidator_fileSizeExactlyLimit(t *testing.T) {
	cv := guardrails.NewContentValidator(false, 10)
	if err := cv.Validate("main.tf", strings.Repeat("x", 10)); err != nil {
		t.Errorf("expected file at exact limit to be allowed: %v", err)
	}
}
