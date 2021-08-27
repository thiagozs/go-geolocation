package server

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thiagozs/geolocation-go/pkg/utils"
)

func (s *Server) MaxMindHandler(c *gin.Context) {
	ip := c.Param("ip")

	if !utils.IsValidIPAddress(ip) {
		c.JSON(http.StatusOK, gin.H{"message": "need ip address"})
		return
	}

	record, err := s.services.Lookup(net.ParseIP(ip))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": record})
}

func (s *Server) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": "healthz"})
}

func (s *Server) Readiness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": "readness"})
}
