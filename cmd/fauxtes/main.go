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
	if v := os.Getenv("FAUXTES_TLS_CERT"); v != "" {
		cfg.TLSCert = v
	}
	if v := os.Getenv("FAUXTES_TLS_KEY"); v != "" {
		cfg.TLSKey = v
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	defer srv.DB.Close()

	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		fmt.Printf("fauxtes-server listening on %s (TLS)\n", cfg.Addr)
		if err := http.ListenAndServeTLS(cfg.Addr, cfg.TLSCert, cfg.TLSKey, srv.Mux); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("fauxtes-server listening on %s\n", cfg.Addr)
		if err := http.ListenAndServe(cfg.Addr, srv.Mux); err != nil {
			log.Fatal(err)
		}
	}
}
