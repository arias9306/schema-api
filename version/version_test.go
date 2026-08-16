package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	assert.Equal(t, "schema-api dev (commit none, built unknown)", String())
}

func TestStringWithBuildInfo(t *testing.T) {
	Version = "v1.2.3"
	Commit = "abc123"
	BuildDate = "2026-01-02T03:04:05Z"

	assert.Equal(t, "schema-api v1.2.3 (commit abc123, built 2026-01-02T03:04:05Z)", String())
}
