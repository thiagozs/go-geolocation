package utils

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultDownloadURL = "https://download.maxmind.com/app/geoip_download?suffix=tar.gz"

const DefaultChecksumURL = "https://download.maxmind.com/app/geoip_download?suffix=tar.gz.sha256"

const DefaultChecksumExt = ".sha256"

const defaultHTTPTimeout = 30 * time.Second

type DatabaseDownloader struct {
	LicenseKey         string
	TargetFilePath     string
	localChecksumPath  string
	DownloadURL        string
	ChecksumURL        string
	httpClient         *http.Client
	MinRefreshInterval time.Duration
}

func NewDatabaseDownloader(licenseKey, targetFilePath string, timeout, minRefresh time.Duration) *DatabaseDownloader {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &DatabaseDownloader{
		LicenseKey:         licenseKey,
		TargetFilePath:     targetFilePath,
		localChecksumPath:  targetFilePath + DefaultChecksumExt,
		DownloadURL:        DefaultDownloadURL,
		ChecksumURL:        DefaultChecksumURL,
		httpClient:         &http.Client{Timeout: timeout},
		MinRefreshInterval: minRefresh,
	}
}

func (downloader *DatabaseDownloader) LocalChecksum() (string, error) {
	if !downloader.fileExists(downloader.TargetFilePath) {
		return "", nil
	}

	if !downloader.fileExists(downloader.localChecksumPath) {
		return "", nil
	}

	localChecksum, err := os.ReadFile(downloader.localChecksumPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(localChecksum)), nil
}

func (downloader *DatabaseDownloader) RemoteChecksum(ctx context.Context) (string, error) {
	resp, err := downloader.doGETRequest(ctx, downloader.ChecksumURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected checksum status code: %d", resp.StatusCode)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(result)), nil
}

func (downloader *DatabaseDownloader) ShouldDownload(ctx context.Context) (bool, string, error) {
	localChecksum, err := downloader.LocalChecksum()
	if err != nil {
		return false, "", err
	}

	remoteChecksum, err := downloader.RemoteChecksum(ctx)
	if err != nil {
		return false, "", err
	}

	if localChecksum == "" {
		return true, remoteChecksum, nil
	}

	return !strings.EqualFold(remoteChecksum, localChecksum), remoteChecksum, nil
}

func (downloader *DatabaseDownloader) EnsureLatest(ctx context.Context, force bool) (bool, string, error) {
	if force {
		if err := downloader.download(ctx, ""); err != nil {
			return false, "", err
		}
		return true, "force update requested", nil
	}

	if !downloader.fileExists(downloader.TargetFilePath) {
		if err := downloader.download(ctx, ""); err != nil {
			return false, "", err
		}
		return true, "database file missing", nil
	}

	if downloader.MinRefreshInterval > 0 {
		info, err := os.Stat(downloader.TargetFilePath)
		if err != nil {
			return false, "", err
		}
		age := time.Since(info.ModTime())
		if age < downloader.MinRefreshInterval {
			return false, fmt.Sprintf("last update %s ago, minimum refresh window %s", age.Round(time.Second), downloader.MinRefreshInterval), nil
		}
	}

	shouldDownload, remoteChecksum, err := downloader.ShouldDownload(ctx)
	if err != nil {
		return false, "", err
	}

	if !shouldDownload {
		return false, "database already up to date", nil
	}

	if err := downloader.download(ctx, remoteChecksum); err != nil {
		return false, "", err
	}

	return true, "remote checksum changed", nil
}

func (downloader *DatabaseDownloader) download(ctx context.Context, remoteChecksum string) error {
	resp, err := downloader.doGETRequest(ctx, downloader.DownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected download status code: %d", resp.StatusCode)
	}

	uncompressedStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)
	foundFile := false

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if !strings.HasSuffix(header.Name, ".mmdb") {
			continue
		}

		if err := downloader.ensureTargetDir(); err != nil {
			return err
		}

		tmpFile, err := os.CreateTemp(filepath.Dir(downloader.TargetFilePath), "geoip-*.mmdb")
		if err != nil {
			return err
		}

		tmpPath := tmpFile.Name()

		if _, err := io.Copy(tmpFile, tarReader); err != nil {
			tmpFile.Close()
			_ = os.Remove(tmpPath)
			return err
		}

		if err := tmpFile.Close(); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}

		if err := replaceFile(tmpPath, downloader.TargetFilePath); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}

		foundFile = true
		break
	}

	if !foundFile {
		return errors.New("invalid download, tgz doesn't contain a .mmdb file")
	}

	if remoteChecksum == "" {
		checksum, err := downloader.RemoteChecksum(ctx)
		if err != nil {
			return err
		}
		remoteChecksum = checksum
	}

	if err := os.WriteFile(downloader.localChecksumPath, []byte(remoteChecksum+"\n"), 0o644); err != nil {
		return err
	}

	return nil
}

func (downloader *DatabaseDownloader) doGETRequest(ctx context.Context, urlString string) (*http.Response, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	q := parsedURL.Query()
	q.Set("edition_id", "GeoLite2-City")
	q.Set("license_key", downloader.LicenseKey)
	parsedURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Encoding", "")
	req.Header.Set("Connection", "close")
	req.Header.Set("Accept-Encoding", "deflate, identity")

	resp, err := downloader.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, errors.New("invalid license key")
	}

	return resp, nil
}

func (downloader *DatabaseDownloader) ensureTargetDir() error {
	targetFileDir := filepath.Dir(downloader.TargetFilePath)
	if downloader.fileExists(targetFileDir) {
		return nil
	}
	return os.MkdirAll(targetFileDir, 0o755)
}

func (downloader *DatabaseDownloader) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func replaceFile(tmpPath, target string) error {
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpPath, target)
}
