package aws_test

import (
	"testing"

	awsplugin "github.com/groundwork-dev/groundwork/plugins/aws"
	"github.com/groundwork-dev/groundwork/pkg/model"
)

func TestPlugin_name(t *testing.T) {
	p, err := awsplugin.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Name() != "aws" {
		t.Errorf("expected name %q, got %q", "aws", p.Name())
	}
}

func TestPlugin_endToEnd_go(t *testing.T) {
	p, err := awsplugin.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	files := []model.SourceFile{testdataFile("go_main.go", "go")}

	sdks, err := p.Detect(files)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(sdks) == 0 {
		t.Fatal("expected SDKs, got none")
	}

	var allSpecs []model.ResourceSpec
	for _, sdk := range sdks {
		specs, err := p.Map(sdk)
		if err != nil {
			t.Fatalf("Map: %v", err)
		}
		allSpecs = append(allSpecs, specs...)
	}

	if len(allSpecs) == 0 {
		t.Fatal("expected resource specs, got none")
	}
	for _, spec := range allSpecs {
		if spec.Provider != "aws" {
			t.Errorf("expected provider %q, got %q", "aws", spec.Provider)
		}
	}
}

// compile-time interface check.
var _ model.Plugin = (*awsplugin.Plugin)(nil)