package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/thiagozs/geolocation-go/models"
	"github.com/thiagozs/geolocation-go/services"
)

type fakeGeoIP struct {
	record       models.Record
	lookupErr    error
	ready        bool
	dbPath       string
	updateStatus services.UpdateStatus
	updateErr    error
	updateCalls  []bool
	lastLookupIP net.IP
	closeCalls   int
}

func (f *fakeGeoIP) Lookup(ip net.IP) (models.Record, error) {
	f.lastLookupIP = ip
	if f.lookupErr != nil {
		return models.Record{}, f.lookupErr
	}
	return f.record, nil
}

func (f *fakeGeoIP) Update(_ context.Context, force bool) (services.UpdateStatus, error) {
	f.updateCalls = append(f.updateCalls, force)
	if f.updateErr != nil {
		return services.UpdateStatus{}, f.updateErr
	}
	return f.updateStatus, nil
}

func (f *fakeGeoIP) Ready() bool {
	return f.ready
}

func (f *fakeGeoIP) DatabasePath() string {
	if f.dbPath == "" {
		return "/tmp/db.mmdb"
	}
	return f.dbPath
}

func (f *fakeGeoIP) Close() error {
	f.closeCalls++
	return nil
}

func newTestServer(t *testing.T, svc GeoIPService) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	s := &Server{
		cfg: Config{
			HTTPPort: 5000,
			Mode:     "development",
			GeoIP: services.MaxMindConfig{
				HTTPTimeout: 200 * time.Millisecond,
			},
		},
		geoIP: svc,
		log:   logrus.NewEntry(logrus.New()),
	}
	s.RegisterRoutes()
	return s
}

func performRequest(router http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	s := newTestServer(t, &fakeGeoIP{})
	resp := performRequest(s.router, http.MethodGet, "/healthz")

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if body["data"] != "healthz" {
		t.Fatalf("expected healthz response, got %s", body["data"])
	}
}

func TestReadiness(t *testing.T) {
	svc := &fakeGeoIP{ready: false}
	s := newTestServer(t, svc)

	resp := performRequest(s.router, http.MethodGet, "/readiness")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when not ready, got %d", resp.Code)
	}

	svc.ready = true
	resp = performRequest(s.router, http.MethodGet, "/readiness")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 when ready, got %d", resp.Code)
	}
}

func TestMaxMindHandlerValidation(t *testing.T) {
	s := newTestServer(t, &fakeGeoIP{})

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"missing address", "/ip", http.StatusBadRequest},
		{"invalid address", "/ip?address=not-an-ip", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performRequest(s.router, http.MethodGet, tt.path)
			if resp.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestMaxMindHandlerSuccess(t *testing.T) {
	record := models.Record{IP: "1.1.1.1"}
	svc := &fakeGeoIP{
		record: record,
		ready:  true,
	}
	s := newTestServer(t, svc)

	resp := performRequest(s.router, http.MethodGet, "/ip?address=1.1.1.1")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload struct {
		Data models.Record `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if payload.Data.IP != "1.1.1.1" {
		t.Fatalf("expected IP 1.1.1.1, got %s", payload.Data.IP)
	}

	if svc.lastLookupIP.String() != "1.1.1.1" {
		t.Fatalf("expected lookup to be called with 1.1.1.1, got %s", svc.lastLookupIP)
	}
}

func TestMaxMindHandlerErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"database missing", services.ErrMaxMindDatabaseMissing, http.StatusServiceUnavailable},
		{"unexpected error", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeGeoIP{
				lookupErr: tt.err,
				ready:     true,
			}
			s := newTestServer(t, svc)
			resp := performRequest(s.router, http.MethodGet, "/ip?address=8.8.8.8")

			if resp.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestDownloaderHandlerNilService(t *testing.T) {
	s := newTestServer(t, nil)
	resp := performRequest(s.router, http.MethodGet, "/updatedb")

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestDownloaderHandlerErrors(t *testing.T) {
	svc := &fakeGeoIP{
		updateErr: services.ErrMaxMindLicenseMissing,
	}
	s := newTestServer(t, svc)

	resp := performRequest(s.router, http.MethodGet, "/updatedb")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing license, got %d", resp.Code)
	}

	svc.updateErr = errors.New("download failed")
	resp = performRequest(s.router, http.MethodGet, "/updatedb?force=true")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for generic error, got %d", resp.Code)
	}

	if len(svc.updateCalls) == 0 || !svc.updateCalls[len(svc.updateCalls)-1] {
		t.Fatalf("expected force flag to be passed on update call")
	}
}

func TestDownloaderHandlerSuccess(t *testing.T) {
	svc := &fakeGeoIP{
		updateStatus: services.UpdateStatus{
			Updated: true,
			Reason:  "remote checksum changed",
		},
	}
	s := newTestServer(t, svc)

	resp := performRequest(s.router, http.MethodGet, "/updatedb")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if body["update"] != true {
		t.Fatalf("expected update flag true, got %v", body["update"])
	}

	if body["message"] != "remote checksum changed" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}
