package version

import "encoding/json"

// These variables are intended to be overridden at build time using -ldflags.
// Example:
//   go build -ldflags "-X 'go-core/version.Version=1.0.0' -X 'go-core/version.Commit=$(git rev-parse HEAD)' -X 'go-core/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" ./...

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info holds version information for JSON output.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// JSON returns version info as a JSON string.
func JSON() string {
	b, _ := json.Marshal(Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	})
	return string(b)
}
