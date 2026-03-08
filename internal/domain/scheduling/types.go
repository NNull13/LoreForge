package scheduling

import "time"

type Mode string

const (
	ModeRandomWindow  Mode = "random_window"
	ModeFixedInterval Mode = "fixed_interval"
)

type Config struct {
	Mode          Mode
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
