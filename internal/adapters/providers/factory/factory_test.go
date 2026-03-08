package factory

import (
	"testing"

	"loreforge/internal/config"
)

func TestNewImageProviderSupportsRealDrivers(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"mock", "openai_image", "vertex_imagen"} {
		t.Run(driver, func(t *testing.T) {
			_, err := NewImageProvider(config.ProviderDriver{Driver: driver, Model: "test-model"})
			if err != nil {
				t.Fatalf("NewImageProvider returned error: %v", err)
			}
		})
	}
}

func TestNewVideoProviderSupportsRealDrivers(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"mock", "vertex_veo", "runway_gen4"} {
		t.Run(driver, func(t *testing.T) {
			_, err := NewVideoProvider(config.ProviderDriver{Driver: driver, Model: "test-model"})
			if err != nil {
				t.Fatalf("NewVideoProvider returned error: %v", err)
			}
		})
	}
}

func TestNewProviderRejectsUnsupportedDriver(t *testing.T) {
	t.Parallel()

	if _, err := NewImageProvider(config.ProviderDriver{Driver: "nope"}); err == nil {
		t.Fatal("expected error for unsupported image driver")
	}
	if _, err := NewVideoProvider(config.ProviderDriver{Driver: "nope"}); err == nil {
		t.Fatal("expected error for unsupported video driver")
	}
}
