package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/thiagozs/geolocation-go/server"
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

	srv, err := server.NewServer(httpPort)
	if err != nil {
		log.Fatal(err)
	}
	srv.RegisterRoutes()
	srv.RegisterHTTP()

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
