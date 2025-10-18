package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"github.com/sirupsen/logrus"
	"github.com/thiagozs/geolocation-go/models"
	"github.com/thiagozs/geolocation-go/pkg/utils"
)

var (
	ErrMaxMindLicenseMissing  = errors.New("maxmind license key not configured")
	ErrMaxMindDatabaseMissing = errors.New("maxmind database not loaded")
)

type MaxMindConfig struct {
	DatabasePath       string
	LicenseKey         string
	HTTPTimeout        time.Duration
	MinRefreshInterval time.Duration
}

type UpdateStatus struct {
	Updated bool
	Reason  string
}

type MaxMindService struct {
	mu         sync.RWMutex
	reader     *maxminddb.Reader
	log        *logrus.Entry
	cfg        MaxMindConfig
	downloader *utils.DatabaseDownloader
}

func NewMaxMindService(log *logrus.Entry, cfg MaxMindConfig) (*MaxMindService, error) {
	cfg = applyDefaults(cfg)

	service := &MaxMindService{
		log: log,
		cfg: cfg,
	}

	if cfg.LicenseKey != "" {
		service.downloader = utils.NewDatabaseDownloader(cfg.LicenseKey, cfg.DatabasePath, cfg.HTTPTimeout, cfg.MinRefreshInterval)
	}

	if err := service.reloadReader(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if service.downloader == nil {
				return nil, fmt.Errorf("open maxmind database: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPTimeout)
			defer cancel()

			if _, _, updateErr := service.downloader.EnsureLatest(ctx, true); updateErr != nil {
				return nil, fmt.Errorf("download maxmind database: %w", updateErr)
			}

			if err := service.reloadReader(); err != nil {
				return nil, fmt.Errorf("open maxmind database after download: %w", err)
			}
		} else {
			return nil, fmt.Errorf("open maxmind database: %w", err)
		}
	}

	log.WithField("path", cfg.DatabasePath).Info("maxmind database ready")

	return service, nil
}

func (m *MaxMindService) Lookup(ip net.IP) (models.Record, error) {
	if ip == nil {
		return models.Record{}, errors.New("invalid IP address")
	}

	reader := m.currentReader()
	if reader == nil {
		return models.Record{}, ErrMaxMindDatabaseMissing
	}

	var record models.Record
	if err := reader.Lookup(ip, &record); err != nil {
		return models.Record{}, err
	}

	record.IP = ip.String()
	return record, nil
}

func (m *MaxMindService) Update(ctx context.Context, force bool) (UpdateStatus, error) {
	if m.downloader == nil {
		return UpdateStatus{}, ErrMaxMindLicenseMissing
	}

	updated, reason, err := m.downloader.EnsureLatest(ctx, force)
	if err != nil {
		return UpdateStatus{}, err
	}

	if updated {
		if err := m.reloadReader(); err != nil {
			return UpdateStatus{}, err
		}
		m.log.WithField("path", m.cfg.DatabasePath).Info("maxmind database reloaded")
	}

	return UpdateStatus{
		Updated: updated,
		Reason:  reason,
	}, nil
}

func (m *MaxMindService) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.reader == nil {
		return nil
	}

	err := m.reader.Close()
	m.reader = nil
	return err
}

func (m *MaxMindService) Ready() bool {
	return m.currentReader() != nil
}

func (m *MaxMindService) DatabasePath() string {
	return m.cfg.DatabasePath
}

func (m *MaxMindService) currentReader() *maxminddb.Reader {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reader
}

func (m *MaxMindService) reloadReader() error {
	reader, err := maxminddb.Open(m.cfg.DatabasePath)
	if err != nil {
		return err
	}

	m.mu.Lock()
	oldReader := m.reader
	m.reader = reader
	m.mu.Unlock()

	if oldReader != nil {
		_ = oldReader.Close()
	}
	return nil
}

func applyDefaults(cfg MaxMindConfig) MaxMindConfig {
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "db/GeoLite2-City.mmdb"
	}

	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}

	if cfg.MinRefreshInterval <= 0 {
		cfg.MinRefreshInterval = 24 * time.Hour
	}

	return cfg
}
