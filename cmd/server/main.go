// Command server runs the library HTTP API on port 8080 with sample data.
package main

import (
	"log"
	"net/http"
	"time"

	"buecherei/internal/api"
	"buecherei/internal/library"
)

func main() {
	lib := library.New()
	if err := seed(lib); err != nil {
		log.Fatalf("seed: %v", err)
	}

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           api.NewHandler(lib),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
