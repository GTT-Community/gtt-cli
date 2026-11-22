package app

import (
	"path/filepath"

	"github.com/GTT-Community/gtt-cli/internal/fsx"
)

// putBackBySentinel reports whether the sentinel of an agent session archived, as a write it reverted, exactly this content of this file:
// what it keeps under gtt-domain/proposals/withdrawn/<time>-reverted/ is what the command wrote, never lost. When that is the version an
// update installs, the file in the project is the old one the sentinel restored, not a change of the person's.
func putBackBySentinel(root, rel, want string) bool {
	matches, _ := filepath.Glob(filepath.Join(root, "gtt-domain", "proposals", "withdrawn", "*-reverted*", filepath.FromSlash(rel)))
	for _, m := range matches {
		if sum, err := fsx.HashFile(m); err == nil && sum == want {
			return true
		}
	}
	return false
}
