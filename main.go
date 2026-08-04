package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
	"xivstrings/pkg/server"
	"xivstrings/pkg/store"
	"xivstrings/pkg/version"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address (e.g. :8080)")
	dataDir := flag.String("data", "data", "directory containing JSON data files and index files")
	uiDir := flag.String("ui", "ui/dist", "directory containing UI static files")
	flag.Parse()

	// A GitHub outage must not keep the server down when the data is already here.
	// Only startup degrades like this: POST /api/version asks for an update on
	// purpose, so runUpdate still reports the failure instead of quietly succeeding.
	result, err := version.EnsureVersion(*dataDir)
	if err != nil {
		log.Printf("could not check the latest release: %v", err)
		result, err = version.ResolveLocalVersion(*dataDir)
		if err != nil {
			log.Fatalf("no usable local data, cannot start: %v", err)
		}
		log.Printf("continuing with local data, version %s", result.Version)
	}
	log.Printf("using version %s (data: %s, index: %s)", result.Version, result.StringDir, result.IndexDir)

	st, err := store.LoadStore(result.StringDir, result.IndexDir)
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}

	mux := server.CreateMux(server.ServerConfig{
		Store:       st,
		UiDir:       *uiDir,
		BaseDir:     *dataDir,
		UpdateToken: os.Getenv("XIVSTRINGS_UPDATE_TOKEN"),
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("xivstrings server listening on %s (version: %s, ui dir: %s)", *addr, result.Version, *uiDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
