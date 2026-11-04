package secretbox

import (
	"crypto/hmac"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey() string {
	return base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
}

func TestSealOpenRoundTrip(t *testing.T) {
	b, err := FromBase64(testKey())
	require.NoError(t, err)

	sealed, err := b.Seal("refresh-token-value", "quickbooks:refresh:tenant-a")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(sealed, "v1:"))
	assert.NotContains(t, sealed, "refresh-token-value")

	got, err := b.Open(sealed, "quickbooks:refresh:tenant-a")
	require.NoError(t, err)
	assert.Equal(t, "refresh-token-value", got)

	again, err := b.Seal("refresh-token-value", "quickbooks:refresh:tenant-a")
	require.NoError(t, err)
	assert.NotEqual(t, sealed, again, "fresh nonce per seal")
}

func TestOpenRejectsOtherOwnerKeyOrTampering(t *testing.T) {
	b, err := FromBase64(testKey())
	require.NoError(t, err)
	sealed, err := b.Seal("secret", "quickbooks:access:tenant-a")
	require.NoError(t, err)

	_, err = b.Open(sealed, "quickbooks:access:tenant-b")
	assert.ErrorIs(t, err, ErrDecrypt, "a value copied to another tenant's row doesn't open")

	other, err := Random()
	require.NoError(t, err)
	_, err = other.Open(sealed, "quickbooks:access:tenant-a")
	assert.ErrorIs(t, err, ErrDecrypt)

	raw, _ := base64.StdEncoding.DecodeString(sealed[3:])
	raw[len(raw)-1] ^= 1
	_, err = b.Open("v1:"+base64.StdEncoding.EncodeToString(raw), "quickbooks:access:tenant-a")
	assert.ErrorIs(t, err, ErrDecrypt)

	_, err = b.Open("plaintext", "x")
	assert.ErrorIs(t, err, ErrDecrypt)
}

func TestKeyValidation(t *testing.T) {
	assert.True(t, ValidKey(testKey()))
	assert.True(t, ValidKey(strings.TrimRight(testKey(), "=")))
	assert.False(t, ValidKey(""))
	assert.False(t, ValidKey("short"))
	assert.False(t, ValidKey(base64.StdEncoding.EncodeToString([]byte("sixteen-bytes-ab"))))
}

func TestMACIsPurposeBound(t *testing.T) {
	b, err := FromBase64(testKey())
	require.NoError(t, err)
	m1 := b.MAC("oauth-state", []byte("msg"))
	assert.True(t, hmac.Equal(m1, b.MAC("oauth-state", []byte("msg"))))
	assert.False(t, hmac.Equal(m1, b.MAC("other", []byte("msg"))))
	assert.False(t, hmac.Equal(m1, b.MAC("oauth-state", []byte("msg2"))))
}
