package polling

import (
	"context"
	"time"
)

func Until(ctx context.Context, interval time.Duration, fn func(context.Context) (bool, error)) error {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		done, err := fn(ctx)
		if done || err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
