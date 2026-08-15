// Package version holds build-time version information injected via ldflags.
package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("schema-api %s (commit %s, built %s)", Version, Commit, BuildDate)
}
