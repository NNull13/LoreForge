package scheduler

import (
	"fmt"
	"math/rand"
	"time"
)

type Config struct {
	Mode          string
	MinInterval   time.Duration
	MaxInterval   time.Duration
	FixedInterval time.Duration
	Seed          int64
	Timezone      string
}

type State struct {
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	NextRunAt time.Time  `json:"next_run_at"`
}

type Scheduler struct {
	rng *rand.Rand
	cfg Config
	loc *time.Location
}

func New(cfg Config) (*Scheduler, error) {
	if cfg.Mode == "" {
		cfg.Mode = "random_window"
	}
	if cfg.Seed == 0 {
		cfg.Seed = time.Now().UnixNano()
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, err
	}
	return &Scheduler{rng: rand.New(rand.NewSource(cfg.Seed)), cfg: cfg, loc: loc}, nil
}

func (s *Scheduler) NextRun(now time.Time) (time.Time, error) {
	n := now.In(s.loc)
	switch s.cfg.Mode {
	case "fixed_interval":
		if s.cfg.FixedInterval <= 0 {
			return time.Time{}, fmt.Errorf("fixed_interval must be > 0")
		}
		return n.Add(s.cfg.FixedInterval), nil
	case "random_window":
		if s.cfg.MinInterval <= 0 || s.cfg.MaxInterval <= 0 || s.cfg.MaxInterval < s.cfg.MinInterval {
			return time.Time{}, fmt.Errorf("invalid random window")
		}
		delta := s.cfg.MaxInterval - s.cfg.MinInterval
		if delta == 0 {
			return n.Add(s.cfg.MinInterval), nil
		}
		jitter := time.Duration(s.rng.Int63n(int64(delta)))
		return n.Add(s.cfg.MinInterval + jitter), nil
	default:
		return time.Time{}, fmt.Errorf("unknown mode %s", s.cfg.Mode)
	}
}
