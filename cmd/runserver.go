package cmd

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thiagozs/geolocation-go/server"
	"github.com/thiagozs/geolocation-go/services"
)

var runserverCmd = &cobra.Command{
	Use:   "runserver",
	Short: "Run micro service for geolocation",
	Run:   runserver,
}

var (
	cfgFile  string
	httpPort int
)

func init() {
	runserverCmd.PersistentFlags().StringVar(&cfgFile, "config", ".env", "config file (default is $HOME/.env)")
	runserverCmd.PersistentFlags().IntVar(&httpPort, "http", 5000, "port for http server")
	rootCmd.AddCommand(runserverCmd)
}

func runserver(cmd *cobra.Command, args []string) {
	serverCfg := server.Config{
		HTTPPort: httpPort,
		Mode:     resolveMode(),
		GeoIP:    buildMaxMindConfig(),
	}

	srv, err := server.NewServer(serverCfg)
	if err != nil {
		log.Fatal(err)
	}

	srv.RegisterRoutes()
	srv.RegisterHTTP()

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

func resolveMode() string {
	mode := strings.TrimSpace(viper.GetString("MODE"))
	if mode == "" {
		return "development"
	}
	return mode
}

func buildMaxMindConfig() services.MaxMindConfig {
	cfg := services.MaxMindConfig{
		DatabasePath: strings.TrimSpace(viper.GetString("MAXMIND_DB_PATH")),
		LicenseKey:   strings.TrimSpace(viper.GetString("MAXMIND_KEY")),
	}

	if timeout := readDuration("MAXMIND_HTTP_TIMEOUT"); timeout > 0 {
		cfg.HTTPTimeout = timeout
	}

	if refresh := readDuration("MAXMIND_REFRESH_INTERVAL"); refresh > 0 {
		cfg.MinRefreshInterval = refresh
	}

	return cfg
}

func readDuration(key string) time.Duration {
	value := strings.TrimSpace(viper.GetString(key))
	if value == "" {
		return 0
	}

	if dur, err := time.ParseDuration(value); err == nil {
		return dur
	}

	if secs, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Duration(secs) * time.Second
	}

	log.Printf("invalid duration for %s: %s", key, value)
	return 0
}
