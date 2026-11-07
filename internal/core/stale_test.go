package core

import (
	"testing"
	"time"
)

func TestStaleDays(t *testing.T) {
	old := BuildDate
	defer func() { BuildDate = old }()
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	BuildDate = "2026-10-13T10:00:00Z"
	if d, stale := StaleDays(now); d != 25 || !stale {
		t.Fatalf("a 25-day-old build must be stale, got %d %v", d, stale)
	}
	BuildDate = "2026-11-01"
	if _, stale := StaleDays(now); stale {
		t.Fatal("a 6-day-old build must not be stale")
	}
}
