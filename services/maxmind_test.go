package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewMaxMindServiceWithoutDatabaseOrLicenseFails(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := MaxMindConfig{
		DatabasePath: filepath.Join(t.TempDir(), "GeoLite2-City.mmdb"),
	}

	if _, err := NewMaxMindService(log, cfg); err == nil {
		t.Fatalf("expected error when database file is missing without license")
	}
}

func TestMaxMindServiceUpdateWithoutLicense(t *testing.T) {
	dbPath := filepath.Join("db", "GeoLite2-City.mmdb")
	if _, err := os.Stat(dbPath); err != nil {
		t.Skipf("maxmind database not available: %v", err)
	}

	log := logrus.NewEntry(logrus.New())
	cfg := MaxMindConfig{
		DatabasePath: dbPath,
	}

	svc, err := NewMaxMindService(log, cfg)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}
	defer svc.Close()

	if !svc.Ready() {
		t.Fatalf("expected service to be ready after loading existing database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	status, err := svc.Update(ctx, false)
	if err == nil {
		t.Fatalf("expected error when updating without license, got status: %#v", status)
	}

	if err != ErrMaxMindLicenseMissing {
		t.Fatalf("expected ErrMaxMindLicenseMissing, got %v", err)
	}
}
