package aws

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/groundwork-dev/groundwork/pkg/model"
)

//go:embed mappings.yaml
var mappingsYAML []byte

type mappingsFile struct {
	Services map[string]string `yaml:"services"`
}

// Mapper converts a DetectedSDK into ResourceSpecs using the YAML mapping table.
type Mapper struct {
	services map[string]string // service name → terraform resource type
}

// NewMapper parses the embedded mappings.yaml and returns a ready Mapper.
func NewMapper() (*Mapper, error) {
	var mf mappingsFile
	if err := yaml.Unmarshal(mappingsYAML, &mf); err != nil {
		return nil, fmt.Errorf("aws mapper: parse mappings: %w", err)
	}
	return &Mapper{services: mf.Services}, nil
}

// Map converts a DetectedSDK into a slice of ResourceSpecs.
// Services that have no mapping entry are silently skipped.
func (m *Mapper) Map(sdk model.DetectedSDK) ([]model.ResourceSpec, error) {
	var specs []model.ResourceSpec
	for _, svc := range sdk.Services {
		resType, ok := m.services[svc]
		if !ok {
			continue
		}
		specs = append(specs, model.ResourceSpec{
			Provider: "aws",
			Type:     resType,
			Name:     svc,
			Params:   map[string]any{},
		})
	}
	return specs, nil
}