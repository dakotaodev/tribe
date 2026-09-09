package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequireRejectsMissingAndInvalidBearerTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := newSigningKey(t)
	server := newJWKSServer(t, key)
	defer server.Close()
	verifier := newTestVerifier(t, server.URL, time.Now)
	router := protectedRouter(verifier)

	for _, authorization := range []string{"", "Basic credential", "Bearer", "Bearer invalid.token.value"} {
		t.Run(authorization, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if authorization != "" {
				request.Header.Set("Authorization", authorization)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if recorder.Body.String() != `{"error":{"code":"unauthorized","message":"authentication required"}}` {
				t.Fatalf("body = %s", recorder.Body.String())
			}
		})
	}
}

func TestRequireMakesOnlyVerifiedIdentityAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := newSigningKey(t)
	server := newJWKSServer(t, key)
	defer server.Close()
	verifier := newTestVerifier(t, server.URL, time.Now)
	router := protectedRouter(verifier)
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+signedToken(t, key, testIssuer, testAudience, time.Now().Add(time.Hour), testUserID))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != `{"user_id":"`+testUserID+`"}` {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func protectedRouter(verifier *Verifier) *gin.Engine {
	router := gin.New()
	router.GET("/protected", Require(verifier), func(context *gin.Context) {
		identity, ok := IdentityFromContext(context)
		if !ok {
			context.Status(http.StatusInternalServerError)
			return
		}
		context.JSON(http.StatusOK, gin.H{"user_id": identity.UserID})
	})
	return router
}
