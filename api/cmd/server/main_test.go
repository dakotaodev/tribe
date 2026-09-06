package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", contentType)
	}
	if body := recorder.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("body = %q, want status response", body)
	}
}

func TestLoadConfigUsesPortEnvironmentVariable(t *testing.T) {
	t.Setenv("PORT", "9090")

	config := loadConfig()

	if config.port != "9090" {
		t.Fatalf("port = %q, want %q", config.port, "9090")
	}
}

func TestLoadConfigDefaultsPort(t *testing.T) {
	t.Setenv("PORT", "")

	config := loadConfig()

	if config.port != defaultPort {
		t.Fatalf("port = %q, want %q", config.port, defaultPort)
	}
}

func TestServeShutsDownWhenContextCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := httptest.NewServer(newRouter(slog.New(slog.NewTextHandler(io.Discard, nil))))
	server.Close()

	apiServer := &http.Server{
		Addr:    server.Listener.Addr().String(),
		Handler: newRouter(slog.New(slog.NewTextHandler(io.Discard, nil))),
	}
	context, cancel := context.WithCancel(context.Background())
	errors := make(chan error, 1)
	go func() {
		errors <- serve(context, apiServer, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()

	deadline := time.Now().Add(time.Second)
	for {
		response, err := http.Get("http://" + apiServer.Addr + "/health")
		if err == nil {
			response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-errors:
		if err != nil {
			t.Fatalf("serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not shut down")
	}
}
