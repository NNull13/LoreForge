package auth

import (
	"fmt"
	"os"
	"strings"

	"loreforge/internal/adapters/providers/shared/provider_errors"
)

func BearerTokenFromEnv(envVar string) (string, error) {
	token := strings.TrimSpace(os.Getenv(envVar))
	if token == "" {
		return "", fmt.Errorf("%w: env %s is empty", provider_errors.ErrProviderMisconfigured, envVar)
	}
	return token, nil
}

func GoogleAccessToken() (string, error) {
	for _, name := range []string{"GOOGLE_CLOUD_ACCESS_TOKEN", "VERTEX_AI_ACCESS_TOKEN"} {
		token := strings.TrimSpace(os.Getenv(name))
		if token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("%w: GOOGLE_CLOUD_ACCESS_TOKEN or VERTEX_AI_ACCESS_TOKEN is required", provider_errors.ErrProviderMisconfigured)
}

func RequiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%w: env %s is empty", provider_errors.ErrProviderMisconfigured, name)
	}
	return value, nil
}
