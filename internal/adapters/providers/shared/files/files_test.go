package files

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteTempAssetAndToDataURI(t *testing.T) {
	t.Parallel()

	path, err := WriteTempAsset("asset", ".txt", []byte("hello"))
	if err != nil {
		t.Fatalf("WriteTempAsset returned error: %v", err)
	}
	defer os.Remove(path)

	uri, err := ToDataURI(path)
	if err != nil {
		t.Fatalf("ToDataURI returned error: %v", err)
	}
	if !strings.HasPrefix(uri, "data:text/plain;") {
		t.Fatalf("unexpected data uri: %s", uri)
	}
}

func TestDownloadToTemp(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(strings.NewReader("png-bytes")),
		}, nil
	})}

	path, mimeType, err := DownloadToTemp(context.Background(), client, "https://example.com/asset.png", "asset", nil)
	if err != nil {
		t.Fatalf("DownloadToTemp returned error: %v", err)
	}
	defer os.Remove(path)

	if mimeType != "image/png" {
		t.Fatalf("mime type = %q, want image/png", mimeType)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(content) != "png-bytes" {
		t.Fatalf("unexpected downloaded content: %q", content)
	}
}

func TestGSHelpers(t *testing.T) {
	t.Parallel()

	bucket, object, err := SplitGSURI("gs://bucket/object.txt")
	if err != nil {
		t.Fatalf("SplitGSURI returned error: %v", err)
	}
	if bucket != "bucket" || object != "object.txt" {
		t.Fatalf("unexpected split: %s %s", bucket, object)
	}
	url, err := GCSMediaURLWithBase("gs://bucket/object.txt", "https://storage.googleapis.com")
	if err != nil {
		t.Fatalf("GCSMediaURLWithBase returned error: %v", err)
	}
	if !strings.Contains(url, "/storage/v1/b/bucket/o/object.txt?alt=media") {
		t.Fatalf("unexpected media url: %s", url)
	}
}

func TestWriteBase64TempAndFallbackMime(t *testing.T) {
	t.Parallel()

	path, err := WriteBase64Temp("asset", "text/plain", base64.StdEncoding.EncodeToString([]byte("hello")))
	if err != nil {
		t.Fatalf("WriteBase64Temp returned error: %v", err)
	}
	defer os.Remove(path)

	if filepath.Ext(path) == "" {
		t.Fatalf("expected temp file extension, got %s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected decoded content: %q", content)
	}

	unknown := filepath.Join(t.TempDir(), "payload.unknown")
	if err := os.WriteFile(unknown, []byte("raw"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	uri, err := ToDataURI(unknown)
	if err != nil {
		t.Fatalf("ToDataURI returned error: %v", err)
	}
	if !strings.HasPrefix(uri, "data:application/octet-stream;") {
		t.Fatalf("unexpected fallback data uri: %s", uri)
	}
}

func TestDownloadToTempAndGSHelpersRejectInvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := WriteBase64Temp("asset", "text/plain", "***"); err == nil {
		t.Fatal("expected invalid base64 error")
	}

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer test" {
			t.Fatalf("missing forwarded header: %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       io.NopCloser(strings.NewReader("blocked")),
		}, nil
	})}

	if _, _, err := DownloadToTemp(context.Background(), client, "https://example.com/blocked", "asset", map[string]string{"Authorization": "Bearer test"}); err == nil {
		t.Fatal("expected download failure on non-2xx response")
	}
	if _, _, err := SplitGSURI("https://storage.googleapis.com/bucket/object"); err == nil {
		t.Fatal("expected invalid gs uri error")
	}
	if _, err := GCSMediaURL("gs://bucket"); err == nil {
		t.Fatal("expected invalid media url error")
	}
	if got := extFromMIME("application/x-custom"); got != ".bin" {
		t.Fatalf("extFromMIME fallback = %q, want .bin", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
