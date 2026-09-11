package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SupabaseRepository struct {
	baseURL, secretKey string
	client             *http.Client
}

func NewSupabaseRepository(baseURL, secretKey string, client *http.Client) *SupabaseRepository {
	return &SupabaseRepository{baseURL: strings.TrimRight(baseURL, "/"), secretKey: secretKey, client: client}
}

func (r *SupabaseRepository) Get(ctx context.Context, userID string) (Profile, error) {
	endpoint := r.baseURL + "/rest/v1/users?select=id,username,display_name,bio,created_at,updated_at&id=eq." + url.QueryEscape(userID)
	request, err := r.request(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Profile{}, err
	}
	response, err := r.client.Do(request)
	if err != nil {
		return Profile{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Profile{}, fmt.Errorf("read profile: status %d", response.StatusCode)
	}
	var profiles []Profile
	if err := json.NewDecoder(response.Body).Decode(&profiles); err != nil {
		return Profile{}, err
	}
	if len(profiles) == 0 {
		return Profile{}, ErrNotFound
	}
	return profiles[0], nil
}

func (r *SupabaseRepository) Upsert(ctx context.Context, userID string, input Update) (Profile, error) {
	body, err := json.Marshal(map[string]string{"id": userID, "username": input.Username, "display_name": input.DisplayName, "bio": input.Bio})
	if err != nil {
		return Profile{}, err
	}
	request, err := r.request(ctx, http.MethodPost, r.baseURL+"/rest/v1/users?on_conflict=id", bytes.NewReader(body))
	if err != nil {
		return Profile{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Prefer", "resolution=merge-duplicates,return=representation")
	response, err := r.client.Do(request)
	if err != nil {
		return Profile{}, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict {
		return Profile{}, ErrUsernameTaken
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Profile{}, fmt.Errorf("upsert profile: status %d", response.StatusCode)
	}
	var profiles []Profile
	if err := json.NewDecoder(response.Body).Decode(&profiles); err != nil {
		return Profile{}, err
	}
	if len(profiles) != 1 {
		return Profile{}, errors.New("upsert profile returned no profile")
	}
	return profiles[0], nil
}

func (r *SupabaseRepository) request(ctx context.Context, method, endpoint string, body *bytes.Reader) (*http.Request, error) {
	var requestBody any
	if body != nil {
		requestBody = body
	}
	reader, _ := requestBody.(*bytes.Reader)
	request, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err == nil {
		request.Header.Set("apikey", r.secretKey)
		request.Header.Set("Authorization", "Bearer "+r.secretKey)
	}
	return request, err
}
