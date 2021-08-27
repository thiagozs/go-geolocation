package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/thiagozs/geolocation-go/services"
)

type Server struct {
	httpPort int
	http     *http.Server
	router   *gin.Engine
	services *services.MaxMindDB
	env      string
	log      *logrus.Entry
}

func NewServer(httpPort int) (*Server, error) {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logger := logrus.NewEntry(logrus.New())

	maxmind, err := services.NewMindMax(logger)
	if err != nil {
		return &Server{}, err
	}
	return &Server{
		httpPort: httpPort,
		services: maxmind,
		env:      strings.ToUpper(os.Getenv("MODE")),
		log:      logger,
	}, nil
}

func (s *Server) RegisterHTTP() {
	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%s", strconv.Itoa(s.httpPort)),
		Handler:      s.router,
		ReadTimeout:  time.Duration(5) * time.Second,
		WriteTimeout: time.Duration(5) * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

func (s *Server) RegisterRoutes() {
	if !(s.env == strings.ToUpper("development") ||
		s.env == strings.ToUpper("dev")) {
		gin.SetMode(gin.ReleaseMode)
	}
	s.router = gin.Default()
	s.router.Use(cors.Default())

	// endpoint ip
	s.router.GET("/ip", s.MaxMindHandler)

	// health check
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/readiness", s.Readiness)
}

func (s *Server) Run() error {
	s.log.Println("Start server...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.http.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			s.log.Fatal(err)
		}
	}()

	<-quit

	s.gracefulShutdown()

	return nil
}

func (s *Server) gracefulShutdown() {
	s.log.Println("Shutdown server...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.services.Close(); err != nil {
		s.log.Printf("Could not close database MaxMind: %v\n", err)
	}

	s.http.SetKeepAlivesEnabled(false)
	if err := s.http.Shutdown(ctx); err != nil {
		s.log.Fatalf("Could not gracefully shutdown: %v\n", err)
	}

	<-ctx.Done()
}
