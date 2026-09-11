package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dakota/tribe/api/internal/users"
	"github.com/gin-gonic/gin"
)

type verifierStub struct {
	subject string
	err     error
}

func (v verifierStub) Verify(context.Context, string) (string, error) { return v.subject, v.err }

func TestMiddlewareRejectsMissingAndInvalidTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name, header string
		verifier     Verifier
	}{
		{name: "missing", verifier: verifierStub{subject: "user"}},
		{name: "malformed", header: "Basic abc", verifier: verifierStub{subject: "user"}},
		{name: "invalid", header: "Bearer abc", verifier: verifierStub{err: errors.New("invalid")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(Middleware(test.verifier))
			router.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodGet, "/me", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d", response.Code)
			}
		})
	}
}

func TestMiddlewareUsesVerifiedSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Middleware(verifierStub{subject: "verified-user"}))
	router.GET("/me", func(c *gin.Context) { c.String(http.StatusOK, c.GetString(users.AuthenticatedUserIDKey)) })
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "verified-user" {
		t.Fatalf("status/body = %d/%q", response.Code, response.Body.String())
	}
}
