package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/thiagozs/geolocation-go/models"
	"github.com/thiagozs/geolocation-go/services"
)

type GeoIPService interface {
	Lookup(net.IP) (models.Record, error)
	Update(context.Context, bool) (services.UpdateStatus, error)
	Ready() bool
	DatabasePath() string
	Close() error
}

type Config struct {
	HTTPPort int
	Mode     string
	GeoIP    services.MaxMindConfig
}

type Server struct {
	cfg    Config
	http   *http.Server
	router *gin.Engine
	geoIP  GeoIPService
	log    *logrus.Entry
}

func NewServer(cfg Config) (*Server, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	serverLogger := logrus.NewEntry(logger).WithField("component", "server")
	geoLogger := logrus.NewEntry(logger).WithField("component", "geoip")

	geoSvc, err := services.NewMaxMindService(geoLogger, cfg.GeoIP)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:   cfg,
		geoIP: geoSvc,
		log:   serverLogger,
	}, nil
}

func (s *Server) RegisterRoutes() {
	if !strings.EqualFold(s.cfg.Mode, "development") && !strings.EqualFold(s.cfg.Mode, "dev") {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery(), cors.Default())

	router.GET("/ip", s.MaxMindHandler)
	router.GET("/healthz", s.Healthz)
	router.GET("/readiness", s.Readiness)
	router.GET("/updatedb", s.DownloaderMaxMind)

	s.router = router
}

func (s *Server) RegisterHTTP() {
	if s.router == nil {
		s.RegisterRoutes()
	}

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.HTTPPort),
		Handler:      s.router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

func (s *Server) Run() error {
	if s.router == nil {
		s.RegisterRoutes()
	}
	if s.http == nil {
		s.RegisterHTTP()
	}

	s.log.WithField("port", s.cfg.HTTPPort).Info("starting server")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.WithError(err).Error("http server failure")
		}
	}()

	<-quit

	s.gracefulShutdown()

	return nil
}

func (s *Server) gracefulShutdown() {
	s.log.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.geoIP.Close(); err != nil {
		s.log.WithError(err).Warn("could not close maxmind database")
	}

	if s.http != nil {
		s.http.SetKeepAlivesEnabled(false)
		if err := s.http.Shutdown(ctx); err != nil {
			s.log.WithError(err).Error("could not gracefully shutdown http server")
		}
	}

	<-ctx.Done()
}
