package terragrunt

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/groundwork-dev/groundwork/pkg/model"
)

//go:embed templates
var templateFS embed.FS

// Generator renders Terragrunt HCL files from ResourceSpecs.
type Generator struct {
	OutputDir string
}

// New returns a Generator that writes output to outputDir.
func New(outputDir string) *Generator {
	return &Generator{OutputDir: outputDir}
}

// Generate creates the Terragrunt directory structure for the given specs.
//
// Output layout:
//
//	<OutputDir>/
//	├── terragrunt.hcl          ← root (remote_state)
//	└── modules/
//	    └── <spec.Name>/
//	        ├── terragrunt.hcl  ← per-resource module
//	        └── main.tf         ← Terraform resource definition
func (g *Generator) Generate(specs []model.ResourceSpec, backend model.BackendConfig) error {
	rootTmpl, err := loadTemplate("root")
	if err != nil {
		return err
	}
	moduleTmpl, err := loadTemplate("module")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(g.OutputDir, 0o755); err != nil {
		return fmt.Errorf("generator: create output dir: %w", err)
	}

	// Render root terragrunt.hcl with backend configuration.
	if err := renderFile(rootTmpl, backend, filepath.Join(g.OutputDir, "terragrunt.hcl")); err != nil {
		return fmt.Errorf("generator: render root: %w", err)
	}

	for _, spec := range specs {
		moduleDir := filepath.Join(g.OutputDir, "modules", spec.Name)
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			return fmt.Errorf("generator: create module dir %q: %w", moduleDir, err)
		}

		if err := renderFile(moduleTmpl, spec, filepath.Join(moduleDir, "terragrunt.hcl")); err != nil {
			return fmt.Errorf("generator: render module %q: %w", spec.Name, err)
		}

		tfTmpl, err := loadTFTemplate(spec.Name)
		if err != nil {
			return fmt.Errorf("generator: load tf template %q: %w", spec.Name, err)
		}
		if err := renderFile(tfTmpl, spec, filepath.Join(moduleDir, "main.tf")); err != nil {
			return fmt.Errorf("generator: render tf %q: %w", spec.Name, err)
		}
	}

	return nil
}

// loadTFTemplate loads a service-specific .tf template, falling back to a generic stub.
func loadTFTemplate(service string) (*template.Template, error) {
	path := fmt.Sprintf("templates/tf/%s.tf.tmpl", service)
	content, err := templateFS.ReadFile(path)
	if err != nil {
		content, err = templateFS.ReadFile("templates/tf/generic.tf.tmpl")
		if err != nil {
			return nil, fmt.Errorf("generator: read tf template %q: %w", service, err)
		}
	}
	tmpl, err := template.New(service).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("generator: parse tf template %q: %w", service, err)
	}
	return tmpl, nil
}

// loadTemplate parses the named embedded template (root or module).
func loadTemplate(name string) (*template.Template, error) {
	path := fmt.Sprintf("templates/%s.hcl.tmpl", name)
	content, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("generator: read template %q: %w", name, err)
	}
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("generator: parse template %q: %w", name, err)
	}
	return tmpl, nil
}

// renderFile executes tmpl with data and writes the result to dest.
func renderFile(tmpl *template.Template, data any, dest string) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template %q: %w", tmpl.Name(), err)
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %q: %w", dest, err)
	}
	return nil
}