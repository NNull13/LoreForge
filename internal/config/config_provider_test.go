package config

import "testing"

func TestValidateRejectsMissingProviderFields(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Universe: UniverseConfig{Path: "./universes/example-universe"},
		Channels: ChannelsConfig{Filesystem: FilesystemChannelConfig{Enabled: true, OutputDir: "./out"}},
		Artists: []ArtistConfig{
			{
				ID:   "image-artist",
				Type: "image",
				Provider: ProviderDriver{
					Driver: "openai_image",
					Model:  "gpt-image-1.5",
				},
				Scheduler: SchedulerConfig{
					Enabled:       true,
					Mode:          "fixed_interval",
					FixedInterval: "1h",
					Timezone:      "UTC",
				},
			},
		},
	}

	if err := cfg.Validate("."); err == nil {
		t.Fatal("expected validation error for missing api_key_env")
	}
}
