package neonauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// fakeNeon serves a JWKS for one Ed25519 key and signs tokens with it, the
// way Neon Auth does (iss = aud = the Neon Auth origin).
type fakeNeon struct {
	srv     *httptest.Server
	kid     string
	priv    ed25519.PrivateKey
	fetches atomic.Int32
}

func newFakeNeon(t *testing.T) *fakeNeon {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	f := &fakeNeon{kid: "kid-1", priv: priv}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/neondb/auth/.well-known/jwks.json", r.URL.Path)
		f.fetches.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": f.kid, "kty": "OKP", "crv": "Ed25519", "alg": "EdDSA",
			"x": base64.RawURLEncoding.EncodeToString(pub),
		}}})
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeNeon) baseURL() string { return f.srv.URL + "/neondb/auth" }

func (f *fakeNeon) sign(t *testing.T, mutate func(jwt.MapClaims, *jwt.Token)) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user-123", "email": "a@example.com", "emailVerified": true, "banned": false,
		"iss": f.srv.URL, "aud": f.srv.URL,
		"iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tok.Header["kid"] = f.kid
	if mutate != nil {
		mutate(claims, tok)
	}
	s, err := tok.SignedString(f.priv)
	require.NoError(t, err)
	return s
}

func newVerifier(t *testing.T, f *fakeNeon) *Verifier {
	t.Helper()
	v, err := New(f.baseURL(), "")
	require.NoError(t, err)
	return v
}

func TestVerify_ValidToken(t *testing.T) {
	f := newFakeNeon(t)
	v := newVerifier(t, f)

	claims, err := v.Verify(context.Background(), f.sign(t, nil))
	require.NoError(t, err)
	require.Equal(t, "user-123", claims.Subject)
	require.Equal(t, "a@example.com", claims.Email)
	require.True(t, claims.EmailVerified)

	// Keys are cached: a second token does not refetch the JWKS.
	_, err = v.Verify(context.Background(), f.sign(t, nil))
	require.NoError(t, err)
	require.Equal(t, int32(1), f.fetches.Load())
}

func TestVerify_Rejects(t *testing.T) {
	f := newFakeNeon(t)
	other := newFakeNeon(t) // a different key pair and origin

	cases := map[string]string{
		"wrong issuer":   f.sign(t, func(c jwt.MapClaims, _ *jwt.Token) { c["iss"] = "https://evil.example" }),
		"wrong audience": f.sign(t, func(c jwt.MapClaims, _ *jwt.Token) { c["aud"] = "https://evil.example" }),
		"expired":        f.sign(t, func(c jwt.MapClaims, _ *jwt.Token) { c["exp"] = time.Now().Add(-time.Hour).Unix() }),
		"no expiry":      f.sign(t, func(c jwt.MapClaims, _ *jwt.Token) { delete(c, "exp") }),
		"no subject":     f.sign(t, func(c jwt.MapClaims, _ *jwt.Token) { delete(c, "sub") }),
		"banned":         f.sign(t, func(c jwt.MapClaims, _ *jwt.Token) { c["banned"] = true }),
		"unknown kid":    f.sign(t, func(_ jwt.MapClaims, tok *jwt.Token) { tok.Header["kid"] = "rotated-away" }),
		"no kid":         f.sign(t, func(_ jwt.MapClaims, tok *jwt.Token) { delete(tok.Header, "kid") }),
		// Signed by another key that claims our kid and origin.
		"bad signature": other.sign(t, func(c jwt.MapClaims, tok *jwt.Token) {
			c["iss"], c["aud"] = f.srv.URL, f.srv.URL
			tok.Header["kid"] = f.kid
		}),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newVerifier(t, f).Verify(context.Background(), token)
			require.Error(t, err)
		})
	}
}

func TestVerify_RejectsHS256(t *testing.T) {
	f := newFakeNeon(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123", "iss": f.srv.URL, "aud": f.srv.URL, "exp": time.Now().Add(time.Minute).Unix(),
	})
	tok.Header["kid"] = f.kid
	s, err := tok.SignedString([]byte("secret"))
	require.NoError(t, err)

	_, err = newVerifier(t, f).Verify(context.Background(), s)
	require.Error(t, err)
	require.False(t, IsNeonToken(s))
}

func TestVerify_UnknownKidRefetchesAtMostOncePerInterval(t *testing.T) {
	f := newFakeNeon(t)
	v := newVerifier(t, f)
	unknown := f.sign(t, func(_ jwt.MapClaims, tok *jwt.Token) { tok.Header["kid"] = "nope" })

	for range 5 {
		_, err := v.Verify(context.Background(), unknown)
		require.ErrorIs(t, err, ErrUnknownKey)
	}
	require.Equal(t, int32(1), f.fetches.Load())
}

func TestIsNeonToken(t *testing.T) {
	f := newFakeNeon(t)
	require.True(t, IsNeonToken(f.sign(t, nil)))
	require.False(t, IsNeonToken("not-a-jwt"))
	require.False(t, IsNeonToken("%%%.x.y"))
}

func TestNew_DerivesIssuerAndJWKS(t *testing.T) {
	v, err := New("https://ep-x.neonauth.c-7.us-east-2.aws.neon.tech/neondb/auth/", "")
	require.NoError(t, err)
	require.Equal(t, "https://ep-x.neonauth.c-7.us-east-2.aws.neon.tech", v.issuer)
	require.Equal(t, "https://ep-x.neonauth.c-7.us-east-2.aws.neon.tech/neondb/auth/.well-known/jwks.json", v.jwksURL)

	_, err = New("not a url", "")
	require.Error(t, err)
}

// TestVerify_LiveNeon checks a real Neon Auth token. It runs only when
// NEON_AUTH_TEST_BASE_URL (a branch's Neon Auth URL) and NEON_AUTH_TEST_TOKEN
// (a fresh JWT from <base>/token) are set.
func TestVerify_LiveNeon(t *testing.T) {
	base, token := os.Getenv("NEON_AUTH_TEST_BASE_URL"), os.Getenv("NEON_AUTH_TEST_TOKEN")
	if base == "" || token == "" {
		t.Skip("NEON_AUTH_TEST_BASE_URL / NEON_AUTH_TEST_TOKEN not set")
	}
	v, err := New(base, "")
	require.NoError(t, err)
	claims, err := v.Verify(context.Background(), token)
	require.NoError(t, err)
	require.NotEmpty(t, claims.Subject)
}
