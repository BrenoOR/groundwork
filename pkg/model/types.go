package model

// SourceFile represents a single file from the scanned project.
type SourceFile struct {
	Path     string
	Language string
	Content  []byte
}

// DetectedSDK represents an SDK identified within the project source files.
type DetectedSDK struct {
	Name     string   // e.g. "aws-sdk-go-v2"
	Services []string // e.g. ["s3", "dynamodb"]
}

// ResourceSpec describes a cloud resource to be generated.
type ResourceSpec struct {
	Provider string         // e.g. "aws"
	Type     string         // e.g. "s3_bucket"
	Name     string         // logical name used in the generated output
	Params   map[string]any // additional resource parameters
}

// BackendConfig holds the S3 remote state configuration for Terragrunt.
type BackendConfig struct {
	Bucket    string // S3 bucket name
	Region    string // AWS region of the bucket
	LockTable string // DynamoDB table name for state locking
	Encrypt   bool   // enable SSE-S3 encryption at rest
}