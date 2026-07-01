package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-memory-mcp/internal/config"
	"agent-memory-mcp/internal/httpapi"
	"agent-memory-mcp/internal/ingest"
	"agent-memory-mcp/internal/ingest/watcher"
	"agent-memory-mcp/internal/jobs"
	"agent-memory-mcp/internal/mcp"
	"agent-memory-mcp/internal/memory"
	"agent-memory-mcp/internal/patterns"
	"agent-memory-mcp/internal/store/sqlite"
	"agent-memory-mcp/internal/usage"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: agent-memory [serve|mcp|refresh-relevance]")
		os.Exit(1)
	}

	cmd := os.Args[1]
	cfg := config.Load()

	log.Printf("Starting agent-memory (cmd=%s, root=%s)", cmd, cfg.Root)

	store, err := sqlite.Open(cfg.Root)
	if err != nil {
		log.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := sqlite.RunLegacyMigration(ctx, store, cfg.Root); err != nil {
		log.Printf("Legacy migration error: %v", err)
	}
	cancel()

	memoryService := memory.NewService(store)
	usageService := usage.NewService(store)

	switch cmd {
	case "serve":
		server := httpapi.NewServer(memoryService, usageService)
		srv := &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: server.Handler(),
		}
		
		go func() {
			log.Printf("Listening on http://%s", cfg.HTTPAddr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server error: %v", err)
			}
		}()
		
		w, err := watcher.New(500 * time.Millisecond)
		if err != nil {
			log.Printf("failed to create watcher: %v", err)
		} else {
			ingestService := ingest.NewService(w, memoryService)
			ingestService.Start(context.Background())
			w.AddRecursive(cfg.Root)
			log.Printf("Watcher started on %s", cfg.Root)
		}

		// Start background intelligence jobs
		detector := patterns.NewDetector(store)
		jobRunner := jobs.NewRunner(store, detector)
		jobRunner.Start(context.Background())

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down server...")

		jobRunner.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
		
	case "mcp":
		mcpServer := mcp.NewServer(memoryService, usageService)
		if err := mcpServer.ServeStdio(); err != nil {
			log.Fatalf("MCP Server error: %v", err)
		}

	case "refresh-relevance":
		ctx := context.Background()
		c1, c2, err := memoryService.RefreshRelevance(ctx, nil)
		if err != nil {
			log.Fatalf("Refresh error: %v", err)
		}
		log.Printf("Refreshed %d memories, classified %d as noise", c1, c2)

	default:
		log.Fatalf("Unknown command: %s", cmd)
	}
}
