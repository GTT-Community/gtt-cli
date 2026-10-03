// Package core holds the CLI's own identity and its exit and error contract.
// It depends on nothing but the standard library.
package core

import (
	"runtime/debug"
	"strings"
)

// Version is the CLI release. It is independent of the Bootstrap version;
// compatibility between the two is negotiated, never assumed. Release builds
// set it with -ldflags; `go install module@vX.Y.Z` takes it from the module
// version; a plain source build keeps this base value.
var Version = baseVersion

// baseVersion is the version of a plain source build. It must stay a
// constant: -ldflags -X only replaces constant-initialized strings.
const baseVersion = "1.0.1"

func init() {
	if Version == baseVersion {
		Version = moduleVersion(baseVersion)
	}
}

// moduleVersion returns the version Go recorded for this module when it was
// installed from a tagged release, or base otherwise.
func moduleVersion(base string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return base
	}
	v := strings.TrimPrefix(info.Main.Version, "v")
	if _, ok := ParseSemver(v); !ok {
		return base // "(devel)" or a pseudo-version
	}
	return v
}

// CLICapabilities are the capability ids this CLI declares to a Bootstrap
// during negotiation.
var CLICapabilities = []string{"contract.negotiate.v1", "operation.execute.v1"}

// BootstrapSchemas are the Bootstrap schema versions this CLI understands.
var BootstrapSchemas = []int{1}

// BootstrapID is the only Bootstrap identity this CLI operates.
const BootstrapID = "gtt-bootstrap"

// SupportedCapabilities maps each Bootstrap capability this CLI consumes to
// the version it implements. A Bootstrap that offers one of these at another
// version is refused; a capability not listed here is simply not used.
var SupportedCapabilities = map[string]int{
	"project.detect":       1,
	"ade.detect":           1,
	"ade.install":          1,
	"template.materialize": 1,
	"methodology.profile":  1,
	"source.select":        1,
	"validation":           1,
	"status":               1,
	"freeze":               1,
	"session-context":      1,
	"export-policy":        1,
	"recovery":             1,
	"guard":                1,
	"retrieval":            1,
	"developer-experience": 1,
}

// OutputSchema is the version of every JSON document the CLI emits.
const OutputSchema = 1
