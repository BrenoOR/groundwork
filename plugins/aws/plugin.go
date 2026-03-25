package aws

import (
	"fmt"

	"github.com/groundwork-dev/groundwork/pkg/model"
)

// Plugin is the AWS provider plugin. It implements model.Plugin by combining
// Detector and Mapper into a single, registrable unit.
type Plugin struct {
	detector *Detector
	mapper   *Mapper
}

// New constructs the AWS Plugin. Returns an error if the mapper fails to
// initialise (e.g. malformed mappings.yaml).
func New() (*Plugin, error) {
	mapper, err := NewMapper()
	if err != nil {
		return nil, fmt.Errorf("aws plugin: %w", err)
	}
	return &Plugin{
		detector: &Detector{},
		mapper:   mapper,
	}, nil
}

// Name implements model.Plugin.
func (p *Plugin) Name() string { return "aws" }

// Detect implements model.Detector.
func (p *Plugin) Detect(files []model.SourceFile) ([]model.DetectedSDK, error) {
	return p.detector.Detect(files)
}

// Map implements model.Mapper.
func (p *Plugin) Map(sdk model.DetectedSDK) ([]model.ResourceSpec, error) {
	return p.mapper.Map(sdk)
}

// compile-time interface check.
var _ model.Plugin = (*Plugin)(nil)