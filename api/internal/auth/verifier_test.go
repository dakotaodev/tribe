package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "https://tribe.test/auth/v1"
	testAudience = "authenticated"
	testUserID   = "2a0f741d-d063-4337-8e45-d9edc2a234ab"
)

func TestVerifierAcceptsValidSupabaseToken(t *testing.T) {
	key := newSigningKey(t)
	server := newJWKSServer(t, key)
	defer server.Close()
	verifier := newTestVerifier(t, server.URL, time.Now)

	identity, err := verifier.Verify(context.Background(), signedToken(t, key, testIssuer, testAudience, time.Now().Add(time.Hour), testUserID))

	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if identity.UserID != testUserID {
		t.Fatalf("UserID = %q, want %q", identity.UserID, testUserID)
	}
}

func TestVerifierRejectsInvalidClaimsAndSignature(t *testing.T) {
	key := newSigningKey(t)
	otherKey := newSigningKey(t)
	server := newJWKSServer(t, key)
	defer server.Close()
	verifier := newTestVerifier(t, server.URL, time.Now)

	testCases := []struct {
		name  string
		token string
	}{
		{"expired", signedToken(t, key, testIssuer, testAudience, time.Now().Add(-time.Minute), testUserID)},
		{"wrong issuer", signedToken(t, key, "https://other.test/auth/v1", testAudience, time.Now().Add(time.Hour), testUserID)},
		{"wrong audience", signedToken(t, key, testIssuer, "another-audience", time.Now().Add(time.Hour), testUserID)},
		{"invalid subject", signedToken(t, key, testIssuer, testAudience, time.Now().Add(time.Hour), "not-a-uuid")},
		{"non-user role", signedTokenWithRole(t, key, "test-key", testIssuer, testAudience, time.Now().Add(time.Hour), testUserID, "service_role")},
		{"wrong signature", signedToken(t, otherKey, testIssuer, testAudience, time.Now().Add(time.Hour), testUserID)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := verifier.Verify(context.Background(), testCase.token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestVerifierRefreshesJWKSForUnknownKeyID(t *testing.T) {
	firstKey := newSigningKey(t)
	secondKey := newSigningKey(t)
	currentKey := firstKey
	currentKeyID := "first-key"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/.well-known/jwks.json" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(jwksDocument{Keys: []jwk{publicJWKWithID(currentKey, currentKeyID)}})
	}))
	defer server.Close()
	verifier := newTestVerifier(t, server.URL, time.Now)

	if _, err := verifier.Verify(context.Background(), signedTokenWithKeyID(t, firstKey, "first-key", testIssuer, testAudience, time.Now().Add(time.Hour), testUserID)); err != nil {
		t.Fatalf("initial Verify() error = %v", err)
	}
	currentKey = secondKey
	currentKeyID = "second-key"
	if _, err := verifier.Verify(context.Background(), signedTokenWithKeyID(t, secondKey, "second-key", testIssuer, testAudience, time.Now().Add(time.Hour), testUserID)); err != nil {
		t.Fatalf("Verify() after key rotation error = %v", err)
	}
}

func newSigningKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	return key
}

func newJWKSServer(t *testing.T, key *rsa.PrivateKey) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/.well-known/jwks.json" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(jwksDocument{Keys: []jwk{publicJWK(key)}})
	}))
}

func newTestVerifier(t *testing.T, jwksBaseURL string, now func() time.Time) *Verifier {
	t.Helper()
	verifier, err := NewVerifier(Config{Issuer: testIssuer, Audience: testAudience, JWKSURL: jwksBaseURL + "/.well-known/jwks.json"}, nil)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	verifier.now = now
	return verifier
}

func signedToken(t *testing.T, key *rsa.PrivateKey, issuer, audience string, expiresAt time.Time, subject string) string {
	return signedTokenWithKeyID(t, key, "test-key", issuer, audience, expiresAt, subject)
}

func signedTokenWithKeyID(t *testing.T, key *rsa.PrivateKey, keyID, issuer, audience string, expiresAt time.Time, subject string) string {
	return signedTokenWithRole(t, key, keyID, issuer, audience, expiresAt, subject, "authenticated")
}

func signedTokenWithRole(t *testing.T, key *rsa.PrivateKey, keyID, issuer, audience string, expiresAt time.Time, subject, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer, Subject: subject, Audience: []string{audience}, ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Role: role,
	})
	token.Header["kid"] = keyID
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return signed
}

func publicJWK(key *rsa.PrivateKey) jwk {
	return publicJWKWithID(key, "test-key")
}

func publicJWKWithID(key *rsa.PrivateKey, keyID string) jwk {
	return jwk{
		KeyID: keyID, KeyType: "RSA", Algorithm: jwt.SigningMethodRS256.Alg(),
		N: base64.RawURLEncoding.EncodeToString(key.N.Bytes()), E: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}
