package httpx

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// GracefulShutdown handles graceful shutdown of HTTP server
func GracefulShutdown(server *http.Server, logger *zap.Logger, timeout time.Duration) {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt graceful shutdown
	logger.Info("Starting graceful shutdown", zap.Duration("timeout", timeout))
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed", zap.Error(err))
		return
	}

	logger.Info("Server shutdown completed")
}

// StartServerWithGracefulShutdown starts server and handles graceful shutdown
func StartServerWithGracefulShutdown(server *http.Server, logger *zap.Logger, shutdownTimeout time.Duration) {
	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Handle graceful shutdown
	GracefulShutdown(server, logger, shutdownTimeout)
}
