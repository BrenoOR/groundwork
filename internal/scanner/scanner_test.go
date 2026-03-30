package scanner_test

import (
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/groundwork-dev/groundwork/internal/scanner"
)

// testdataPath returns the absolute path to a testdata subdirectory.
func testdataPath(sub string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", sub)
}

func TestScan_goProject(t *testing.T) {
	s := scanner.New(testdataPath("go_project"))
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one file")
	}
	for _, f := range files {
		if f.Language != "go" {
			t.Errorf("file %q: expected language %q, got %q", f.Path, "go", f.Language)
		}
		if len(f.Content) == 0 {
			t.Errorf("file %q: content is empty", f.Path)
		}
	}
}

func TestScan_pythonProject(t *testing.T) {
	s := scanner.New(testdataPath("python_project"))
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one file")
	}
	for _, f := range files {
		if f.Language != "python" {
			t.Errorf("file %q: expected language %q, got %q", f.Path, "python", f.Language)
		}
	}
}

func TestScan_nodeProject(t *testing.T) {
	s := scanner.New(testdataPath("node_project"))
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one file")
	}
	for _, f := range files {
		if f.Language != "nodejs" {
			t.Errorf("file %q: expected language %q, got %q", f.Path, "nodejs", f.Language)
		}
	}
}

func TestScan_mixedProject(t *testing.T) {
	s := scanner.New(testdataPath("mixed_project"))
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	langs := map[string]int{}
	for _, f := range files {
		langs[f.Language]++
	}

	if langs["go"] == 0 {
		t.Error("expected at least one go file in mixed project")
	}
	if langs["nodejs"] == 0 {
		t.Error("expected at least one nodejs file in mixed project")
	}
}

func TestScan_markerFilesExcluded(t *testing.T) {
	s := scanner.New(testdataPath("go_project"))
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range files {
		base := filepath.Base(f.Path)
		if base == "go.mod" || base == "package.json" || base == "requirements.txt" || base == "pyproject.toml" {
			t.Errorf("marker file %q should not appear in results", f.Path)
		}
	}
}

func TestScan_contentMatchesDisk(t *testing.T) {
	s := scanner.New(testdataPath("go_project"))
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, filepath.Base(f.Path))
	}
	sort.Strings(names)

	if len(names) == 0 || names[0] != "main.go" {
		t.Errorf("expected main.go in results, got %v", names)
	}
}

func TestScan_javaProject(t *testing.T) {
	s := scanner.New(testdataPath("java_project"))
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one file")
	}
	for _, f := range files {
		if f.Language != "java" {
			t.Errorf("file %q: expected language %q, got %q", f.Path, "java", f.Language)
		}
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, filepath.Base(f.Path))
	}
	sort.Strings(names)
	found := false
	for _, n := range names {
		if n == "App.java" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected App.java in results, got %v", names)
	}
}

func TestScan_invalidPath(t *testing.T) {
	s := scanner.New("/nonexistent/path/that/does/not/exist")
	_, err := s.Scan()
	if err == nil {
		t.Fatal("expected error for invalid root path, got nil")
	}
}