// Package core holds the CLI's own identity and its exit and error contract.
// It depends on nothing but the standard library.
package core

// Version is the CLI release. It is independent of the Bootstrap version;
// compatibility between the two is negotiated, never assumed.
var Version = "1.0.0"

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
