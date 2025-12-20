package main

import (
	"flag"
	"log"
	"net/http"
	"time"
	"xivstrings/pkg/server"
	"xivstrings/pkg/store"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address (e.g. :8080)")
	dataDir := flag.String("data", "data", "directory containing JSON data files")
	uiDir := flag.String("ui", "ui/dist", "directory containing UI static files")
	flag.Parse()

	store, err := store.LoadStore(*dataDir)
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}

	mux := server.CreateMux(server.ServerConfig{
		Store: store,
		UiDir: *uiDir,
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("xivstrings server listening on %s (data dir: %s, ui dir: %s)", *addr, *dataDir, *uiDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
