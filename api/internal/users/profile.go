package users

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound      = errors.New("profile not found")
	ErrUsernameTaken = errors.New("username already taken")
	usernamePattern  = regexp.MustCompile(`^[a-z0-9_]+$`)
)

type Profile struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Bio         string    `json:"bio"`
	AvatarURL   *string   `json:"avatar_url"`
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
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Get(ctx context.Context, callerID string) (Profile, error) {
	return s.repository.Get(ctx, callerID)
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

	return s.repository.Upsert(ctx, callerID, input)
}
