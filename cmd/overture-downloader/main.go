package main

import (
	"flag"
	"fmt"
	"os"

	"overture-downloader/internal/downloader"
)

func main() {
	natsHost := flag.String("host", "127.0.0.1", "NATS server host")
	natsPort := flag.Int("port", 4222, "NATS server port")
	maxWorkers := flag.Uint("workers", 1, "Number of concurrent processes that can run at any given time")
	jwksUrl := flag.String("jwks", "https://turfbuilder.org/.well-known/jwks.json", "The full URL for your applications JSON Web Key Set. Typically located at `<HOST>/.well-known/jwks.json`.")
	apiPort := flag.Int("api-port", 23363, "Port for the REST API")
	overtureRelease := flag.String("overture-release", "2026-06-17.0", "Overture Maps data release to query")
	debug := flag.Bool("debug", false, "Disable JWT validation. Never use in production.")

	flag.Parse()

	if *maxWorkers < 1 {
		fmt.Fprintln(os.Stderr, "error: --workers must be at least 1")
		os.Exit(1)
	}

	downloader.Run(downloader.Config{
		NatsHost:        *natsHost,
		NatsPort:        *natsPort,
		MaxWorkers:      *maxWorkers,
		JwksUrl:         *jwksUrl,
		ApiPort:         *apiPort,
		OvertureRelease: *overtureRelease,
		Debug:           *debug,
	})
}
