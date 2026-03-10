package http_client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientJSONSuccess(t *testing.T) {
	t.Parallel()

	client := New(0)
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("X-Test"); got != "ok" {
			t.Fatalf("missing header: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		if !strings.Contains(string(body), `"a":1`) {
			t.Fatalf("unexpected request body: %s", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	})}

	resp, body, err := client.JSON(context.Background(), http.MethodPost, "https://api.test/resource", map[string]string{"X-Test": "ok"}, map[string]any{"a": 1})
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want 200", resp.StatusCode)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestClientJSONErrorStatusReturnsBody(t *testing.T) {
	t.Parallel()

	client := New(0)
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       io.NopCloser(strings.NewReader("nope\n")),
		}, nil
	})}
	_, body, err := client.JSON(context.Background(), http.MethodGet, "https://api.test/resource", nil, nil)
	if err == nil {
		t.Fatal("expected status error")
	}
	if string(body) == "" {
		t.Fatal("expected response body on error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
