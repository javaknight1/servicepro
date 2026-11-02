package email

import (
	"context"
	"testing"
)

func TestResolveFrom(t *testing.T) {
	ctx := context.Background()
	msg := NewEmailMessage("to@example.test", "s", "b")

	from, reply := ResolveFrom(ctx, msg, "ServicePro", "noreply@sp.test", "noreply@sp.test")
	if from != "ServicePro <noreply@sp.test>" || reply != "noreply@sp.test" {
		t.Fatalf("platform default: %q %q", from, reply)
	}

	tctx := WithSender(ctx, Sender{BusinessName: "Brightline <Facility>\r\nBcc: x", ReplyTo: "office@brightline.test"})
	from, reply = ResolveFrom(tctx, msg, "ServicePro", "noreply@sp.test", "noreply@sp.test")
	if from != "Brightline FacilityBcc: x via ServicePro <noreply@sp.test>" {
		t.Fatalf("tenant sender (sanitised): %q", from)
	}
	if reply != "office@brightline.test" {
		t.Fatalf("replies go to the business: %q", reply)
	}
	if BusinessName(tctx) != "Brightline FacilityBcc: x" || BusinessName(ctx) != "ServicePro" {
		t.Fatalf("BusinessName: %q / %q", BusinessName(tctx), BusinessName(ctx))
	}

	msg.ReplyTo = "explicit@example.test"
	if _, reply = ResolveFrom(tctx, msg, "", "noreply@sp.test", ""); reply != "explicit@example.test" {
		t.Fatalf("explicit reply-to wins: %q", reply)
	}
}
