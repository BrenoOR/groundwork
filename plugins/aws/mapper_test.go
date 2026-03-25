package aws_test

import (
	"sort"
	"testing"

	awsplugin "github.com/groundwork-dev/groundwork/plugins/aws"
	"github.com/groundwork-dev/groundwork/pkg/model"
)

func newMapper(t *testing.T) *awsplugin.Mapper {
	t.Helper()
	m, err := awsplugin.NewMapper()
	if err != nil {
		t.Fatalf("NewMapper: %v", err)
	}
	return m
}

func TestMapper_knownServices(t *testing.T) {
	m := newMapper(t)

	cases := []struct {
		service  string
		wantType string
	}{
		{"s3", "aws_s3_bucket"},
		{"dynamodb", "aws_dynamodb_table"},
		{"sqs", "aws_sqs_queue"},
		{"sns", "aws_sns_topic"},
		{"lambda", "aws_lambda_function"},
		{"rds", "aws_db_instance"},
		{"secretsmanager", "aws_secretsmanager_secret"},
	}

	for _, tc := range cases {
		t.Run(tc.service, func(t *testing.T) {
			sdk := model.DetectedSDK{Name: "aws-sdk", Services: []string{tc.service}}
			specs, err := m.Map(sdk)
			if err != nil {
				t.Fatalf("Map: %v", err)
			}
			if len(specs) != 1 {
				t.Fatalf("expected 1 spec, got %d", len(specs))
			}
			if specs[0].Type != tc.wantType {
				t.Errorf("service %q: got type %q, want %q", tc.service, specs[0].Type, tc.wantType)
			}
			if specs[0].Provider != "aws" {
				t.Errorf("service %q: expected provider %q, got %q", tc.service, "aws", specs[0].Provider)
			}
		})
	}
}

func TestMapper_unknownServiceSkipped(t *testing.T) {
	m := newMapper(t)
	sdk := model.DetectedSDK{Name: "aws-sdk", Services: []string{"unknownservice"}}
	specs, err := m.Map(sdk)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if len(specs) != 0 {
		t.Errorf("expected no specs for unknown service, got %v", specs)
	}
}

func TestMapper_multipleServices(t *testing.T) {
	m := newMapper(t)
	sdk := model.DetectedSDK{
		Name:     "aws-sdk",
		Services: []string{"s3", "dynamodb", "sqs"},
	}
	specs, err := m.Map(sdk)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if len(specs) != 3 {
		t.Fatalf("expected 3 specs, got %d", len(specs))
	}

	types := make([]string, len(specs))
	for i, s := range specs {
		types[i] = s.Type
	}
	sort.Strings(types)
	want := []string{"aws_dynamodb_table", "aws_s3_bucket", "aws_sqs_queue"}
	if !equalSlices(types, want) {
		t.Errorf("got %v, want %v", types, want)
	}
}