package output

import (
	"bytes"
	"fmt"
	"strings"
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

type tagged struct{}

func (tagged) Sprintf(format string, args ...any) string { return fmt.Sprintf("es:"+format, args...) }
func (tagged) T(message string) string                   { return "es:" + message }

func TestHumanLinesAreTranslatedAndJSONIsNot(t *testing.T) {
	var out bytes.Buffer
	r := Renderer{Out: &out, Lang: tagged{}}
	r.Info("step %s", "x")
	r.Emit(map[string]string{"k": "v"}, func(p Printer) { p.Line("line %d", 1); p.Section("title", "item") })
	if got := out.String(); got != "es:step x\nes:line 1\n\nes:title:\n  es:item\n" {
		t.Errorf("got %q", got)
	}
	out.Reset()
	r.JSON = true
	r.Emit(map[string]string{"k": "v"}, func(p Printer) {})
	if !strings.Contains(out.String(), `"k": "v"`) {
		t.Errorf("JSON is never translated: %q", out.String())
	}
}
