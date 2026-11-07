package core

import (
	"runtime/debug"
	"time"
)

// BuildDate is the day this binary was built (YYYY-MM-DD). Release builds set it with -ldflags; a plain
// source build leaves it empty and falls back to the VCS time Go recorded, if any.
var BuildDate = ""

// StaleAfterDays is how old a CLI build may be before it says so. GTT releases often; a CLI that is
// weeks old runs a Bootstrap that lacks what the newer ones carry.
const StaleAfterDays = 21

func buildTime() (time.Time, bool) {
	if len(BuildDate) >= 10 {
		if t, err := time.Parse("2006-01-02", BuildDate[:10]); err == nil {
			return t, true
		}
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.time" && len(s.Value) >= 10 {
				if t, err := time.Parse("2006-01-02", s.Value[:10]); err == nil {
					return t, true
				}
			}
		}
	}
	return time.Time{}, false
}

// StaleDays reports how many days old this build is, and whether that is past StaleAfterDays.
// It uses no network: the age of the binary is a fact the binary knows.
func StaleDays(now time.Time) (int, bool) {
	built, ok := buildTime()
	if !ok {
		return 0, false
	}
	days := int(now.Sub(built).Hours() / 24)
	return days, days > StaleAfterDays
}
