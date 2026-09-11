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
	profile := Profile{ID: id, Username: update.Username, DisplayName: update.DisplayName, Bio: *update.Bio}
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
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Body.String() != `{"error":{"code":"PROFILE_ONBOARDING_REQUIRED","message":"Create your profile to continue."}}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestPutMeMutatesOnlyAuthenticatedCaller(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{
		"caller": {ID: "caller", Username: "caller_user", Bio: "old bio"},
		"other":  {ID: "other", Username: "other_user"},
	}, updates: map[string]Update{}}
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

func TestPutMeReturnsCreatedAndStandardEnvelopeForNewProfile(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{}, updates: map[string]Update{}}
	response := performRequest(NewHandler(NewService(repository)), http.MethodPut,
		map[string]any{"username": "new_user", "display_name": "New User"}, "caller")
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Data Profile `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.ID != "caller" || body.Data.Bio != "" {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestPutMePreservesOmittedBioWhenUpdating(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{
		"caller": {ID: "caller", Username: "caller", DisplayName: "Caller", Bio: "keep me"},
	}, updates: map[string]Update{}}
	response := performRequest(NewHandler(NewService(repository)), http.MethodPut,
		map[string]any{"username": "caller", "display_name": "Updated Caller"}, "caller")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := repository.profiles["caller"].Bio; got != "keep me" {
		t.Fatalf("bio = %q, want preserved value", got)
	}
}

func TestPutMeRejectsUnknownUserIDWithoutMutation(t *testing.T) {
	repository := &stubRepository{profiles: map[string]Profile{
		"caller": {ID: "caller", Username: "caller", DisplayName: "Caller"},
		"other":  {ID: "other", Username: "other", DisplayName: "Other"},
	}, updates: map[string]Update{}}
	response := performRequest(NewHandler(NewService(repository)), http.MethodPut,
		map[string]any{"username": "changed", "display_name": "Changed", "user_id": "other"}, "caller")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(repository.updates) != 0 || repository.profiles["other"].Username != "other" {
		t.Fatalf("unexpected mutation: %#v", repository.updates)
	}
}

func TestPutMeRejectsMalformedAndMissingRequiredFields(t *testing.T) {
	for name, body := range map[string]string{
		"malformed":        `{"username":`,
		"missing username": `{"display_name":"Caller"}`,
		"multiple values":  `{"username":"caller","display_name":"Caller"}{}`,
	} {
		t.Run(name, func(t *testing.T) {
			repository := &stubRepository{profiles: map[string]Profile{}, updates: map[string]Update{}}
			response := performRawRequest(NewHandler(NewService(repository)), body, "caller")
			if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"INVALID_REQUEST"`)) {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestPutMeRejectsWrongFieldTypeAsValidationFailure(t *testing.T) {
	for name, body := range map[string]string{
		"numeric username": `{"username":7,"display_name":"Caller"}`,
		"null bio":         `{"username":"caller","display_name":"Caller","bio":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			repository := &stubRepository{profiles: map[string]Profile{}, updates: map[string]Update{}}
			response := performRawRequest(NewHandler(NewService(repository)), body, "caller")
			if response.Code != http.StatusUnprocessableEntity || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"VALIDATION_FAILED"`)) {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
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
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error.Code != "USERNAME_TAKEN" {
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
	if !bytes.Contains(response.Body.Bytes(), []byte(`"code":"AVATAR_UPLOAD_FAILED"`)) {
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

func performRawRequest(handler *Handler, body, caller string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(AuthenticatedUserIDKey, caller) })
	router.PUT("/me", handler.PutMe)
	request := httptest.NewRequest(http.MethodPut, "/me", bytes.NewBufferString(body))
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
