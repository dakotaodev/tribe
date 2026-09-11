package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubRepository struct {
	profiles map[string]Profile
	updates  map[string]Update
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
