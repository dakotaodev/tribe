package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound      = errors.New("profile not found")
	ErrUsernameTaken = errors.New("username already taken")
	ErrInvalidAvatar = errors.New("avatar must be a JPEG, PNG, or WebP image no larger than 5 MB")
	usernamePattern  = regexp.MustCompile(`^[a-z0-9_]+$`)
)

type Profile struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Bio         string    `json:"bio"`
	AvatarURL   *string   `json:"avatar_url"`
	AvatarPath  *string   `json:"avatar_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Update struct {
	Username    string
	DisplayName string
	Bio         string
}

type ValidationErrors map[string]string

func (e ValidationErrors) Error() string { return "profile validation failed" }

type Repository interface {
	Get(context.Context, string) (Profile, error)
	Upsert(context.Context, string, Update) (Profile, error)
	SetAvatar(context.Context, string, string) (Profile, error)
}

type AvatarStore interface {
	Put(context.Context, string, string, []byte) error
	Delete(context.Context, string) error
	SignedURL(context.Context, string) (string, error)
}

type Service struct {
	repository Repository
	avatars    AvatarStore
	newKey     func(string, string) (string, error)
}

func NewService(repository Repository, avatars ...AvatarStore) *Service {
	service := &Service{repository: repository, newKey: avatarKey}
	if len(avatars) != 0 {
		service.avatars = avatars[0]
	}
	return service
}

func (s *Service) Get(ctx context.Context, callerID string) (Profile, error) {
	profile, err := s.repository.Get(ctx, callerID)
	return s.withAvatarURL(ctx, profile, err)
}

func (s *Service) Update(ctx context.Context, callerID string, input Update) (Profile, error) {
	input.Username = strings.ToLower(strings.TrimSpace(input.Username))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Bio = strings.TrimSpace(input.Bio)

	validation := ValidationErrors{}
	if len(input.Username) < 3 || len(input.Username) > 30 || !usernamePattern.MatchString(input.Username) {
		validation["username"] = "must be 3-30 lowercase letters, numbers, or underscores"
	}
	if utf8.RuneCountInString(input.DisplayName) < 1 || utf8.RuneCountInString(input.DisplayName) > 80 {
		validation["display_name"] = "must be between 1 and 80 characters"
	}
	if utf8.RuneCountInString(input.Bio) > 500 {
		validation["bio"] = "must be at most 500 characters"
	}
	if len(validation) != 0 {
		return Profile{}, validation
	}

	profile, err := s.repository.Upsert(ctx, callerID, input)
	return s.withAvatarURL(ctx, profile, err)
}

func (s *Service) UpdateAvatar(ctx context.Context, callerID, contentType string, contents []byte) (Profile, error) {
	if s.avatars == nil {
		return Profile{}, errors.New("avatar storage is not configured")
	}
	detectedType := http.DetectContentType(contents)
	extension := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}[detectedType]
	if extension == "" || len(contents) == 0 || len(contents) > 5*1024*1024 {
		return Profile{}, ErrInvalidAvatar
	}
	if contentType != "" && contentType != detectedType {
		return Profile{}, ErrInvalidAvatar
	}
	key, err := s.newKey(callerID, extension)
	if err != nil {
		return Profile{}, err
	}
	if err := s.avatars.Put(ctx, key, detectedType, contents); err != nil {
		return Profile{}, fmt.Errorf("upload avatar: %w", err)
	}
	old, err := s.repository.Get(ctx, callerID)
	if err != nil {
		_ = s.avatars.Delete(ctx, key)
		return Profile{}, err
	}
	profile, err := s.repository.SetAvatar(ctx, callerID, key)
	if err != nil {
		_ = s.avatars.Delete(ctx, key)
		return Profile{}, err
	}
	if old.AvatarPath != nil && *old.AvatarPath != key {
		// Association is already durable, so stale-object cleanup is best effort.
		_ = s.avatars.Delete(ctx, *old.AvatarPath)
	}
	return s.withAvatarURL(ctx, profile, nil)
}

func (s *Service) withAvatarURL(ctx context.Context, profile Profile, err error) (Profile, error) {
	if err != nil || profile.AvatarPath == nil || s.avatars == nil {
		return profile, err
	}
	avatarURL, err := s.avatars.SignedURL(ctx, *profile.AvatarPath)
	if err != nil {
		return Profile{}, fmt.Errorf("sign avatar URL: %w", err)
	}
	profile.AvatarURL = &avatarURL
	return profile, nil
}

func avatarKey(callerID, extension string) (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return callerID + "/" + hex.EncodeToString(random[:]) + "." + extension, nil
}
