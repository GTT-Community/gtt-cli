// Package output renders one result either for a human or as versioned JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Renderer writes results.
type Renderer struct {
	Out     io.Writer
	Err     io.Writer
	JSON    bool
	Verbose bool
	// LogLevel is the least severe level written by Log; empty means warn.
	LogLevel string
}

// LogLevels are the internal log levels, most severe first.
var LogLevels = []string{"error", "warn", "info", "debug"}

func severity(level string) int {
	for i, l := range LogLevels {
		if l == level {
			return i
		}
	}
	return -1
}

// Log writes an internal log line to stderr when level is at least as severe
// as LogLevel. Callers pass operation names and results only, never secrets.
func (r Renderer) Log(level, format string, args ...any) {
	threshold := r.LogLevel
	if threshold == "" {
		threshold = "warn"
	}
	if s := severity(level); r.Err == nil || s < 0 || s > severity(threshold) {
		return
	}
	fmt.Fprintf(r.Err, level+": "+format+"\n", args...)
}

// Emit writes v as JSON when JSON output was requested, otherwise calls
// human with a line writer.
func (r Renderer) Emit(v any, human func(p Printer)) error {
	if r.JSON {
		enc := json.NewEncoder(r.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	human(Printer{w: r.Out})
	return nil
}

// Info writes a progress line for humans; silent in JSON mode.
func (r Renderer) Info(format string, args ...any) {
	if !r.JSON {
		fmt.Fprintf(r.Out, format+"\n", args...)
	}
}

// Detail writes a line only in verbose mode.
func (r Renderer) Detail(text string) {
	if r.Verbose && !r.JSON && strings.TrimSpace(text) != "" {
		fmt.Fprintln(r.Out, strings.TrimRight(text, "\n"))
	}
}

// Printer writes human-readable lines.
type Printer struct{ w io.Writer }

// Line writes one formatted line.
func (p Printer) Line(format string, args ...any) { fmt.Fprintf(p.w, format+"\n", args...) }

// Section writes a heading followed by indented lines.
func (p Printer) Section(title string, lines ...string) {
	fmt.Fprintf(p.w, "\n%s:\n", title)
	for _, l := range lines {
		fmt.Fprintf(p.w, "  %s\n", l)
	}
}

// List joins values or says none.
func List(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

// YesNo renders a boolean.
func YesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
