package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("API server starting", "port", port)
	if err := newRouter().Run(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("API server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/healthz", healthHandler)

	return router
}

func healthHandler(context *gin.Context) {
	context.JSON(200, gin.H{"status": "ok"})
}
