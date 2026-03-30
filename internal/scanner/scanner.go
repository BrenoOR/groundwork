package scanner

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/groundwork-dev/groundwork/pkg/model"
)

// languageMarkers maps marker filenames to the language they identify.
// When a marker is found in a directory, all source files under that
// directory are tagged with the corresponding language.
var languageMarkers = map[string]string{
	"go.mod":           "go",
	"package.json":     "nodejs",
	"requirements.txt": "python",
	"pyproject.toml":   "python",
}

// skipDirs contains directory names that should never be walked.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
}

// Scanner walks a project directory and collects source files.
type Scanner struct {
	RootPath string
}

// New returns a Scanner rooted at the given path.
func New(rootPath string) *Scanner {
	return &Scanner{RootPath: rootPath}
}

// Scan walks RootPath and returns all source files with their detected language.
// Language is inferred from marker files (go.mod, package.json, etc.).
// Files inside skipped directories are ignored.
func (s *Scanner) Scan() ([]model.SourceFile, error) {
	// First pass: collect all marker files to build a dir→language map.
	dirLang := map[string]string{}

	err := filepath.WalkDir(s.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			if lang, ok := languageMarkers[d.Name()]; ok {
				dir := filepath.Dir(path)
				// A more specific (deeper) marker takes precedence.
				if _, exists := dirLang[dir]; !exists {
					dirLang[dir] = lang
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Second pass: collect source files and resolve their language.
	var files []model.SourceFile

	err = filepath.WalkDir(s.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip marker files themselves and non-regular files.
		if _, isMarker := languageMarkers[d.Name()]; isMarker {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		lang := s.resolveLanguage(path, dirLang)

		content, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return nil
			}
			return err
		}

		files = append(files, model.SourceFile{
			Path:     path,
			Language: lang,
			Content:  content,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// resolveLanguage finds the language for a file by walking up the directory
// tree until it finds a dir with a known marker, or returns "unknown".
func (s *Scanner) resolveLanguage(filePath string, dirLang map[string]string) string {
	dir := filepath.Dir(filePath)
	for {
		if lang, ok := dirLang[dir]; ok {
			return lang
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "unknown"
}