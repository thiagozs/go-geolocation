package main

import (
	"github.com/thiagozs/geolocation-go/cmd"
)

func main() {

	// srv, err := server.NewServer(8181)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// srv.RegisterRoutes()
	// srv.RegisterHTTP()

	// if err := srv.Run(); err != nil {
	// 	log.Fatal(err)
	// }

	cmd.Execute()

}
