package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thiagozs/geolocation-go/models"
	"github.com/thiagozs/geolocation-go/pkg/utils"
	"github.com/thiagozs/geolocation-go/services"
)

func (s *Server) MaxMindHandler(c *gin.Context) {
	var req models.Request
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing address parameter"})
		return
	}

	addr := strings.TrimSpace(req.Address)
	if addr == "" || !utils.IsValidIPAddress(addr) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid ip address"})
		return
	}

	ip := net.ParseIP(addr)
	record, err := s.geoIP.Lookup(ip)
	if err != nil {
		if errors.Is(err, services.ErrMaxMindDatabaseMissing) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database not loaded"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": record})
}

func (s *Server) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": "healthz"})
}

func (s *Server) Readiness(c *gin.Context) {
	if !s.geoIP.Ready() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "message": "maxmind database not available"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

func (s *Server) DownloaderMaxMind(c *gin.Context) {
	if s.geoIP == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "geoip service not configured"})
		return
	}

	force := strings.EqualFold(c.Query("force"), "true")
	timeout := s.cfg.GeoIP.HTTPTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx := c.Request.Context()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	status, err := s.geoIP.Update(ctx, force)
	if err != nil {
		if errors.Is(err, services.ErrMaxMindLicenseMissing) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing maxmind license key"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	message := status.Reason
	if message == "" {
		if status.Updated {
			message = "database downloaded"
		} else {
			message = "database already up to date"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"update":  status.Updated,
		"file":    s.geoIP.DatabasePath(),
		"message": message,
	})
}
