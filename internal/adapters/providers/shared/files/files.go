package files

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"loreforge/internal/adapters/providers/shared/provider_errors"
)

func WriteTempAsset(prefix, extension string, content []byte) (string, error) {
	if extension == "" {
		extension = ".bin"
	}
	file, err := os.CreateTemp("", prefix+"-*"+extension)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func WriteBase64Temp(prefix, mimeType, encoded string) (string, error) {
	content, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return WriteTempAsset(prefix, extFromMIME(mimeType), content)
}

func DownloadToTemp(ctx context.Context, client *http.Client, sourceURL, prefix string, headers map[string]string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", "", err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", provider_errors.ErrAssetDownloadFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("%w: status %d", provider_errors.ErrAssetDownloadFailed, resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	mimeType := resp.Header.Get("Content-Type")
	path, err := WriteTempAsset(prefix, extFromMIME(mimeType), content)
	return path, mimeType, err
}

func ToDataURI(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(content), nil
}

func SplitGSURI(uri string) (bucket string, object string, err error) {
	if !strings.HasPrefix(uri, "gs://") {
		return "", "", fmt.Errorf("invalid gs uri: %s", uri)
	}
	rest := strings.TrimPrefix(uri, "gs://")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid gs uri: %s", uri)
	}
	return parts[0], parts[1], nil
}

func GCSMediaURL(uri string) (string, error) {
	return GCSMediaURLWithBase(uri, "https://storage.googleapis.com")
}

func GCSMediaURLWithBase(uri string, base string) (string, error) {
	bucket, object, err := SplitGSURI(uri)
	if err != nil {
		return "", err
	}
	base = strings.TrimRight(base, "/")
	return base + "/storage/v1/b/" + url.PathEscape(bucket) + "/o/" + url.PathEscape(object) + "?alt=media", nil
}

func extFromMIME(mimeType string) string {
	if exts, _ := mime.ExtensionsByType(strings.TrimSpace(strings.Split(mimeType, ";")[0])); len(exts) > 0 {
		return exts[0]
	}
	return ".bin"
}
