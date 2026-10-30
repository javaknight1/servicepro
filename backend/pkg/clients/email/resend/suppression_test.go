package resend

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
)

type staticSuppressor map[string]bool

func (s staticSuppressor) IsSuppressed(_ context.Context, addr string) (bool, error) {
	return s[strings.ToLower(addr)], nil
}

// T159: bounced/complained addresses are never sent to.
func TestSend_SkipsSuppressedRecipients(t *testing.T) {
	f := &fakeResend{}
	srv := httptest.NewServer(f)
	defer srv.Close()
	c, _ := newTestClient(t, srv)
	c.SetSuppressor(staticSuppressor{"bounced@example.test": true})

	_, err := c.Send(context.Background(), email.NewEmailMessage("Bounced@example.test", "Hi", "<p>Hi</p>"))
	require.True(t, errors.Is(err, email.ErrAllRecipientsSuppressed))
	assert.Empty(t, f.keys, "nothing reaches Resend")

	msg := email.NewEmailMessage("ok@example.test", "Hi", "<p>Hi</p>")
	msg.To = append(msg.To, "bounced@example.test")
	_, err = c.Send(context.Background(), msg)
	require.NoError(t, err)
	require.Len(t, f.bodies, 1)
	assert.Contains(t, f.bodies[0], "ok@example.test")
	assert.NotContains(t, f.bodies[0], "bounced@example.test")
}
