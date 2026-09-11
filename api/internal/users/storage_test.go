package users

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseAvatarStoreUploadsPrivateObjectWithServerCredentials(t *testing.T) {
	var receivedPath, receivedAuthorization, receivedContentType, receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		receivedPath = request.URL.EscapedPath()
		receivedAuthorization = request.Header.Get("Authorization")
		receivedContentType = request.Header.Get("Content-Type")
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewSupabaseAvatarStore(server.URL, "server-secret", server.Client())
	err := store.Put(context.Background(), "caller/avatar name.png", "image/png", []byte("contents"))

	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if receivedPath != "/storage/v1/object/avatars/caller/avatar%20name.png" {
		t.Fatalf("path = %q", receivedPath)
	}
	if receivedAuthorization != "Bearer server-secret" || receivedContentType != "image/png" || receivedBody != "contents" {
		t.Fatalf("authorization = %q, content type = %q, body = %q", receivedAuthorization, receivedContentType, receivedBody)
	}
}

func TestSupabaseAvatarStoreSurfacesUploadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := NewSupabaseAvatarStore(server.URL, "secret", server.Client()).Put(
		context.Background(), "caller/avatar.png", "image/png", []byte("contents"),
	)
	if err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("Put() error = %v", err)
	}
}

func TestSupabaseAvatarStoreBuildsSignedURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/storage/v1/object/sign/avatars/caller/avatar.png" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"signedURL":"/storage/v1/object/sign/avatars/token"}`))
	}))
	defer server.Close()

	got, err := NewSupabaseAvatarStore(server.URL, "secret", server.Client()).SignedURL(context.Background(), "caller/avatar.png")
	if err != nil {
		t.Fatalf("SignedURL() error = %v", err)
	}
	if want := server.URL + "/storage/v1/object/sign/avatars/token"; got != want {
		t.Fatalf("SignedURL() = %q, want %q", got, want)
	}
}
