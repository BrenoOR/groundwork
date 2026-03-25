package model_test

import (
	"testing"

	"github.com/groundwork-dev/groundwork/pkg/model"
)

// stubPlugin is a compile-time assertion that any concrete type implementing
// all three methods satisfies the Plugin interface.
type stubPlugin struct{}

func (s *stubPlugin) Name() string { return "stub" }

func (s *stubPlugin) Detect(_ []model.SourceFile) ([]model.DetectedSDK, error) {
	return []model.DetectedSDK{{Name: "stub-sdk", Services: []string{"svc"}}}, nil
}

func (s *stubPlugin) Map(sdk model.DetectedSDK) ([]model.ResourceSpec, error) {
	return []model.ResourceSpec{{Provider: "stub", Type: sdk.Services[0], Name: sdk.Name}}, nil
}

// Compile-time interface satisfaction check.
var _ model.Plugin = (*stubPlugin)(nil)

func TestDetect_returnsSDKs(t *testing.T) {
	p := &stubPlugin{}
	files := []model.SourceFile{{Path: "main.go", Language: "go", Content: []byte("package main")}}

	sdks, err := p.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sdks) != 1 {
		t.Fatalf("expected 1 sdk, got %d", len(sdks))
	}
	if sdks[0].Name != "stub-sdk" {
		t.Errorf("expected sdk name %q, got %q", "stub-sdk", sdks[0].Name)
	}
}

func TestMap_returnsResourceSpecs(t *testing.T) {
	p := &stubPlugin{}
	sdk := model.DetectedSDK{Name: "stub-sdk", Services: []string{"svc"}}

	specs, err := p.Map(sdk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].Provider != "stub" {
		t.Errorf("expected provider %q, got %q", "stub", specs[0].Provider)
	}
	if specs[0].Type != "svc" {
		t.Errorf("expected type %q, got %q", "svc", specs[0].Type)
	}
}

func TestSourceFile_fields(t *testing.T) {
	f := model.SourceFile{Path: "foo/bar.py", Language: "python", Content: []byte("import boto3")}
	if f.Path != "foo/bar.py" {
		t.Errorf("unexpected path: %s", f.Path)
	}
	if f.Language != "python" {
		t.Errorf("unexpected language: %s", f.Language)
	}
}

func TestResourceSpec_params(t *testing.T) {
	spec := model.ResourceSpec{
		Provider: "aws",
		Type:     "s3_bucket",
		Name:     "my-bucket",
		Params:   map[string]any{"versioning": true},
	}
	v, ok := spec.Params["versioning"].(bool)
	if !ok || !v {
		t.Errorf("expected versioning param to be true")
	}
}