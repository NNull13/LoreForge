package factory

import (
	"fmt"
	"strings"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/adapters/providers/mock"
	"loreforge/internal/config"
)

func NewTextProvider(cfg config.ProviderDriver) (providercontracts.TextProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.TextProvider{Model: cfg.Model}, nil
	default:
		return nil, fmt.Errorf("unsupported text provider driver: %s", cfg.Driver)
	}
}

func NewVideoProvider(cfg config.ProviderDriver) (providercontracts.VideoProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.VideoProvider{Model: cfg.Model}, nil
	default:
		return nil, fmt.Errorf("unsupported video provider driver: %s", cfg.Driver)
	}
}

func NewImageProvider(cfg config.ProviderDriver) (providercontracts.ImageProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.ImageProvider{Model: cfg.Model}, nil
	default:
		return nil, fmt.Errorf("unsupported image provider driver: %s", cfg.Driver)
	}
}

func normalizeDriver(driver string) string {
	value := strings.TrimSpace(strings.ToLower(driver))
	if value == "" {
		return "mock"
	}
	return value
}
