package scheduler

import (
	"testing"
	"time"
)

func TestNextRun_FixedInterval(t *testing.T) {
	s, err := New(Config{
		Mode:          "fixed_interval",
		FixedInterval: 2 * time.Hour,
		Seed:          1,
		Timezone:      "UTC",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	now := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)
	next, err := s.NextRun(now)
	if err != nil {
		t.Fatalf("NextRun failed: %v", err)
	}
	if got, want := next.Sub(now), 2*time.Hour; got != want {
		t.Fatalf("unexpected interval: got %v, want %v", got, want)
	}
}

func TestNextRun_RandomWindowBounds(t *testing.T) {
	s, err := New(Config{
		Mode:        "random_window",
		MinInterval: 1 * time.Hour,
		MaxInterval: 3 * time.Hour,
		Seed:        99,
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	now := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		next, err := s.NextRun(now)
		if err != nil {
			t.Fatalf("NextRun failed: %v", err)
		}
		d := next.Sub(now)
		if d < time.Hour || d > 3*time.Hour {
			t.Fatalf("interval out of bounds: %v", d)
		}
	}
}
