package api

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"trawler/pkg/logging"
)

// StartHealthServer starts the health check HTTP server
// It runs in a goroutine and handles graceful shutdown
func StartHealthServer(port int, stopChan <-chan struct{}) error {
	mux := http.NewServeMux()

	// Register health check endpoints
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/live", LivenessHandler)
	mux.HandleFunc("/ready", ReadinessHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to capture server errors
	serverErr := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("Health API server listening on port %d", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("health server error: %w", err)
		}
	}()

	// Wait for stop signal
	select {
	case err := <-serverErr:
		return err
	case <-stopChan:
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Shutting down health API server...")

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("health server shutdown error: %w", err)
		}

		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Health API server stopped")
		return nil
	}
}
