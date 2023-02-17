package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
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

type DatabaseDownloader struct {
	LicenseKey        string
	TargetFilePath    string
	localChecksumPath string
	DownloadURL       string
	ChecksumURL       string
	httpClient        *http.Client
}

func NewDatabaseDownloader(licenseKey string, targetFilePath string, timeout time.Duration) *DatabaseDownloader {
	return &DatabaseDownloader{
		LicenseKey:        licenseKey,
		TargetFilePath:    targetFilePath,
		localChecksumPath: targetFilePath + DefaultChecksumExt,
		DownloadURL:       DefaultDownloadURL,
		ChecksumURL:       DefaultChecksumURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
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

	return string(localChecksum), nil

}

func (downloader *DatabaseDownloader) RemoteChecksum() (string, error) {

	resp, err := downloader.doGETRequest(downloader.ChecksumURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(result), nil

}

func (downloader *DatabaseDownloader) ShouldDownload() (bool, error) {

	localChecksum, err := downloader.LocalChecksum()
	if err != nil {
		return false, err
	}

	remoteChecksum, err := downloader.RemoteChecksum()
	if err != nil {
		return false, err
	}

	return remoteChecksum != localChecksum, nil

}

func (downloader *DatabaseDownloader) Download() error {

	resp, err := downloader.doGETRequest(downloader.DownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	uncompressedStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

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

		targetFileDir := filepath.Dir(downloader.TargetFilePath)
		if !downloader.fileExists(targetFileDir) {
			if err := os.MkdirAll(targetFileDir, 0755); err != nil {
				return err
			}
		}

		outFile, err := os.Create(downloader.TargetFilePath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, tarReader)
		if err != nil {
			return err
		}

		remoteChecksum, err := downloader.RemoteChecksum()
		if err != nil {
			return err
		}

		if err := os.WriteFile(downloader.localChecksumPath, []byte(remoteChecksum), 0666); err != nil {
			return err
		}

		foundFile = true

	}

	if foundFile {
		return nil
	}

	return errors.New("invalid download, tgz doesn't contain a .mmdb file")

}

func (downloader *DatabaseDownloader) doGETRequest(urlString string) (*http.Response, error) {

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	q := parsedURL.Query()
	q.Set("edition_id", "GeoLite2-City")
	q.Set("license_key", downloader.LicenseKey)
	parsedURL.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
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

	if resp.StatusCode == 401 {
		return nil, errors.New("invalid license key")
	}

	return resp, nil

}

func (downloader *DatabaseDownloader) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
