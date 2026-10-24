// Package neonauth verifies JWTs issued by Neon Auth (Neon's managed Better
// Auth, T158). Neon Auth signs short-lived (15-minute) EdDSA/Ed25519 tokens;
// the public keys are served at <NEON_AUTH_BASE_URL>/.well-known/jwks.json and
// both `iss` and `aud` are the origin of the Neon Auth URL.
// See https://neon.com/docs/auth/guides/plugins/jwt.
package neonauth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// jwksRefreshInterval is the minimum time between JWKS fetches, so a
	// stream of tokens with an unknown kid cannot hammer Neon.
	jwksRefreshInterval = time.Minute
	// leeway tolerates small clock skew on exp/nbf/iat.
	leeway = 30 * time.Second
)

// ErrUnknownKey is returned when no JWKS key matches the token's kid.
var ErrUnknownKey = errors.New("neonauth: no signing key for kid")

// Claims are the Neon Auth JWT claims the API uses. Subject is the Neon Auth
// user id (users.auth_user_id).
type Claims struct {
	jwt.RegisteredClaims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"emailVerified,omitempty"`
	Banned        bool   `json:"banned,omitempty"`
}

// Verifier validates Neon Auth JWTs against the branch's JWKS.
type Verifier struct {
	issuer  string
	jwksURL string
	client  *http.Client

	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	fetchedAt time.Time
}

// New builds a Verifier for a Neon Auth base URL (NEON_AUTH_BASE_URL, e.g.
// https://ep-x.neonauth.<region>.aws.neon.tech/neondb/auth). jwksURL may be
// empty, in which case <baseURL>/.well-known/jwks.json is used.
func New(baseURL, jwksURL string) (*Verifier, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("neonauth: invalid base URL")
	}
	if jwksURL == "" {
		jwksURL = u.String() + "/.well-known/jwks.json"
	}
	return &Verifier{
		issuer:  u.Scheme + "://" + u.Host,
		jwksURL: jwksURL,
		client:  &http.Client{Timeout: 5 * time.Second},
		keys:    map[string]ed25519.PublicKey{},
	}, nil
}

// IsNeonToken reports whether a JWT's header says EdDSA, i.e. it is a Neon
// Auth token rather than one of the API's own HS256 tokens. It does not
// verify anything.
func IsNeonToken(token string) bool {
	header, _, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(header)
	if err != nil {
		return false
	}
	var h struct {
		Alg string `json:"alg"`
	}
	return json.Unmarshal(raw, &h) == nil && h.Alg == "EdDSA"
}

// Verify checks the signature, issuer, audience and expiry of a Neon Auth
// JWT and returns its claims. The subject is required and banned users are
// rejected.
func (v *Verifier) Verify(ctx context.Context, token string) (*Claims, error) {
	claims := &Claims{}
	keyFor := func(t *jwt.Token) (any, error) { return v.keyFor(ctx, t) }
	_, err := jwt.ParseWithClaims(token, claims, keyFor,
		jwt.WithValidMethods([]string{"EdDSA"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(leeway),
	)
	if err != nil {
		return nil, err
	}
	if claims.Subject == "" {
		return nil, errors.New("neonauth: token has no subject")
	}
	if claims.Banned {
		return nil, errors.New("neonauth: user is banned")
	}
	return claims, nil
}

func (v *Verifier) keyFor(ctx context.Context, t *jwt.Token) (any, error) {
	kid, _ := t.Header["kid"].(string)
	if kid == "" {
		return nil, errors.New("neonauth: token has no kid")
	}
	if key, ok := v.cachedKey(kid); ok {
		return key, nil
	}
	// Unknown kid: Neon may have rotated keys. Refetch, at most once a minute.
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	if key, ok := v.cachedKey(kid); ok {
		return key, nil
	}
	return nil, ErrUnknownKey
}

func (v *Verifier) cachedKey(kid string) (ed25519.PublicKey, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[kid]
	return key, ok
}

type jwkSet struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Crv string `json:"crv"`
		X   string `json:"x"`
	} `json:"keys"`
}

func (v *Verifier) refresh(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.fetchedAt.IsZero() && time.Since(v.fetchedAt) < jwksRefreshInterval {
		return nil
	}
	v.fetchedAt = time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("neonauth: build JWKS request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("neonauth: fetch JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("neonauth: fetch JWKS: status %d", resp.StatusCode)
	}
	var set jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("neonauth: decode JWKS: %w", err)
	}
	keys := make(map[string]ed25519.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" || k.Kid == "" {
			continue
		}
		x, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(x) != ed25519.PublicKeySize {
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(x)
	}
	v.keys = keys
	return nil
}
