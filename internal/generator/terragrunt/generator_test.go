package terragrunt_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/groundwork-dev/groundwork/internal/generator/terragrunt"
	"github.com/groundwork-dev/groundwork/pkg/model"
)

func goldenPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "golden", rel)
}

func readGolden(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(goldenPath(rel))
	if err != nil {
		t.Fatalf("read golden %q: %v", rel, err)
	}
	return string(b)
}

var testSpecs = []model.ResourceSpec{
	{Provider: "aws", Type: "aws_s3_bucket", Name: "s3", Params: map[string]any{}},
	{Provider: "aws", Type: "aws_dynamodb_table", Name: "dynamodb", Params: map[string]any{}},
	{Provider: "aws", Type: "aws_sqs_queue", Name: "sqs", Params: map[string]any{}},
}

var testBackend = model.BackendConfig{
	Bucket:    "my-tf-state",
	Region:    "us-east-1",
	LockTable: "my-tf-state-lock",
	Encrypt:   true,
}

func TestGenerate_createsOutputDir(t *testing.T) {
	out := t.TempDir()
	g := terragrunt.New(filepath.Join(out, "infra"))

	if err := g.Generate(testSpecs, testBackend); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if _, err := os.Stat(g.OutputDir); err != nil {
		t.Errorf("output dir not created: %v", err)
	}
}

func TestGenerate_rootHCL_goldenFile(t *testing.T) {
	out := t.TempDir()
	g := terragrunt.New(out)

	if err := g.Generate(testSpecs, testBackend); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(out, "terragrunt.hcl"))
	if err != nil {
		t.Fatalf("read root terragrunt.hcl: %v", err)
	}

	want := readGolden(t, "terragrunt.hcl")
	if string(got) != want {
		t.Errorf("root terragrunt.hcl mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenerate_moduleHCL_goldenFiles(t *testing.T) {
	cases := []struct {
		module string
		golden string
	}{
		{"s3", "modules/s3/terragrunt.hcl"},
		{"dynamodb", "modules/dynamodb/terragrunt.hcl"},
		{"sqs", "modules/sqs/terragrunt.hcl"},
	}

	out := t.TempDir()
	g := terragrunt.New(out)

	if err := g.Generate(testSpecs, testBackend); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.module, func(t *testing.T) {
			got, err := os.ReadFile(filepath.Join(out, "modules", tc.module, "terragrunt.hcl"))
			if err != nil {
				t.Fatalf("read module file: %v", err)
			}
			want := readGolden(t, tc.golden)
			if string(got) != want {
				t.Errorf("module %q mismatch:\ngot:\n%s\nwant:\n%s", tc.module, got, want)
			}
		})
	}
}

func TestGenerate_directoryStructure(t *testing.T) {
	out := t.TempDir()
	g := terragrunt.New(out)

	if err := g.Generate(testSpecs, testBackend); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expected := []string{
		"terragrunt.hcl",
		filepath.Join("modules", "s3", "terragrunt.hcl"),
		filepath.Join("modules", "dynamodb", "terragrunt.hcl"),
		filepath.Join("modules", "sqs", "terragrunt.hcl"),
	}

	for _, rel := range expected {
		path := filepath.Join(out, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %q not found: %v", rel, err)
		}
	}
}

func TestGenerate_emptySpecs(t *testing.T) {
	out := t.TempDir()
	g := terragrunt.New(out)

	if err := g.Generate(nil, testBackend); err != nil {
		t.Fatalf("Generate with no specs: %v", err)
	}

	if _, err := os.Stat(filepath.Join(out, "terragrunt.hcl")); err != nil {
		t.Errorf("root terragrunt.hcl missing for empty specs: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(out, "modules"))
	if len(entries) != 0 {
		t.Errorf("expected no modules, got %d", len(entries))
	}
}

func TestGenerate_backendFields(t *testing.T) {
	out := t.TempDir()
	g := terragrunt.New(out)

	backend := model.BackendConfig{
		Bucket:    "custom-bucket",
		Region:    "eu-west-1",
		LockTable: "custom-lock",
		Encrypt:   false,
	}

	if err := g.Generate(testSpecs, backend); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(out, "terragrunt.hcl"))
	if err != nil {
		t.Fatalf("read root: %v", err)
	}
	s := string(content)

	checks := []string{
		`bucket         = "custom-bucket"`,
		`region         = "eu-west-1"`,
		`dynamodb_table = "custom-lock"`,
		`encrypt        = false`,
	}
	for _, want := range checks {
		if !contains(s, want) {
			t.Errorf("root HCL missing %q\ngot:\n%s", want, s)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := range s {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
