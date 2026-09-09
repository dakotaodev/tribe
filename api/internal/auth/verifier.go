// Package auth verifies Supabase access tokens and exposes the authenticated
// application identity to transport code.
package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwksCacheLifetime = 10 * time.Minute

var (
	ErrInvalidToken  = errors.New("invalid authentication token")
	errInvalidConfig = errors.New("invalid Supabase JWT configuration")
)

// Identity is the verified Supabase subject. UserID maps directly to
// public.users.id, which is constrained to auth.users.id by the database.
type Identity struct {
	UserID string
}

// Config contains the expected, server-owned attributes of Supabase access
// tokens. JWKSURL must point to the issuer's public signing keys.
type Config struct {
	Issuer   string
	Audience string
	JWKSURL  string
}

// Verifier validates tokens against the configured Supabase JWKS.
type Verifier struct {
	issuer   string
	audience string
	jwksURL  string
	client   *http.Client
	now      func() time.Time

	mu        sync.Mutex
	keys      map[string]verificationKey
	fetchedAt time.Time
}

type verificationKey struct {
	algorithm string
	key       crypto.PublicKey
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Algorithm string `json:"alg"`
	Curve     string `json:"crv"`
	N         string `json:"n"`
	E         string `json:"e"`
	X         string `json:"x"`
	Y         string `json:"y"`
}

type accessClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// NewVerifier constructs a verifier for asymmetric Supabase signing keys.
func NewVerifier(config Config, client *http.Client) (*Verifier, error) {
	issuer, err := validateURL(config.Issuer)
	if err != nil {
		return nil, fmt.Errorf("%w: issuer: %v", errInvalidConfig, err)
	}
	jwksURL, err := validateURL(config.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("%w: JWKS URL: %v", errInvalidConfig, err)
	}
	if strings.TrimSpace(config.Audience) == "" {
		return nil, fmt.Errorf("%w: audience is required", errInvalidConfig)
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	return &Verifier{
		issuer: issuer.String(), audience: config.Audience, jwksURL: jwksURL.String(),
		client: client, now: time.Now,
	}, nil
}

// Verify returns an identity only after the token's signature and required
// issuer, audience, expiry, and UUID subject claims have been validated.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (Identity, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Identity{}, ErrInvalidToken
	}

	unsigned, _, err := new(jwt.Parser).ParseUnverified(rawToken, jwt.MapClaims{})
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	kid, _ := unsigned.Header["kid"].(string)
	if kid == "" {
		return Identity{}, ErrInvalidToken
	}

	key, err := v.keyFor(ctx, kid)
	if err != nil {
		return Identity{}, ErrInvalidToken
	}

	claims := accessClaims{}
	parsed, err := jwt.ParseWithClaims(rawToken, &claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != key.algorithm {
			return nil, ErrInvalidToken
		}
		return key.key, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg(), jwt.SigningMethodRS384.Alg(), jwt.SigningMethodRS512.Alg(), jwt.SigningMethodES256.Alg(), jwt.SigningMethodES384.Alg(), jwt.SigningMethodES512.Alg()}),
		jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience), jwt.WithExpirationRequired(), jwt.WithTimeFunc(v.now),
	)
	if err != nil || !parsed.Valid || claims.Role != "authenticated" || !isUUID(claims.Subject) {
		return Identity{}, ErrInvalidToken
	}

	return Identity{UserID: claims.Subject}, nil
}

func (v *Verifier) keyFor(ctx context.Context, keyID string) (verificationKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.keys == nil || v.now().Sub(v.fetchedAt) >= jwksCacheLifetime {
		if err := v.refreshKeys(ctx); err != nil {
			return verificationKey{}, err
		}
	}
	key, ok := v.keys[keyID]
	if ok {
		return key, nil
	}

	// A new key may have appeared since the cache was populated during key
	// rotation. Refresh once before treating the token as invalid.
	if err := v.refreshKeys(ctx); err != nil {
		return verificationKey{}, err
	}
	key, ok = v.keys[keyID]
	if !ok {
		return verificationKey{}, ErrInvalidToken
	}
	return key, nil
}

func (v *Verifier) refreshKeys(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	response, err := v.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS response status %d", response.StatusCode)
	}

	var document jwksDocument
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	keys := make(map[string]verificationKey, len(document.Keys))
	for _, rawKey := range document.Keys {
		key, err := rawKey.publicKey()
		if err != nil || rawKey.KeyID == "" || rawKey.Algorithm == "" {
			continue
		}
		keys[rawKey.KeyID] = verificationKey{algorithm: rawKey.Algorithm, key: key}
	}
	if len(keys) == 0 {
		return errors.New("JWKS contains no supported signing keys")
	}
	v.keys = keys
	v.fetchedAt = v.now()
	return nil
}

func (rawKey jwk) publicKey() (crypto.PublicKey, error) {
	switch rawKey.KeyType {
	case "RSA":
		if !strings.HasPrefix(rawKey.Algorithm, "RS") {
			return nil, errors.New("unsupported RSA algorithm")
		}
		n, err := decodeBase64URLInteger(rawKey.N)
		if err != nil {
			return nil, err
		}
		e, err := decodeBase64URLInteger(rawKey.E)
		if err != nil || !e.IsInt64() || e.Int64() < 3 {
			return nil, errors.New("invalid RSA exponent")
		}
		return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
	case "EC":
		curve := map[string]elliptic.Curve{"P-256": elliptic.P256(), "P-384": elliptic.P384(), "P-521": elliptic.P521()}[rawKey.Curve]
		if curve == nil || !strings.HasPrefix(rawKey.Algorithm, "ES") {
			return nil, errors.New("unsupported elliptic-curve key")
		}
		x, err := decodeBase64URLInteger(rawKey.X)
		if err != nil {
			return nil, err
		}
		y, err := decodeBase64URLInteger(rawKey.Y)
		if err != nil || !curve.IsOnCurve(x, y) {
			return nil, errors.New("invalid elliptic-curve point")
		}
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
	default:
		return nil, errors.New("unsupported key type")
	}
}

func decodeBase64URLInteger(value string) (*big.Int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) == 0 {
		return nil, errors.New("invalid base64url integer")
	}
	return new(big.Int).SetBytes(decoded), nil
}

func validateURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("must be an absolute URL")
	}
	return parsed, nil
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') && !(character >= 'A' && character <= 'F') {
			return false
		}
	}
	return true
}
