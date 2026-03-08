package timeutil

import (
	"testing"
	"time"
)

func TestRealClockNow(t *testing.T) {
	t.Parallel()

	before := time.Now().Add(-time.Second)
	got := RealClock{}.Now()
	after := time.Now().Add(time.Second)
	if got.Before(before) || got.After(after) {
		t.Fatalf("unexpected clock value: %s", got)
	}
}
