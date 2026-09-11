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

const avatarBucket = "avatars"

type SupabaseAvatarStore struct {
	baseURL, secretKey string
	client             *http.Client
}

func NewSupabaseAvatarStore(baseURL, secretKey string, client *http.Client) *SupabaseAvatarStore {
	return &SupabaseAvatarStore{baseURL: strings.TrimRight(baseURL, "/"), secretKey: secretKey, client: client}
}

func (s *SupabaseAvatarStore) Put(ctx context.Context, key, contentType string, contents []byte) error {
	request, err := s.request(ctx, http.MethodPost, "/storage/v1/object/"+avatarBucket+"/"+escapeKey(key), bytes.NewReader(contents))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", contentType)
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("storage returned status %d", response.StatusCode)
	}
	return nil
}

func (s *SupabaseAvatarStore) Delete(ctx context.Context, key string) error {
	body, _ := json.Marshal(map[string][]string{"prefixes": []string{key}})
	request, err := s.request(ctx, http.MethodDelete, "/storage/v1/object/"+avatarBucket, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("storage returned status %d", response.StatusCode)
	}
	return nil
}

func (s *SupabaseAvatarStore) SignedURL(ctx context.Context, key string) (string, error) {
	body, _ := json.Marshal(map[string]int{"expiresIn": 3600})
	request, err := s.request(ctx, http.MethodPost, "/storage/v1/object/sign/"+avatarBucket+"/"+escapeKey(key), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("storage returned status %d", response.StatusCode)
	}
	var result struct {
		SignedURL string `json:"signedURL"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.SignedURL == "" {
		return "", errors.New("storage returned no signed URL")
	}
	if strings.HasPrefix(result.SignedURL, "http") {
		return result.SignedURL, nil
	}
	return s.baseURL + result.SignedURL, nil
}

func (s *SupabaseAvatarStore) request(ctx context.Context, method, path string, body *bytes.Reader) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, body)
	if err == nil {
		request.Header.Set("apikey", s.secretKey)
		request.Header.Set("Authorization", "Bearer "+s.secretKey)
	}
	return request, err
}

func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
