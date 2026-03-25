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

// normalizeService lower-cases and strips common suffixes so that
// "s3control", "s3-control" and "s3" all map to the same key.
func normalizeService(svc string) string {
	svc = strings.ToLower(svc)
	svc = strings.ReplaceAll(svc, "-", "")
	return svc
}