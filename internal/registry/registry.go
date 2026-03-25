package registry

import (
	"errors"
	"fmt"

	"github.com/groundwork-dev/groundwork/pkg/model"
)

// ErrDuplicatePlugin is returned when a plugin with the same name is registered twice.
var ErrDuplicatePlugin = errors.New("registry: duplicate plugin name")

// Registry holds the set of registered plugins and orchestrates detection and mapping.
type Registry struct {
	plugins map[string]model.Plugin
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{plugins: map[string]model.Plugin{}}
}

// Register adds a plugin to the registry.
// Returns ErrDuplicatePlugin if a plugin with the same name was already registered.
func (r *Registry) Register(p model.Plugin) error {
	if _, exists := r.plugins[p.Name()]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicatePlugin, p.Name())
	}
	r.plugins[p.Name()] = p
	return nil
}

// RunAll runs every registered plugin against the provided files and aggregates
// the resulting ResourceSpecs. Errors from individual plugins are collected and
// returned as a single joined error so all plugins always get a chance to run.
func (r *Registry) RunAll(files []model.SourceFile) ([]model.ResourceSpec, error) {
	var (
		allSpecs []model.ResourceSpec
		errs     []error
	)

	for _, p := range r.plugins {
		sdks, err := p.Detect(files)
		if err != nil {
			errs = append(errs, fmt.Errorf("plugin %q detect: %w", p.Name(), err))
			continue
		}

		for _, sdk := range sdks {
			specs, err := p.Map(sdk)
			if err != nil {
				errs = append(errs, fmt.Errorf("plugin %q map %q: %w", p.Name(), sdk.Name, err))
				continue
			}
			allSpecs = append(allSpecs, specs...)
		}
	}

	return allSpecs, errors.Join(errs...)
}

// Plugins returns the number of registered plugins.
func (r *Registry) Len() int { return len(r.plugins) }