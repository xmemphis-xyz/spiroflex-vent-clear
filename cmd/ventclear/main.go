package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mtojek/spiroflex-vent-clear"
	"github.com/mtojek/spiroflex-vent-clear/api"
)

func setupLogging() (func(), error) {
	path := os.Getenv("VENTCLEAR_LOG_FILE")
	if path == "" {
		path = "logs/ventclear.log"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, err
	}
	log.SetOutput(io.MultiWriter(os.Stderr, file))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	return func() { _ = file.Close() }, nil
}

func main() {
	closeLog, err := setupLogging()
	if err != nil {
		log.Fatalf("unable to initialize log file: %v", err)
	}
	defer closeLog()

	c, err := spiroflex.LoadConfig()
	if err != nil {
		log.Fatalf("can't load config: %v", err)
	}

	webServer := api.NewWebServer(c)
	srv := &http.Server{
		Addr:    c.API.Endpoint,
		Handler: webServer.Handler(),
	}

	log.Printf("Server started at %v", c.API.Endpoint)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("srv.ListenAndServe failed: %v", err)
	}
}
