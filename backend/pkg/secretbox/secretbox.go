// Package secretbox encrypts small secrets (OAuth tokens of third-party
// integrations, T192) for storage at rest with AES-256-GCM.
//
// The key comes from the environment (INTEGRATIONS_ENCRYPTION_KEY, 32 bytes,
// base64) and is never logged. Every value is sealed with associated data
// naming its owner (e.g. "quickbooks:access:<tenant id>"), so a ciphertext
// copied into another tenant's row or another column does not decrypt.
//
// Sealed format: "v1:" + base64(nonce || ciphertext || tag).
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const prefix = "v1:"

// KeySize is the AES-256 key length in bytes.
const KeySize = 32

var (
	// ErrInvalidKey is returned for a key that is not 32 bytes of base64.
	ErrInvalidKey = errors.New("secretbox: key must be 32 bytes, base64 encoded (openssl rand -base64 32)")
	// ErrDecrypt is returned when a value can't be opened (wrong key,
	// wrong associated data, tampered or malformed).
	ErrDecrypt = errors.New("secretbox: cannot decrypt value")
)

// Box seals and opens values with one key.
type Box struct {
	aead cipher.AEAD
	key  []byte
}

// New returns a Box for a raw 32-byte key.
func New(key []byte) (*Box, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	return &Box{aead: aead, key: append([]byte(nil), key...)}, nil
}

// FromBase64 returns a Box for a base64 (standard or URL, padded or not)
// encoded 32-byte key.
func FromBase64(encoded string) (*Box, error) {
	key, err := decodeKey(strings.TrimSpace(encoded))
	if err != nil {
		return nil, ErrInvalidKey
	}
	return New(key)
}

// ValidKey reports whether encoded is a usable key.
func ValidKey(encoded string) bool {
	_, err := FromBase64(encoded)
	return err == nil
}

// Random returns a Box with a fresh random key (development fallback: what
// it seals is unreadable after a restart).
func Random() (*Box, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	return New(key)
}

func decodeKey(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil && len(b) == KeySize {
			return b, nil
		}
	}
	return nil, ErrInvalidKey
}

// Seal encrypts plaintext bound to associated data ad.
func (b *Box) Seal(plaintext, ad string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secretbox: %w", err)
	}
	out := b.aead.Seal(nonce, nonce, []byte(plaintext), []byte(ad))
	return prefix + base64.StdEncoding.EncodeToString(out), nil
}

// Open decrypts a value sealed with the same key and associated data.
func (b *Box) Open(sealed, ad string) (string, error) {
	if !strings.HasPrefix(sealed, prefix) {
		return "", ErrDecrypt
	}
	raw, err := base64.StdEncoding.DecodeString(sealed[len(prefix):])
	if err != nil || len(raw) < b.aead.NonceSize()+b.aead.Overhead() {
		return "", ErrDecrypt
	}
	n := b.aead.NonceSize()
	plain, err := b.aead.Open(nil, raw[:n], raw[n:], []byte(ad))
	if err != nil {
		return "", ErrDecrypt
	}
	return string(plain), nil
}

// MAC returns HMAC-SHA256(purpose-derived key, msg). Each purpose gets its
// own derived key, so a MAC made for one purpose (e.g. OAuth state) is never
// valid for another.
func (b *Box) MAC(purpose string, msg []byte) []byte {
	derived := hmac.New(sha256.New, b.key)
	derived.Write([]byte("secretbox-mac:" + purpose))
	m := hmac.New(sha256.New, derived.Sum(nil))
	m.Write(msg)
	return m.Sum(nil)
}
