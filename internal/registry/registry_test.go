package registry_test

import (
	"errors"
	"testing"

	"github.com/groundwork-dev/groundwork/internal/registry"
	"github.com/groundwork-dev/groundwork/pkg/model"
)

// --- test doubles ---

type stubPlugin struct {
	name  string
	sdks  []model.DetectedSDK
	specs []model.ResourceSpec
	// inject errors
	detectErr error
	mapErr    error
}

func (s *stubPlugin) Name() string { return s.name }

func (s *stubPlugin) Detect(_ []model.SourceFile) ([]model.DetectedSDK, error) {
	return s.sdks, s.detectErr
}

func (s *stubPlugin) Map(_ model.DetectedSDK) ([]model.ResourceSpec, error) {
	return s.specs, s.mapErr
}

var _ model.Plugin = (*stubPlugin)(nil)

// --- helpers ---

func makePlugin(name string, services []string, resourceType string) *stubPlugin {
	sdks := []model.DetectedSDK{{Name: name + "-sdk", Services: services}}
	specs := []model.ResourceSpec{{Provider: name, Type: resourceType, Name: services[0]}}
	return &stubPlugin{name: name, sdks: sdks, specs: specs}
}

// --- tests ---

func TestRegistry_registerAndLen(t *testing.T) {
	r := registry.New()
	if r.Len() != 0 {
		t.Fatalf("expected empty registry, got len=%d", r.Len())
	}

	p := makePlugin("aws", []string{"s3"}, "aws_s3_bucket")
	if err := r.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if r.Len() != 1 {
		t.Errorf("expected len=1, got %d", r.Len())
	}
}

func TestRegistry_duplicatePlugin(t *testing.T) {
	r := registry.New()
	p := makePlugin("aws", []string{"s3"}, "aws_s3_bucket")

	if err := r.Register(p); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := r.Register(p)
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
	if !errors.Is(err, registry.ErrDuplicatePlugin) {
		t.Errorf("expected ErrDuplicatePlugin, got %v", err)
	}
}

func TestRegistry_runAll_singlePlugin(t *testing.T) {
	r := registry.New()
	p := makePlugin("aws", []string{"s3"}, "aws_s3_bucket")
	_ = r.Register(p)

	specs, err := r.RunAll([]model.SourceFile{})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].Type != "aws_s3_bucket" {
		t.Errorf("unexpected resource type: %s", specs[0].Type)
	}
}

func TestRegistry_runAll_multiplePlugins(t *testing.T) {
	r := registry.New()
	_ = r.Register(makePlugin("aws", []string{"s3"}, "aws_s3_bucket"))
	_ = r.Register(makePlugin("gcp", []string{"storage"}, "google_storage_bucket"))

	specs, err := r.RunAll([]model.SourceFile{})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs (one per plugin), got %d", len(specs))
	}
}

func TestRegistry_runAll_pluginsDoNotInterfere(t *testing.T) {
	r := registry.New()
	_ = r.Register(makePlugin("aws", []string{"s3"}, "aws_s3_bucket"))
	_ = r.Register(makePlugin("gcp", []string{"pubsub"}, "google_pubsub_topic"))

	specs, err := r.RunAll([]model.SourceFile{})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}

	providers := map[string]bool{}
	for _, s := range specs {
		providers[s.Provider] = true
	}
	if !providers["aws"] || !providers["gcp"] {
		t.Errorf("expected both aws and gcp providers, got %v", providers)
	}
}

func TestRegistry_runAll_detectError_continuesOtherPlugins(t *testing.T) {
	r := registry.New()
	bad := &stubPlugin{name: "bad", detectErr: errors.New("boom")}
	good := makePlugin("aws", []string{"s3"}, "aws_s3_bucket")
	_ = r.Register(bad)
	_ = r.Register(good)

	specs, err := r.RunAll([]model.SourceFile{})
	if err == nil {
		t.Fatal("expected error from bad plugin, got nil")
	}
	// good plugin still produced its spec
	if len(specs) != 1 {
		t.Errorf("expected 1 spec from good plugin, got %d", len(specs))
	}
}

func TestRegistry_runAll_mapError_continuesOtherSDKs(t *testing.T) {
	r := registry.New()
	p := &stubPlugin{
		name:   "aws",
		sdks:   []model.DetectedSDK{{Name: "aws-sdk", Services: []string{"s3"}}},
		mapErr: errors.New("map failed"),
	}
	_ = r.Register(p)

	_, err := r.RunAll([]model.SourceFile{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegistry_runAll_noPlugins(t *testing.T) {
	r := registry.New()
	specs, err := r.RunAll([]model.SourceFile{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 0 {
		t.Errorf("expected no specs, got %d", len(specs))
	}
}