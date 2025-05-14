package main

import (
	"flag"
	"log"

	"github.com/stacklok/toolhive/pkg/gui"
	"github.com/stacklok/toolhive/pkg/logger"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	logger.Initialize()

	server := gui.NewServer()
	log.Printf("Starting ToolHive web interface on http://localhost:8080")
	if err := server.Start(*addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
