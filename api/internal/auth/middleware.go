package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/dakota/tribe/api/internal/users"
	"github.com/gin-gonic/gin"
)

type Verifier interface {
	Verify(context.Context, string) (string, error)
}

func Middleware(verifier Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(c)
			return
		}
		userID, err := verifier.Verify(c.Request.Context(), parts[1])
		if err != nil || userID == "" {
			unauthorized(c)
			return
		}
		c.Set(users.AuthenticatedUserIDKey, userID)
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{
		"code": "unauthorized", "message": "a valid bearer token is required",
	}})
}
