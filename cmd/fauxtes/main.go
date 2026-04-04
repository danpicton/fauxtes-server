package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/danpicton/fauxtes-server/internal/server"
)

func main() {
	cfg := server.DefaultConfig()

	if v := os.Getenv("FAUXTES_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("FAUXTES_DB"); v != "" {
		cfg.DBPath = v
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	defer srv.DB.Close()

	fmt.Printf("fauxtes-server listening on %s\n", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, srv.Mux); err != nil {
		log.Fatal(err)
	}
}
