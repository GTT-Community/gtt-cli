package output

import (
	"bytes"
	"testing"
)

func TestLogWritesToStderrFromTheThresholdUp(t *testing.T) {
	for threshold, want := range map[string]string{
		"":      "error: e\nwarn: w\n",
		"error": "error: e\n",
		"debug": "error: e\nwarn: w\ninfo: i\ndebug: d\n",
	} {
		var out, errw bytes.Buffer
		r := Renderer{Out: &out, Err: &errw, LogLevel: threshold}
		r.Log("error", "e")
		r.Log("warn", "w")
		r.Log("info", "i")
		r.Log("debug", "d")
		r.Log("unknown", "x")
		if errw.String() != want {
			t.Errorf("threshold %q: got %q, want %q", threshold, errw.String(), want)
		}
		if out.Len() != 0 {
			t.Errorf("threshold %q: logs must never reach stdout: %q", threshold, out.String())
		}
	}
}
