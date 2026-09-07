package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const identityContextKey = "authenticatedIdentity"

// Require verifies a bearer token and makes its identity available to the
// downstream Gin handler. Domain code should receive Identity as a plain value.
func Require(verifier *Verifier) gin.HandlerFunc {
	return func(context *gin.Context) {
		token, ok := bearerToken(context.GetHeader("Authorization"))
		if !ok {
			unauthorized(context)
			return
		}
		identity, err := verifier.Verify(context.Request.Context(), token)
		if err != nil {
			unauthorized(context)
			return
		}
		context.Set(identityContextKey, identity)
		context.Next()
	}
}

// IdentityFromContext returns a verified caller identity set by Require.
func IdentityFromContext(context *gin.Context) (Identity, bool) {
	identity, ok := context.Get(identityContextKey)
	if !ok {
		return Identity{}, false
	}
	caller, ok := identity.(Identity)
	return caller, ok
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func unauthorized(context *gin.Context) {
	context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": "unauthorized", "message": "authentication required"},
	})
}
