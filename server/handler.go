package server

import (
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thiagozs/geolocation-go/models"
	"github.com/thiagozs/geolocation-go/pkg/utils"
)

func (s *Server) MaxMindHandler(c *gin.Context) {
	req := models.Request{}
	if c.Bind(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing parametes"})
		return
	}

	if !utils.IsValidIPAddress(req.Address) {
		c.JSON(http.StatusOK, gin.H{"message": "need ip address"})
		return
	}

	record, err := s.services.Lookup(net.ParseIP(req.Address))
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

func (s *Server) DownloaderMaxMind(c *gin.Context) {

	maxmind := os.Getenv("MAXMIND_KEY")
	filePath, _ := os.Getwd()
	filePath = filePath + "/db/GeoLite2-City.mmdb"

	if maxmind == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing maxmind key"})
		return
	}

	downloader := utils.NewDatabaseDownloader(maxmind, filePath, time.Duration(30*time.Second))

	ok, err := downloader.ShouldDownload()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if ok {
		if utils.FileExists(filePath) {
			utils.DeleteFile(filePath)
		}

		if err := downloader.Download(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"update": ok, "file": filePath, "message": "file downloaded"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"update": ok, "file": filePath, "message": "file already updated"})
}
