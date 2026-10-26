// Command server runs the library HTTP API (default port 8080). The state is stored
// in a JSON file; if the file does not exist yet, it starts with sample data.
package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"buecherei/internal/api"
	"buecherei/internal/library"
	"buecherei/internal/storage"
	"buecherei/internal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataPath := flag.String("data", "buecherei.json", "path of the JSON data file")
	flag.Parse()

	lib, err := load(storage.NewFileStore(*dataPath))
	if err != nil {
		log.Fatalf("load %s: %v", *dataPath, err)
	}

	apiHandler := api.NewHandler(lib)
	mux := http.NewServeMux()
	mux.Handle("/books", apiHandler)
	mux.Handle("/books/", apiHandler)
	mux.Handle("GET /{$}", web.Handler())

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s, data in %s", srv.Addr, *dataPath)
	log.Fatal(srv.ListenAndServe())
}

// load restores the library from the store, or seeds and saves sample data if
// the store is empty. Afterwards every change is persisted.
func load(store *storage.FileStore) (*library.Library, error) {
	var lib *library.Library
	snap, err := store.Load()
	switch {
	case errors.Is(err, os.ErrNotExist):
		lib = library.New()
		if err := seed(lib); err != nil {
			return nil, err
		}
		if err := store.Save(lib.Snapshot()); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if lib, err = library.NewFromSnapshot(snap); err != nil {
			return nil, err
		}
	}
	lib.PersistWith(store.Save)
	return lib, nil
}
