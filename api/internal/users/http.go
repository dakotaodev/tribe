package users

import (
	"errors"
	"io"
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
	c.JSON(http.StatusOK, profileResponse(profile))
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
		c.JSON(http.StatusOK, profileResponse(profile))
	}
}

func (h *Handler) PutAvatar(c *gin.Context) {
	// Leave room for multipart framing while preventing arbitrarily large bodies
	// from being buffered to disk before the image limit is checked.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6*1024*1024)
	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse("invalid_avatar", ErrInvalidAvatar.Error(), nil))
		return
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, 5*1024*1024+1))
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse("avatar_upload_failed", "avatar upload failed", nil))
		return
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(contents)
	}
	profile, err := h.service.UpdateAvatar(c.Request.Context(), c.GetString(AuthenticatedUserIDKey), contentType, contents)
	switch {
	case errors.Is(err, ErrInvalidAvatar):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("invalid_avatar", err.Error(), nil))
	case err != nil:
		c.JSON(http.StatusBadGateway, errorResponse("avatar_upload_failed", "avatar upload failed", nil))
	default:
		c.JSON(http.StatusOK, profileResponse(profile))
	}
}

func profileResponse(profile Profile) gin.H {
	return gin.H{"profile": gin.H{
		"id": profile.ID, "username": profile.Username, "display_name": profile.DisplayName,
		"bio": profile.Bio, "avatar_url": profile.AvatarURL, "created_at": profile.CreatedAt, "updated_at": profile.UpdatedAt,
	}, "profile_complete": true}
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
