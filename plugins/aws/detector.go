package aws

import (
	"regexp"
	"strings"

	"github.com/groundwork-dev/groundwork/pkg/model"
)

const sdkName = "aws-sdk"

// Language-specific patterns to extract AWS service names.
var (
	// github.com/aws/aws-sdk-go-v2/service/<service>
	goImportRe = regexp.MustCompile(`github\.com/aws/aws-sdk-go(?:-v2)?/service/(\w+)`)

	// boto3.client('s3') or boto3.resource("dynamodb")
	pythonClientRe = regexp.MustCompile(`boto3\.(?:client|resource)\(\s*['"](\w+)['"]`)

	// @aws-sdk/client-<service> (package.json dep or JS/TS import)
	nodePackageRe = regexp.MustCompile(`@aws-sdk/client-([\w-]+)`)

	// import software.amazon.awssdk.services.<service>.<Class> (SDK v2)
	javaSDKv2Re = regexp.MustCompile(`import\s+software\.amazon\.awssdk\.services\.(\w+)`)

	// import com.amazonaws.services.<service>.<Class> (SDK v1)
	javaSDKv1Re = regexp.MustCompile(`import\s+com\.amazonaws\.services\.(\w+)`)

	// SDK v1 package names carry a version suffix like "v2" (e.g. "dynamodbv2").
	// This pattern matches only "v" followed by digits at the end of a word,
	// so "ec2" (no leading "v") is preserved as-is.
	javaV1VersionRe = regexp.MustCompile(`v\d+$`)
)

// Detector identifies AWS SDK usage across Go, Python, and Node.js source files.
type Detector struct{}

// Detect scans the provided files and returns a single DetectedSDK entry
// containing all AWS services found, grouped by language.
func (d *Detector) Detect(files []model.SourceFile) ([]model.DetectedSDK, error) {
	seen := map[string]bool{}

	for _, f := range files {
		src := string(f.Content)
		var services []string

		switch f.Language {
		case "go":
			services = extractMatches(goImportRe, src, 1)
		case "python":
			services = extractMatches(pythonClientRe, src, 1)
		case "nodejs":
			services = extractNodeServices(src)
		case "java":
			services = extractJavaServices(src)
		}

		for _, svc := range services {
			seen[normalizeService(svc)] = true
		}
	}

	if len(seen) == 0 {
		return nil, nil
	}

	sdk := model.DetectedSDK{Name: sdkName}
	for svc := range seen {
		sdk.Services = append(sdk.Services, svc)
	}
	return []model.DetectedSDK{sdk}, nil
}

// extractMatches returns all unique subgroup captures from a regexp.
func extractMatches(re *regexp.Regexp, src string, group int) []string {
	matches := re.FindAllStringSubmatch(src, -1)
	seen := map[string]bool{}
	var out []string
	for _, m := range matches {
		if v := m[group]; !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// extractNodeServices handles both package.json dependency keys and
// JS/TS require/import statements.
func extractNodeServices(src string) []string {
	return extractMatches(nodePackageRe, src, 1)
}

// extractJavaServices collects AWS service names from Java source files,
// handling both SDK v2 (software.amazon.awssdk) and SDK v1 (com.amazonaws).
func extractJavaServices(src string) []string {
	seen := map[string]bool{}
	var out []string
	for _, svc := range extractMatches(javaSDKv2Re, src, 1) {
		n := normalizeService(svc)
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	for _, svc := range extractMatches(javaSDKv1Re, src, 1) {
		n := normalizeService(stripVersionSuffix(svc))
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// stripVersionSuffix removes trailing "v<N>" version suffixes from SDK v1
// service package names (e.g. "dynamodbv2" → "dynamodb", "ec2" stays "ec2").
func stripVersionSuffix(svc string) string {
	return javaV1VersionRe.ReplaceAllString(strings.ToLower(svc), "")
}

// normalizeService lower-cases and strips common suffixes so that
// "s3control", "s3-control" and "s3" all map to the same key.
func normalizeService(svc string) string {
	svc = strings.ToLower(svc)
	svc = strings.ReplaceAll(svc, "-", "")
	return svc
}