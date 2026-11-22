package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// A human act that writes the instruction plane (.gtt/scripts, .claude/hooks, the frozen design) records its window, so the sentinel
// of an agent session that is running in the same project tells it from the agent's own command: a path changed INSIDE the window is
// the human's and is kept; one changed outside it is put back. `gtt update` writes a hundred of those files for half a minute. Without
// the window, an agent command that ran meanwhile read the new scripts and hooks as its own, put some of them back, and left the project
// half updated: the new contracts beside the old scripts, and the ledger ahead of the files.
//
// The format and the place are the Bootstrap's (gtt_act.begin_act, governed-sentinel.human_window): a JSON record under
// <cache>/gtt/sentinel/<sha256 of the real project path, first 16>/human-act.json. The CLI writes it itself so that it does not depend
// on the release installed in the project, which may be the one that is being replaced.

// sentinelDir is where the sentinel keeps its state for a project: the same formula as the Bootstrap, outside the project.
func sentinelDir(root string) string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("XDG_CACHE_HOME")
	}
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".cache")
	}
	real := root
	if r, err := filepath.EvalSymlinks(root); err == nil {
		real = r
	}
	if abs, err := filepath.Abs(real); err == nil {
		real = abs
	}
	sum := sha256.Sum256([]byte(real))
	return filepath.Join(base, "gtt", "sentinel", hex.EncodeToString(sum[:])[:16])
}

func writeLease(where string, record map[string]any) error {
	if err := os.MkdirAll(where, 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	tmp := filepath.Join(where, "human-act.json.tmp")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(where, "human-act.json"))
}

// beginHumanAct opens the window for the act and returns the function that closes it. An act is never blocked by the bookkeeping: when
// the window cannot be recorded, the act runs as before. A window already open (GTT_HUMAN_ACT is set: an outer act owns it) is shared.
// The variable is set for the scripts this process runs, so that none of them opens its own window and closes this one early.
func beginHumanAct(root, act string) func() {
	if os.Getenv("GTT_HUMAN_ACT") != "" {
		return func() {}
	}
	where := sentinelDir(root)
	started := float64(time.Now().UnixNano()) / 1e9
	if err := writeLease(where, map[string]any{"pid": os.Getpid(), "act": act, "started": started, "ended": nil}); err != nil {
		return func() {}
	}
	os.Setenv("GTT_HUMAN_ACT", act)
	return func() {
		os.Unsetenv("GTT_HUMAN_ACT")
		_ = writeLease(where, map[string]any{"pid": os.Getpid(), "act": act, "started": started, "ended": float64(time.Now().UnixNano()) / 1e9})
	}
}
