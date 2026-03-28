package guardrails

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// execProvisionerRe matches local-exec and remote-exec provisioner blocks in HCL.
var execProvisionerRe = regexp.MustCompile(`provisioner\s+["'](?:local|remote)-exec["']`)

// allowedExtensions is the default set of file extensions the server will write.
var allowedExtensions = map[string]bool{
	".tf":     true,
	".hcl":    true,
	".tfvars": true,
	".json":   true,
	".yaml":   true,
	".yml":    true,
}

// ContentValidator validates the path and content of files before writing.
type ContentValidator struct {
	allowExecProvisioners bool
	maxFileBytes          int64
}

// NewContentValidator returns a ContentValidator.
func NewContentValidator(allowExecProvisioners bool, maxFileBytes int64) *ContentValidator {
	return &ContentValidator{
		allowExecProvisioners: allowExecProvisioners,
		maxFileBytes:          maxFileBytes,
	}
}

// Validate checks a single file's path and content against all rules.
// Returns a descriptive error if any rule is violated.
func (v *ContentValidator) Validate(path, content string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if !allowedExtensions[ext] {
		return fmt.Errorf("file %q has disallowed extension %q (allowed: .tf .hcl .tfvars .json .yaml .yml)", path, ext)
	}

	if int64(len(content)) > v.maxFileBytes {
		return fmt.Errorf("file %q exceeds max size of %d bytes", path, v.maxFileBytes)
	}

	// Reject binary content (null bytes indicate non-text).
	for i, b := range []byte(content) {
		if b == 0x00 {
			return fmt.Errorf("file %q appears to contain binary content (null byte at offset %d)", path, i)
		}
	}

	if !v.allowExecProvisioners && execProvisionerRe.MatchString(content) {
		return fmt.Errorf(
			"file %q contains a local-exec or remote-exec provisioner block, "+
				"which is disabled by default; start groundwork-mcp with --allow-exec-provisioners to permit",
			path,
		)
	}

	return nil
}
