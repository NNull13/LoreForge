package auth

import "testing"

func TestBearerTokenFromEnv(t *testing.T) {
	t.Setenv("TOKEN_ENV", "secret")
	token, err := BearerTokenFromEnv("TOKEN_ENV")
	if err != nil {
		t.Fatalf("BearerTokenFromEnv returned error: %v", err)
	}
	if token != "secret" {
		t.Fatalf("token = %q, want secret", token)
	}
}

func TestRequiredEnvAndGoogleAccessTokenErrors(t *testing.T) {
	t.Parallel()

	if _, err := RequiredEnv("MISSING_ENV"); err == nil {
		t.Fatal("expected RequiredEnv error")
	}
	if _, err := GoogleAccessToken(); err == nil {
		t.Fatal("expected GoogleAccessToken error")
	}
}

func TestGoogleAccessTokenUsesFallbackEnv(t *testing.T) {
	t.Setenv("VERTEX_AI_ACCESS_TOKEN", "vertex-token")
	token, err := GoogleAccessToken()
	if err != nil {
		t.Fatalf("GoogleAccessToken returned error: %v", err)
	}
	if token != "vertex-token" {
		t.Fatalf("token = %q, want vertex-token", token)
	}
}
