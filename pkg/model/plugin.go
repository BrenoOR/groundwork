package model

// Detector analyses source files and identifies SDKs in use.
type Detector interface {
	Detect(files []SourceFile) ([]DetectedSDK, error)
}

// Mapper converts a detected SDK into a list of cloud resource specs.
type Mapper interface {
	Map(sdk DetectedSDK) ([]ResourceSpec, error)
}

// Plugin combines Detector and Mapper and exposes a name for registration.
type Plugin interface {
	Detector
	Mapper
	Name() string
}