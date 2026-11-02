package email

import (
	"context"
	"fmt"
	"strings"
)

// PlatformName is the product name shown after "via" and on platform
// emails (sign-up, password reset, invitations).
const PlatformName = "ServicePro"

// Sender is the business an email is sent on behalf of (T265). Emails to a
// tenant's customers — quotes, invoices, receipts, reminders — come from
// "<Business> via ServicePro" at our verified address, with replies going
// to the business. That keeps them clearly the tenant's (FDCPA false-name
// exception, Resend's acceptable-use policy) while SPF/DKIM stay ours.
type Sender struct {
	BusinessName string
	ReplyTo      string
}

type senderKey struct{}

// WithSender attaches the sending business to ctx.
func WithSender(ctx context.Context, s Sender) context.Context {
	return context.WithValue(ctx, senderKey{}, s)
}

// SenderFrom returns the business attached to ctx, if any.
func SenderFrom(ctx context.Context) (Sender, bool) {
	s, ok := ctx.Value(senderKey{}).(Sender)
	if !ok || strings.TrimSpace(s.BusinessName) == "" {
		return Sender{}, false
	}
	return s, true
}

// BusinessName is the sending business's name, or the platform name.
func BusinessName(ctx context.Context) string {
	if s, ok := SenderFrom(ctx); ok {
		return cleanDisplayName(s.BusinessName)
	}
	return PlatformName
}

// ResolveFrom picks the From header and Reply-To for msg: an explicit
// msg.From wins; otherwise a Sender in ctx gives "<Business> via
// ServicePro <fromEmail>"; otherwise the client's default display name.
func ResolveFrom(ctx context.Context, msg *EmailMessage, defaultName, fromEmail, defaultReplyTo string) (from, replyTo string) {
	replyTo = defaultReplyTo
	switch {
	case msg.From != "":
		from = msg.From
		if msg.FromName != "" {
			from = fmt.Sprintf("%s <%s>", cleanDisplayName(msg.FromName), msg.From)
		}
	default:
		name := defaultName
		if s, ok := SenderFrom(ctx); ok {
			name = cleanDisplayName(s.BusinessName) + " via " + PlatformName
			if s.ReplyTo != "" {
				replyTo = s.ReplyTo
			}
		}
		from = fromEmail
		if name != "" {
			from = fmt.Sprintf("%s <%s>", cleanDisplayName(name), fromEmail)
		}
	}
	if msg.ReplyTo != "" {
		replyTo = msg.ReplyTo
	}
	return from, replyTo
}

// cleanDisplayName keeps a header display name safe: no angle brackets,
// quotes or line breaks (header injection), at most 70 characters.
func cleanDisplayName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', '"', '\r', '\n':
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	if len([]rune(name)) > 70 {
		name = string([]rune(name)[:70])
	}
	return name
}
