package polling

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUntilStopsWhenDone(t *testing.T) {
	t.Parallel()

	calls := 0
	err := Until(context.Background(), time.Millisecond, func(context.Context) (bool, error) {
		calls++
		return calls >= 2, nil
	})
	if err != nil {
		t.Fatalf("Until returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestUntilPropagatesErrors(t *testing.T) {
	t.Parallel()

	want := errors.New("boom")
	err := Until(context.Background(), time.Millisecond, func(context.Context) (bool, error) {
		return false, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestUntilHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := Until(ctx, time.Millisecond, func(context.Context) (bool, error) {
		return false, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
}
