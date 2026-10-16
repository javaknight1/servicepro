package services

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T229: invitation links must survive any email address. Before, the email
// was interpolated raw, so "pat+ops@example.com" reached the register page as
// "pat ops@example.com" ("+" decodes to a space in a query string).
func TestInvitationLinks_EncodeQueryValues(t *testing.T) {
	svc := NewInvitationService(nil, nil, "https://app.example.com/")

	email := "pat+ops&x=1@example.com"
	token := "tok/with+chars"

	reg, err := url.Parse(svc.inviteRegisterURL(token, email))
	require.NoError(t, err)
	assert.Equal(t, "https", reg.Scheme)
	assert.Equal(t, "app.example.com", reg.Host)
	assert.Equal(t, "/register", reg.Path)
	assert.Equal(t, email, reg.Query().Get("email"))
	assert.Equal(t, token, reg.Query().Get("invitation"))
	assert.Len(t, reg.Query(), 2, "no stray parameters from the email")

	acc, err := url.Parse(svc.inviteAcceptURL(token))
	require.NoError(t, err)
	assert.Equal(t, "/invitations/accept", acc.Path)
	assert.Equal(t, token, acc.Query().Get("token"))
}
