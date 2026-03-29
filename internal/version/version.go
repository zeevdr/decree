// Package version holds the build version injected via ldflags.
package version

// Set at build time via:
//
//	go build -ldflags "-X github.com/zeevdr/central-config-service/internal/version.Version=1.2.3
//	                    -X github.com/zeevdr/central-config-service/internal/version.Commit=abc1234"
var (
	Version = "dev"
	Commit  = "unknown"
)
