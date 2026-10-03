package execution

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

type recorder struct{ lines []string }

func (r *recorder) Log(level, format string, args ...any) {
	r.lines = append(r.lines, level+": "+fmt.Sprintf(format, args...))
}

func TestRunLogsTheCommandWithoutItsValues(t *testing.T) {
	rec := &recorder{}
	r := Runner{Log: rec}
	path, err := r.LookPath("true")
	if err != nil {
		t.Skip("no true executable")
	}
	if _, err := r.Run(context.Background(), ports.Command{Path: path, Args: []string{"run", "token=s3cret"}, Env: []string{"KEY=hidden"}}); err != nil {
		t.Fatal(err)
	}
	if len(rec.lines) != 1 || rec.lines[0] != "debug: exec true run token=...: exit 0" {
		t.Fatalf("unexpected log: %q", rec.lines)
	}
	if got := strings.Join(rec.lines, ""); strings.Contains(got, "s3cret") || strings.Contains(got, "hidden") {
		t.Errorf("a value leaked into the log: %q", got)
	}
}

func TestRunWithoutALoggerIsSilent(t *testing.T) {
	r := Runner{}
	path, err := r.LookPath("true")
	if err != nil {
		t.Skip("no true executable")
	}
	if _, err := r.Run(context.Background(), ports.Command{Path: path}); err != nil {
		t.Fatal(err)
	}
}

func TestRunCapturesOutputAndExitCode(t *testing.T) {
	r := Runner{}
	sh, err := r.LookPath("sh")
	if err != nil {
		t.Skip("no sh executable")
	}
	res, err := r.Run(context.Background(), ports.Command{Path: sh, Args: []string{"-c", "printf out; printf err >&2; exit 3"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Stdout) != "out" || string(res.Stderr) != "err" || res.ExitCode != 3 {
		t.Errorf("got stdout %q, stderr %q, exit %d", res.Stdout, res.Stderr, res.ExitCode)
	}
}
