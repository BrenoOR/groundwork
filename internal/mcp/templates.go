package mcp

import (
	"fmt"
	"path"

	"github.com/groundwork-dev/groundwork/internal/generator/terragrunt"
)

// loadTemplateContent returns the reference .tf template content for the given service.
// Falls back to the generic template if no service-specific template exists.
func loadTemplateContent(service string) (string, error) {
	fs := terragrunt.TemplateFS
	p := path.Join("templates", "tf", service+".tf.tmpl")
	b, err := fs.ReadFile(p)
	if err != nil {
		b, err = fs.ReadFile("templates/tf/generic.tf.tmpl")
		if err != nil {
			return "", fmt.Errorf("no template found for service %q", service)
		}
	}
	return string(b), nil
}
