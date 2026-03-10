package scheduler

import (
	"context"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/scheduling"
)

type Service struct {
	StateRepo ports.SchedulerStateRepository
}

func (s Service) NextRun(ctx context.Context, def ports.RegisteredGenerator, now time.Time) (time.Time, error) {
	state, _, err := s.EnsureState(ctx, def, now)
	if err != nil {
		return time.Time{}, err
	}
	return state.NextRunAt, nil
}

func (s Service) EnsureState(ctx context.Context, def ports.RegisteredGenerator, now time.Time) (scheduling.State, bool, error) {
	state, err := s.StateRepo.Load(ctx, def.Config.ID)
	if err != nil {
		return scheduling.State{}, false, err
	}
	if !state.NextRunAt.IsZero() {
		return state, false, nil
	}

	scheduler, err := scheduling.NewScheduler(def.Config.Scheduler)
	if err != nil {
		return scheduling.State{}, false, err
	}
	nextRun, err := scheduler.NextRun(now)
	if err != nil {
		return scheduling.State{}, false, err
	}

	bootstrapped := scheduling.State{
		LastRunAt: state.LastRunAt,
		NextRunAt: nextRun,
	}
	if err := s.StateRepo.Save(ctx, def.Config.ID, bootstrapped); err != nil {
		return scheduling.State{}, false, err
	}
	return bootstrapped, true, nil
}

func (s Service) AdvanceAfterRun(ctx context.Context, def ports.RegisteredGenerator, runAt time.Time) error {
	scheduler, err := scheduling.NewScheduler(def.Config.Scheduler)
	if err != nil {
		return err
	}
	nextRun, err := scheduler.NextRun(runAt)
	if err != nil {
		return err
	}
	runAtCopy := runAt
	return s.StateRepo.Save(ctx, def.Config.ID, scheduling.State{
		LastRunAt: &runAtCopy,
		NextRunAt: nextRun,
	})
}
