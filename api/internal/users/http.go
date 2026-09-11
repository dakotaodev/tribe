package users

import (
	"encoding/json"
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
		c.JSON(http.StatusNotFound, errorResponse("PROFILE_ONBOARDING_REQUIRED", "Create your profile to continue.", nil))
		return
	}
	if err != nil {
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, profileResponse(profile))
}

func (h *Handler) PutMe(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	request, validation, err := decodeProfileRequest(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "Request body must contain only supported profile fields.", nil))
		return
	}
	if validation != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse("VALIDATION_FAILED", "Profile validation failed.", validation))
		return
	}
	profile, created, err := h.service.Update(c.Request.Context(), c.GetString(AuthenticatedUserIDKey), Update{
		Username: request.Username, DisplayName: request.DisplayName, Bio: request.Bio,
	})
	validation = nil
	switch {
	case errors.As(err, &validation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("VALIDATION_FAILED", "Profile validation failed.", validation))
	case errors.Is(err, ErrUsernameTaken):
		c.JSON(http.StatusConflict, errorResponse("USERNAME_TAKEN", "That username is unavailable.", ValidationErrors{"username": "Choose a different username."}))
	case err != nil:
		internalError(c)
	default:
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		c.JSON(status, profileResponse(profile))
	}
}

type profileRequest struct {
	Username    string
	DisplayName string
	Bio         *string
}

func decodeProfileRequest(body io.Reader) (profileRequest, ValidationErrors, error) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var encoded struct {
		Username    json.RawMessage `json:"username"`
		DisplayName json.RawMessage `json:"display_name"`
		Bio         json.RawMessage `json:"bio"`
	}
	if err := decoder.Decode(&encoded); err != nil {
		return profileRequest{}, nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return profileRequest{}, nil, errors.New("request body must contain one JSON object")
	}
	if len(encoded.Username) == 0 || len(encoded.DisplayName) == 0 {
		return profileRequest{}, nil, errors.New("username and display_name are required")
	}
	request := profileRequest{}
	validation := ValidationErrors{}
	if err := json.Unmarshal(encoded.Username, &request.Username); err != nil || string(encoded.Username) == "null" {
		validation["username"] = "must be a string"
	}
	if err := json.Unmarshal(encoded.DisplayName, &request.DisplayName); err != nil || string(encoded.DisplayName) == "null" {
		validation["display_name"] = "must be a string"
	}
	if len(encoded.Bio) != 0 {
		var bio string
		if err := json.Unmarshal(encoded.Bio, &bio); err != nil || string(encoded.Bio) == "null" {
			validation["bio"] = "must be a string"
		} else {
			request.Bio = &bio
		}
	}
	if len(validation) != 0 {
		return profileRequest{}, validation, nil
	}
	return request, nil, nil
}

func (h *Handler) PutAvatar(c *gin.Context) {
	// Leave room for multipart framing while preventing arbitrarily large bodies
	// from being buffered to disk before the image limit is checked.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6*1024*1024)
	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse("INVALID_AVATAR", ErrInvalidAvatar.Error(), nil))
		return
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, 5*1024*1024+1))
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse("AVATAR_UPLOAD_FAILED", "Avatar upload failed.", nil))
		return
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(contents)
	}
	profile, err := h.service.UpdateAvatar(c.Request.Context(), c.GetString(AuthenticatedUserIDKey), contentType, contents)
	switch {
	case errors.Is(err, ErrInvalidAvatar):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("INVALID_AVATAR", err.Error(), nil))
	case err != nil:
		c.JSON(http.StatusBadGateway, errorResponse("AVATAR_UPLOAD_FAILED", "Avatar upload failed.", nil))
	default:
		c.JSON(http.StatusOK, profileResponse(profile))
	}
}

func profileResponse(profile Profile) gin.H {
	return gin.H{"data": gin.H{
		"id": profile.ID, "username": profile.Username, "display_name": profile.DisplayName,
		"bio": profile.Bio, "avatar_url": profile.AvatarURL, "created_at": profile.CreatedAt, "updated_at": profile.UpdatedAt,
	}}
}

func errorResponse(code, message string, fields ValidationErrors) gin.H {
	detail := gin.H{"code": code, "message": message}
	if fields != nil {
		detail["fields"] = fields
	}
	return gin.H{"error": detail}
}

func internalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "An internal error occurred.", nil))
}
