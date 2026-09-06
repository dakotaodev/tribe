package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultPort = "8080"

type config struct {
	port            string
	shutdownTimeout time.Duration
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	config := loadConfig()
	server := &http.Server{
		Addr:              ":" + config.port,
		Handler:           newRouter(logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("API server starting", "port", config.port)
	if err := serve(shutdownSignal, server, config.shutdownTimeout, logger); err != nil {
		logger.Error("API server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func loadConfig() config {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	return config{
		port:            port,
		shutdownTimeout: 10 * time.Second,
	}
}

func serve(ctx context.Context, server *http.Server, shutdownTimeout time.Duration, logger *slog.Logger) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP server: %w", err)
	case <-ctx.Done():
		logger.Info("API server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		return nil
	}
}

func newRouter(logger *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(requestLogger(logger), gin.CustomRecovery(func(context *gin.Context, recovered any) {
		logger.Error("request panicked", "error", recovered, "method", context.Request.Method, "path", context.Request.URL.Path)
		context.AbortWithStatus(http.StatusInternalServerError)
	}))
	router.GET("/health", healthHandler)

	return router
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		startedAt := time.Now()
		context.Next()

		logger.Info("HTTP request completed",
			"method", context.Request.Method,
			"path", context.Request.URL.Path,
			"status", context.Writer.Status(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	}
}

func healthHandler(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}
