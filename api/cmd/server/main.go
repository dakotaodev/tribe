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

	"github.com/dakota/tribe/api/internal/auth"
	"github.com/dakota/tribe/api/internal/users"
	"github.com/gin-gonic/gin"
)

const defaultPort = "8080"

type config struct {
	port            string
	shutdownTimeout time.Duration
	auth            auth.Config
	supabaseURL     string
	supabaseSecret  string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	config, err := loadConfig()
	if err != nil {
		logger.Error("invalid API configuration", "error", err)
		os.Exit(1)
	}
	verifier, err := auth.NewVerifier(config.auth, nil)
	if err != nil {
		logger.Error("could not configure Supabase JWT verifier", "error", err)
		os.Exit(1)
	}
	profileRepository := users.NewSupabaseRepository(config.supabaseURL, config.supabaseSecret, http.DefaultClient)
	avatarStore := users.NewSupabaseAvatarStore(config.supabaseURL, config.supabaseSecret, http.DefaultClient)
	profileHandler := users.NewHandler(users.NewService(profileRepository, avatarStore))
	server := &http.Server{
		Addr:              ":" + config.port,
		Handler:           newRouterWithProfile(logger, profileHandler, verifier),
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

func loadConfig() (config, error) {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}
	issuer := strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_JWT_ISSUER")), "/")
	if issuer == "" {
		return config{}, errors.New("SUPABASE_JWT_ISSUER is required")
	}
	audience := strings.TrimSpace(os.Getenv("SUPABASE_JWT_AUDIENCE"))
	if audience == "" {
		audience = "authenticated"
	}

	return config{
		port:            port,
		shutdownTimeout: 10 * time.Second,
		supabaseURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_URL")), "/"),
		supabaseSecret:  strings.TrimSpace(os.Getenv("SUPABASE_SECRET_KEY")),
		auth: auth.Config{
			Issuer: issuer, Audience: audience, JWKSURL: issuer + "/.well-known/jwks.json",
		},
	}, nil
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

func newRouter(logger *slog.Logger, verifier *auth.Verifier) *gin.Engine {
	return newRouterWithProfile(logger, nil, verifier)
}

func newRouterWithProfile(logger *slog.Logger, profileHandler *users.Handler, verifier *auth.Verifier) *gin.Engine {
	router := gin.New()
	router.Use(requestLogger(logger), gin.CustomRecovery(func(context *gin.Context, recovered any) {
		logger.Error("request panicked", "error", recovered, "method", context.Request.Method, "path", context.Request.URL.Path)
		context.AbortWithStatus(http.StatusInternalServerError)
	}))
	router.GET("/health", healthHandler)
	if profileHandler != nil && verifier != nil {
		currentUser := router.Group("/me", auth.Require(verifier), profileUserContext())
		currentUser.GET("", profileHandler.GetMe)
		currentUser.PUT("", profileHandler.PutMe)
		currentUser.PUT("/avatar", profileHandler.PutAvatar)
	}

	return router
}

func profileUserContext() gin.HandlerFunc {
	return func(context *gin.Context) {
		identity, ok := auth.IdentityFromContext(context)
		if !ok {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHENTICATED", "message": "Authentication required."},
			})
			return
		}
		context.Set(users.AuthenticatedUserIDKey, identity.UserID)
		context.Next()
	}
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
