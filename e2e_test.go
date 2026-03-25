package groundwork_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// TestMain compiles the binary once and runs all e2e tests against it.
func TestMain(m *testing.M) {
	// Build the binary into a temp dir shared across tests.
	bin, err := buildBinary()
	if err != nil {
		panic("e2e: build failed: " + err.Error())
	}
	defer os.Remove(bin)

	groundworkBin = bin
	os.Exit(m.Run())
}

var groundworkBin string

func buildBinary() (string, error) {
	tmp, err := os.CreateTemp("", "groundwork-e2e-*")
	if err != nil {
		return "", err
	}
	tmp.Close()

	cmd := exec.Command("go", "build", "-o", tmp.Name(), "./cmd/groundwork")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return tmp.Name(), cmd.Run()
}

func fixtureDir(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "e2e", name)
}

func runGroundwork(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command(groundworkBin, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("groundwork %v: %v", args, err)
	}
}

func assertModules(t *testing.T, outputDir string, want []string) {
	t.Helper()
	modulesDir := filepath.Join(outputDir, "modules")
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		t.Fatalf("read modules dir: %v", err)
	}

	got := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			got = append(got, e.Name())
		}
	}
	sort.Strings(got)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("modules: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("module[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func assertFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %q to exist: %v", path, err)
	}
}

// --- tests ---

func TestE2E_goProject(t *testing.T) {
	out := t.TempDir()
	runGroundwork(t, "--input", fixtureDir("go_project"), "--output", out, "--state-bucket", "e2e-tf-state")

	assertFile(t, filepath.Join(out, "terragrunt.hcl"))
	assertModules(t, out, []string{"dynamodb", "lambda", "s3", "sqs"})

	assertFile(t, filepath.Join(out, "modules", "s3", "terragrunt.hcl"))
}

func TestE2E_pythonProject(t *testing.T) {
	out := t.TempDir()
	runGroundwork(t, "--input", fixtureDir("python_project"), "--output", out, "--state-bucket", "e2e-tf-state")

	assertFile(t, filepath.Join(out, "terragrunt.hcl"))
	assertModules(t, out, []string{"s3", "secretsmanager", "sns", "sqs"})
}

func TestE2E_nodeProject(t *testing.T) {
	out := t.TempDir()
	runGroundwork(t, "--input", fixtureDir("node_project"), "--output", out, "--state-bucket", "e2e-tf-state")

	assertFile(t, filepath.Join(out, "terragrunt.hcl"))
	assertModules(t, out, []string{"dynamodb", "lambda", "s3", "sns", "sqs"})
}

func TestE2E_dryRun_noFilesWritten(t *testing.T) {
	out := t.TempDir()
	runGroundwork(t, "--input", fixtureDir("go_project"), "--output", out, "--dry-run", "--state-bucket", "e2e-tf-state")

	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		t.Errorf("dry-run wrote files: %v", entries)
	}
}

func TestE2E_missingStateBucket_fails(t *testing.T) {
	cmd := exec.Command(groundworkBin, "--input", fixtureDir("go_project"))
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit when --state-bucket is missing")
	}
}

func TestE2E_rootHCL_containsBackendFields(t *testing.T) {
	out := t.TempDir()
	runGroundwork(t, "--input", fixtureDir("go_project"), "--output", out,
		"--state-bucket", "my-state-bucket",
		"--state-region", "eu-west-1",
		"--state-lock-table", "my-lock-table",
	)

	content, err := os.ReadFile(filepath.Join(out, "terragrunt.hcl"))
	if err != nil {
		t.Fatalf("read terragrunt.hcl: %v", err)
	}
	s := string(content)

	for _, want := range []string{
		`bucket         = "my-state-bucket"`,
		`region         = "eu-west-1"`,
		`dynamodb_table = "my-lock-table"`,
		`encrypt        = true`,
	} {
		if !containsSubstr(s, want) {
			t.Errorf("terragrunt.hcl missing %q\ngot:\n%s", want, s)
		}
	}
}

func TestE2E_lockTableDefaultsSuffix(t *testing.T) {
	out := t.TempDir()
	runGroundwork(t, "--input", fixtureDir("go_project"), "--output", out,
		"--state-bucket", "my-bucket",
	)

	content, _ := os.ReadFile(filepath.Join(out, "terragrunt.hcl"))
	if !containsSubstr(string(content), `dynamodb_table = "my-bucket-lock"`) {
		t.Errorf("expected default lock table my-bucket-lock\ngot:\n%s", content)
	}
}

func containsSubstr(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && func() bool {
		for i := range s {
			if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}

func TestE2E_unknownProvider_fails(t *testing.T) {
	cmd := exec.Command(groundworkBin, "--provider", "azure", "--state-bucket", "x")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit for unknown provider")
	}
}

func TestE2E_helpFlag(t *testing.T) {
	cmd := exec.Command(groundworkBin, "--help")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Error("expected help output, got nothing")
	}
}