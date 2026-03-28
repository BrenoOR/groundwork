package guardrails

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sandbox enforces path restrictions. All read/write operations must be
// within one of the AllowedRoots.
type Sandbox struct {
	roots []string // absolute, cleaned paths
}

// NewSandbox returns a Sandbox restricting operations to the given root paths.
// Returns an error if any root cannot be resolved to an absolute path.
func NewSandbox(roots []string) (*Sandbox, error) {
	cleaned := make([]string, 0, len(roots))
	for _, r := range roots {
		abs, err := filepath.Abs(r)
		if err != nil {
			return nil, fmt.Errorf("sandbox: resolve root %q: %w", r, err)
		}
		cleaned = append(cleaned, filepath.Clean(abs))
	}
	return &Sandbox{roots: cleaned}, nil
}

// CheckRead verifies that path is inside an allowed root.
func (s *Sandbox) CheckRead(path string) error {
	return s.check(path)
}

// CheckWrite verifies that path is inside an allowed root.
func (s *Sandbox) CheckWrite(path string) error {
	return s.check(path)
}

// CheckRelative verifies that a relative path does not escape its base directory.
// This prevents path traversal like "../../etc/passwd".
func (s *Sandbox) CheckRelative(base, rel string) error {
	for _, c := range rel {
		if c < 0x20 {
			return fmt.Errorf("sandbox: path %q contains control characters", rel)
		}
	}

	joined := filepath.Join(base, rel)
	clean := filepath.Clean(joined)
	baseClean := filepath.Clean(base)

	if clean != baseClean && !strings.HasPrefix(clean, baseClean+string(filepath.Separator)) {
		return fmt.Errorf("sandbox: path %q escapes base directory", rel)
	}
	return nil
}

func (s *Sandbox) check(path string) error {
	for _, c := range path {
		if c < 0x20 {
			return fmt.Errorf("sandbox: path contains control characters")
		}
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("sandbox: resolve path: %w", err)
	}

	// Resolve symlinks. For paths that don't exist yet (write destination),
	// resolve the parent directory instead.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			parent, err2 := filepath.EvalSymlinks(filepath.Dir(abs))
			if err2 != nil {
				parent = filepath.Dir(abs)
			}
			resolved = filepath.Join(parent, filepath.Base(abs))
		} else {
			return fmt.Errorf("sandbox: resolve symlinks for %q: %w", path, err)
		}
	}

	for _, root := range s.roots {
		if resolved == root || strings.HasPrefix(resolved, root+string(filepath.Separator)) {
			return nil
		}
	}
	return fmt.Errorf("sandbox: path %q is outside allowed roots %v", path, s.roots)
}
