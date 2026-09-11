package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubRepository struct {
	profiles map[string]Profile
	updates  map[string]Update
}

type stubAvatarStore struct {
	putKey, deletedKey string
	putErr             error
}

func (s *stubAvatarStore) Put(_ context.Context, key, _ string, _ []byte) error {
	s.putKey = key
	return s.putErr
}
func (s *stubAvatarStore) Delete(_ context.Context, key string) error { s.deletedKey = key; return nil }
func (s *stubAvatarStore) SignedURL(_ context.Context, key string) (string, error) {
	return "https://signed.example/" + key, nil
}

func (r *stubRepository) Get(_ context.Context, id string) (Profile, error) {
	profile, ok := r.profiles[id]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return profile, nil
}
func (r *stubRepository) Upsert(_ context.Context, id string, update Update) (Profile, error) {
	for owner, profile := range r.profiles {
		if owner != id && profile.Username == update.Username {
			return Profile{}, ErrUsernameTaken
		}
	}
	r.updates[id] = update
	profile := Profile{ID: id, Username: update.Username, DisplayName: update.DisplayName, Bio: update.Bio}
	r.profiles[id] = profile
	return profile, nil
}
func (r *stubRepository) SetAvatar(_ context.Context, id, path string) (Profile, error) {
	profile, ok := r.profiles[id]
	if !ok {
		return Profile{}, ErrNotFound
	}
	profile.AvatarPath = &path
	r.profiles[id] = profile
	return profile, nil
}

func TestGetMeReturnsOnboardingState(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{}, updates: map[string]Update{}}
	response := performRequest(NewHandler(NewService(repository)), http.MethodGet, nil, "caller")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Body.String() != `{"profile":null,"profile_complete":false}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestPutMeMutatesOnlyAuthenticatedCaller(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{"other": {ID: "other", Username: "other_user"}}, updates: map[string]Update{}}
	response := performRequest(NewHandler(NewService(repository)), http.MethodPut,
		map[string]any{"username": " New_User ", "display_name": " New Name ", "bio": " Hello "}, "caller")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if _, ok := repository.updates["other"]; ok {
		t.Fatal("another user's profile was updated")
	}
	if repository.updates["caller"].Username != "new_user" {
		t.Fatalf("update = %#v", repository.updates["caller"])
	}
}

func TestPutMeRejectsDuplicateUsername(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{"other": {ID: "other", Username: "taken_name"}}, updates: map[string]Update{}}
	response := performRequest(NewHandler(NewService(repository)), http.MethodPut,
		map[string]any{"username": "taken_name", "display_name": "Caller", "bio": ""}, "caller")
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error.Code != "username_taken" {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestPutMeReturnsFieldValidationErrors(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{}, updates: map[string]Update{}}
	response := performRequest(NewHandler(NewService(repository)), http.MethodPut,
		map[string]any{"username": "NO", "display_name": "", "bio": ""}, "caller")
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"username"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"display_name"`)) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestPutAvatarAssociatesOnlyAuthenticatedCallerAndDeletesOldObject(t *testing.T) {
	oldPath := "caller/old.jpg"
	repository := &stubRepository{profiles: map[string]Profile{
		"caller": {ID: "caller", Username: "caller", AvatarPath: &oldPath},
		"other":  {ID: "other", Username: "other"},
	}, updates: map[string]Update{}}
	store := &stubAvatarStore{}
	service := NewService(repository, store)
	service.newKey = func(owner, extension string) (string, error) { return owner + "/new." + extension, nil }

	response := performAvatarRequest(NewHandler(service), "caller", "image/jpeg", []byte{0xff, 0xd8, 0xff, 0xe0})

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if store.putKey != "caller/new.jpg" || store.deletedKey != oldPath {
		t.Fatalf("uploaded = %q, deleted = %q", store.putKey, store.deletedKey)
	}
	if repository.profiles["other"].AvatarPath != nil {
		t.Fatal("another user's avatar association was updated")
	}
	if got := *repository.profiles["caller"].AvatarPath; got != "caller/new.jpg" {
		t.Fatalf("caller avatar = %q", got)
	}
	if bytes.Contains(response.Body.Bytes(), []byte("avatar_path")) {
		t.Fatalf("private storage path leaked: %s", response.Body.String())
	}
}

func TestPutAvatarSurfacesStorageFailure(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{"caller": {ID: "caller"}}, updates: map[string]Update{}}
	store := &stubAvatarStore{putErr: errors.New("storage unavailable")}
	response := performAvatarRequest(NewHandler(NewService(repository, store)), "caller", "image/png", []byte("\x89PNG\r\n\x1a\n"))
	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"code":"avatar_upload_failed"`)) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestPutAvatarRejectsUnsupportedContentType(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{"caller": {ID: "caller"}}, updates: map[string]Update{}}
	response := performAvatarRequest(NewHandler(NewService(repository, &stubAvatarStore{})), "caller", "image/gif", []byte("image"))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func performRequest(handler *Handler, method string, body any, caller string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(AuthenticatedUserIDKey, caller) })
	router.GET("/me", handler.GetMe)
	router.PUT("/me", handler.PutMe)
	var encoded []byte
	if body != nil {
		encoded, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, "/me", bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func performAvatarRequest(handler *Handler, caller, contentType string, contents []byte) *httptest.ResponseRecorder {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="avatar"; filename="avatar"`)
	header.Set("Content-Type", contentType)
	part, _ := writer.CreatePart(header)
	_, _ = part.Write(contents)
	_ = writer.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(AuthenticatedUserIDKey, caller) })
	router.PUT("/me/avatar", handler.PutAvatar)
	request := httptest.NewRequest(http.MethodPut, "/me/avatar", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
