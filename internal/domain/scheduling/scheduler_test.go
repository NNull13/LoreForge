package scheduling

import (
	"testing"
	"time"
)

func TestNewSchedulerAndNextRunFixedInterval(t *testing.T) {
	t.Parallel()

	scheduler, err := NewScheduler(Config{
		Mode:          ModeFixedInterval,
		FixedInterval: 2 * time.Hour,
		Timezone:      "UTC",
	})
	if err != nil {
		t.Fatalf("NewScheduler returned error: %v", err)
	}
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	next, err := scheduler.NextRun(now)
	if err != nil {
		t.Fatalf("NextRun returned error: %v", err)
	}
	if got, want := next.Sub(now), 2*time.Hour; got != want {
		t.Fatalf("next interval = %v, want %v", got, want)
	}
}

func TestNewSchedulerAndNextRunRandomWindow(t *testing.T) {
	t.Parallel()

	scheduler, err := NewScheduler(Config{
		Mode:        ModeRandomWindow,
		MinInterval: time.Hour,
		MaxInterval: 3 * time.Hour,
		Seed:        7,
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("NewScheduler returned error: %v", err)
	}
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	next, err := scheduler.NextRun(now)
	if err != nil {
		t.Fatalf("NextRun returned error: %v", err)
	}
	if d := next.Sub(now); d < time.Hour || d > 3*time.Hour {
		t.Fatalf("next run out of range: %v", d)
	}
}

func TestNewSchedulerRejectsInvalidConfigs(t *testing.T) {
	t.Parallel()

	if _, err := NewScheduler(Config{Mode: ModeFixedInterval, FixedInterval: time.Hour, Timezone: "Mars/Olympus"}); err == nil {
		t.Fatal("expected invalid timezone error")
	}
	scheduler, err := NewScheduler(Config{Mode: ModeFixedInterval, Timezone: "UTC"})
	if err != nil {
		t.Fatalf("NewScheduler returned error: %v", err)
	}
	if _, err := scheduler.NextRun(time.Now()); err == nil {
		t.Fatal("expected invalid fixed interval error")
	}
}
