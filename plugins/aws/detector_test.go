package aws_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	awsplugin "github.com/groundwork-dev/groundwork/plugins/aws"
	"github.com/groundwork-dev/groundwork/pkg/model"
)

func testdataFile(name, language string) model.SourceFile {
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "testdata", name)
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return model.SourceFile{Path: path, Language: language, Content: content}
}

func sortedServices(sdks []model.DetectedSDK) []string {
	seen := map[string]bool{}
	for _, sdk := range sdks {
		for _, svc := range sdk.Services {
			seen[svc] = true
		}
	}
	out := make([]string, 0, len(seen))
	for svc := range seen {
		out = append(out, svc)
	}
	sort.Strings(out)
	return out
}

func TestDetector_goServices(t *testing.T) {
	d := &awsplugin.Detector{}
	files := []model.SourceFile{testdataFile("go_main.go", "go")}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svcs := sortedServices(sdks)
	want := []string{"dynamodb", "s3", "sqs"}
	if !equalSlices(svcs, want) {
		t.Errorf("go detector: got %v, want %v", svcs, want)
	}
}

func TestDetector_pythonServices(t *testing.T) {
	d := &awsplugin.Detector{}
	files := []model.SourceFile{testdataFile("python_app.py", "python")}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svcs := sortedServices(sdks)
	want := []string{"dynamodb", "s3", "secretsmanager"}
	if !equalSlices(svcs, want) {
		t.Errorf("python detector: got %v, want %v", svcs, want)
	}
}

func TestDetector_nodeServices(t *testing.T) {
	d := &awsplugin.Detector{}
	files := []model.SourceFile{testdataFile("node_index.js", "nodejs")}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svcs := sortedServices(sdks)
	want := []string{"dynamodb", "s3", "sns", "sqs"}
	if !equalSlices(svcs, want) {
		t.Errorf("node detector: got %v, want %v", svcs, want)
	}
}

func TestDetector_noAWSUsage(t *testing.T) {
	d := &awsplugin.Detector{}
	files := []model.SourceFile{
		{Path: "main.go", Language: "go", Content: []byte(`package main; func main() {}`)},
	}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sdks) != 0 {
		t.Errorf("expected no SDKs, got %v", sdks)
	}
}

func TestDetector_deduplicatesServices(t *testing.T) {
	d := &awsplugin.Detector{}
	src := []byte(`
		import "github.com/aws/aws-sdk-go-v2/service/s3"
		import "github.com/aws/aws-sdk-go-v2/service/s3"
	`)
	files := []model.SourceFile{{Path: "dup.go", Language: "go", Content: src}}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svcs := sortedServices(sdks)
	if len(svcs) != 1 || svcs[0] != "s3" {
		t.Errorf("expected exactly [s3], got %v", svcs)
	}
}

func TestDetector_javaServicesSDKv2(t *testing.T) {
	d := &awsplugin.Detector{}
	files := []model.SourceFile{testdataFile("java_app.java", "java")}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svcs := sortedServices(sdks)
	want := []string{"dynamodb", "s3", "secretsmanager", "sqs"}
	if !equalSlices(svcs, want) {
		t.Errorf("java detector: got %v, want %v", svcs, want)
	}
}

func TestDetector_javaSDKv1VersionSuffix(t *testing.T) {
	d := &awsplugin.Detector{}
	src := []byte(`import com.amazonaws.services.dynamodbv2.AmazonDynamoDB;`)
	files := []model.SourceFile{{Path: "App.java", Language: "java", Content: src}}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svcs := sortedServices(sdks)
	if len(svcs) != 1 || svcs[0] != "dynamodb" {
		t.Errorf("expected [dynamodb], got %v", svcs)
	}
}

func TestDetector_javaDeduplicatesAcrossSDKVersions(t *testing.T) {
	d := &awsplugin.Detector{}
	src := []byte(`
import software.amazon.awssdk.services.s3.S3Client;
import com.amazonaws.services.s3.AmazonS3;
`)
	files := []model.SourceFile{{Path: "App.java", Language: "java", Content: src}}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svcs := sortedServices(sdks)
	if len(svcs) != 1 || svcs[0] != "s3" {
		t.Errorf("expected exactly [s3], got %v", svcs)
	}
}

func TestDetector_javaNoAWSUsage(t *testing.T) {
	d := &awsplugin.Detector{}
	files := []model.SourceFile{
		{Path: "App.java", Language: "java", Content: []byte(`public class App {}`)},
	}

	sdks, err := d.Detect(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sdks) != 0 {
		t.Errorf("expected no SDKs, got %v", sdks)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}