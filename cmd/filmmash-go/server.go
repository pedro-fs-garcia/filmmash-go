package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func InitServer(port string, router *chi.Mux) (*http.Server, error) {
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", httpServer.Addr, err)
	}

	log.Printf("server listening on port %s\n", httpServer.Addr)

	if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return nil, err
	}
	return httpServer, nil
}
