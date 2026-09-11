package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type JWTVerifier struct {
	jwksURL, issuer, audience string
	client                    *http.Client
	mu                        sync.RWMutex
	keys                      map[string]*rsa.PublicKey
}

func NewJWTVerifier(jwksURL, issuer, audience string) *JWTVerifier {
	return &JWTVerifier{jwksURL: jwksURL, issuer: issuer, audience: audience,
		client: &http.Client{Timeout: 5 * time.Second}, keys: make(map[string]*rsa.PublicKey)}
}

func (v *JWTVerifier) Verify(ctx context.Context, encoded string) (string, error) {
	parts := strings.Split(encoded, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid access token")
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	var claims struct {
		Subject  string `json:"sub"`
		Issuer   string `json:"iss"`
		Audience any    `json:"aud"`
		Expires  int64  `json:"exp"`
	}
	if err := decodePart(parts[0], &header); err != nil || header.Algorithm != "RS256" {
		return "", errors.New("invalid access token")
	}
	if err := decodePart(parts[1], &claims); err != nil {
		return "", errors.New("invalid access token")
	}
	key := v.key(header.KeyID)
	if key == nil {
		if err := v.refresh(ctx); err != nil {
			return "", err
		}
		key = v.key(header.KeyID)
	}
	if key == nil {
		return "", errors.New("unknown signing key")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.New("invalid access token")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil || claims.Subject == "" ||
		claims.Issuer != v.issuer || claims.Expires <= time.Now().Unix() || !hasAudience(claims.Audience, v.audience) {
		return "", errors.New("invalid access token")
	}
	return claims.Subject, nil
}

func decodePart(encoded string, destination any) error {
	value, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(value, destination)
}

func hasAudience(value any, expected string) bool {
	switch audience := value.(type) {
	case string:
		return audience == expected
	case []any:
		for _, item := range audience {
			if item == expected {
				return true
			}
		}
	}
	return false
}

func (v *JWTVerifier) key(id string) *rsa.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.keys[id]
}

func (v *JWTVerifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	response, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("JWKS endpoint returned non-200 status")
	}
	var raw struct {
		Keys []struct {
			ID   string `json:"kid"`
			Type string `json:"kty"`
			N    string `json:"n"`
			E    string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		return err
	}
	newKeys := make(map[string]*rsa.PublicKey)
	for _, item := range raw.Keys {
		if item.Type != "RSA" {
			continue
		}
		n, errN := base64.RawURLEncoding.DecodeString(item.N)
		e, errE := base64.RawURLEncoding.DecodeString(item.E)
		if errN != nil || errE != nil {
			continue
		}
		exponent := 0
		for _, b := range e {
			exponent = exponent<<8 + int(b)
		}
		newKeys[item.ID] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}
	}
	if len(newKeys) == 0 {
		return errors.New("JWKS contained no RSA keys")
	}
	v.mu.Lock()
	v.keys = newKeys
	v.mu.Unlock()
	return nil
}
