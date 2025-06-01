// Bootstrapping the Service

package main

import (
	"Microservices-based-E-commerce-System/account"
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/tinrab/retry"
)

type Config struct {
	DatabaseURL string `envconfig:"DATABASE_URL"`
}

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	var r account.Repository
	retry.ForeverSleep(2*time.Second, func(_ int) (err error) { // Keeps trying DB connection every 2 seconds until it works.
		r, err = account.NewPostgresRepository(cfg.DatabaseURL)
		if err != nil {
			log.Print(err)
		}
		return
	})
	defer r.Close()
	log.Println("Listening on port 8080...")
	s := account.NewService(r)
	log.Fatal(account.ListenGRPC(s, 8080)) // Starts the gRPC server.
}
