package users

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const AuthenticatedUserIDKey = "authenticated_user_id"

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetMe(c *gin.Context) {
	profile, err := h.service.Get(c.Request.Context(), c.GetString(AuthenticatedUserIDKey))
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusOK, gin.H{"profile": nil, "profile_complete": false})
		return
	}
	if err != nil {
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": profile, "profile_complete": true})
}

func (h *Handler) PutMe(c *gin.Context) {
	var request struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_error", "request body must be valid JSON", nil))
		return
	}
	profile, err := h.service.Update(c.Request.Context(), c.GetString(AuthenticatedUserIDKey), Update{
		Username: request.Username, DisplayName: request.DisplayName, Bio: request.Bio,
	})
	var validation ValidationErrors
	switch {
	case errors.As(err, &validation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_error", "profile validation failed", validation))
	case errors.Is(err, ErrUsernameTaken):
		c.JSON(http.StatusConflict, errorResponse("username_taken", "username is already taken", nil))
	case err != nil:
		internalError(c)
	default:
		c.JSON(http.StatusOK, gin.H{"profile": profile, "profile_complete": true})
	}
}

func errorResponse(code, message string, fields ValidationErrors) gin.H {
	detail := gin.H{"code": code, "message": message}
	if fields != nil {
		detail["fields"] = fields
	}
	return gin.H{"error": detail}
}

func internalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, errorResponse("internal_error", "an internal error occurred", nil))
}
