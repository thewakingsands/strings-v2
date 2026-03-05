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

	result, err := version.EnsureVersion(*dataDir)
	if err != nil {
		log.Fatalf("failed to ensure version: %v", err)
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
