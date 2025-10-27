package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnsureLatestDownloadsWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "GeoLite2-City.mmdb")
	checksumPath := targetPath + DefaultChecksumExt

	checksumValue := "checksum-initial"
	payload := []byte("dummy maxmind payload v1")

	downloader := NewDatabaseDownloader("license-key", targetPath, time.Second, 0)
	downloader.DownloadURL = "https://example.com/download"
	downloader.ChecksumURL = "https://example.com/checksum"
	downloader.httpClient = newMockHTTPClient(&checksumValue, &payload)

	updated, reason, err := downloader.EnsureLatest(context.Background(), false)
	if err != nil {
		t.Fatalf("EnsureLatest returned error: %v", err)
	}

	if !updated {
		t.Fatalf("expected download to occur, reason: %s", reason)
	}

	verifyFileContent(t, targetPath, payload)
	verifyFileContent(t, checksumPath, []byte(checksumValue+"\n"))

	// second call with same checksum should skip download
	updated, reason, err = downloader.EnsureLatest(context.Background(), false)
	if err != nil {
		t.Fatalf("EnsureLatest second call error: %v", err)
	}

	if updated {
		t.Fatalf("expected no update on second call, got reason: %s", reason)
	}

	if !strings.Contains(reason, "already up to date") {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestEnsureLatestRespectsRefreshIntervalAndForce(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "GeoLite2-City.mmdb")

	checksumValue := "checksum-v1"
	payload := []byte("payload v1")

	downloader := NewDatabaseDownloader("license-key", targetPath, time.Second, time.Hour)
	downloader.DownloadURL = "https://example.com/download"
	downloader.ChecksumURL = "https://example.com/checksum"
	downloader.httpClient = newMockHTTPClient(&checksumValue, &payload)

	if updated, _, err := downloader.EnsureLatest(context.Background(), false); err != nil || !updated {
		t.Fatalf("initial download failed: updated=%v err=%v", updated, err)
	}

	checksumValue = "checksum-v2"
	payload = []byte("payload v2")

	updated, reason, err := downloader.EnsureLatest(context.Background(), false)
	if err != nil {
		t.Fatalf("EnsureLatest error: %v", err)
	}

	if updated {
		t.Fatalf("expected skip due to refresh window, got reason: %s", reason)
	}

	if !strings.Contains(reason, "minimum refresh window") {
		t.Fatalf("expected refresh window message, got: %s", reason)
	}

	updated, reason, err = downloader.EnsureLatest(context.Background(), true)
	if err != nil {
		t.Fatalf("force EnsureLatest error: %v", err)
	}

	if !updated {
		t.Fatalf("expected force update, reason: %s", reason)
	}

	verifyFileContent(t, targetPath, payload)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newMockHTTPClient(checksum *string, payload *[]byte) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/checksum":
				body := io.NopCloser(strings.NewReader(fmt.Sprintf("%s\n", *checksum)))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       body,
					Header:     make(http.Header),
				}, nil
			case "/download":
				data, err := buildTarArchive("GeoLite2-City.mmdb", *payload)
				if err != nil {
					return nil, err
				}

				body := io.NopCloser(bytes.NewReader(data))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       body,
					Header:     make(http.Header),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
		}),
	}
}

func buildTarArchive(name string, payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	hdr := &tar.Header{
		Name: name,
		Mode: 0o600,
		Size: int64(len(payload)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}

	if _, err := tw.Write(payload); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	if err := gzw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func verifyFileContent(t *testing.T, path string, expected []byte) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s failed: %v", path, err)
	}

	if !bytes.Equal(data, expected) {
		t.Fatalf("unexpected content for %s: got %q want %q", path, string(data), string(expected))
	}
}
