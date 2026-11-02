package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	emailmock "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
)

// T265: automatic reminders only go out 8 a.m.–6 p.m. tenant-local.
func TestInReminderWindow(t *testing.T) {
	honolulu := &models.Tenant{Settings: models.TenantSettings{Timezone: "Pacific/Honolulu"}}
	ny := &models.Tenant{Settings: models.TenantSettings{Timezone: "America/New_York"}}
	at := func(s string) time.Time { tm, _ := time.Parse(time.RFC3339, s); return tm }

	// 15:00 UTC = 05:00 Honolulu (too early), 10:00/11:00 New York (fine).
	assert.False(t, inReminderWindow(honolulu, at("2026-11-02T15:00:00Z")))
	assert.True(t, inReminderWindow(ny, at("2026-11-02T15:00:00Z")))
	// 23:30 UTC = 13:30 Honolulu (fine), 18:30 New York (too late).
	assert.True(t, inReminderWindow(honolulu, at("2026-11-02T23:30:00Z")))
	assert.False(t, inReminderWindow(ny, at("2026-11-02T23:30:00Z")))
	// No timezone: UTC.
	assert.True(t, inReminderWindow(&models.Tenant{}, at("2026-11-02T12:00:00Z")))
}

// T265: invoice emails honour the customer's opt-out.
func TestSendInvoiceEmail_HonoursDoNotEmail(t *testing.T) {
	mc := emailmock.NewClient(&emailmock.Config{FromEmail: "noreply@example.test", FromName: "ServicePro"})
	svc := &InvoiceService{emailClient: mc}
	tok := "tok"
	inv := &models.Invoice{InvoiceNumber: "INV-1", PaymentToken: &tok,
		Customer: &models.Customer{Email: "pm@example.test", DoNotEmail: true}}

	svc.sendInvoiceEmail(inv)
	assert.Empty(t, mc.SentEmails, "opted-out customer gets no invoice email")

	inv.Customer.DoNotEmail = false
	svc.sendInvoiceEmail(inv)
	assert.Len(t, mc.SentEmails, 1)
}

// T265: customer emails carry the business name, not "ServicePro".
func TestTenantSender_BrandsCustomerEmails(t *testing.T) {
	mc := emailmock.NewClient(&emailmock.Config{FromEmail: "noreply@example.test", FromName: "ServicePro"})
	ctx := tenantSenderFor(context.Background(), &models.Tenant{Name: "Brightline Facility Services", Email: "office@brightline.test"})

	require.NoError(t, mc.SendInvoiceEmail(ctx, "pm@example.test", &models.Invoice{InvoiceNumber: "INV-7"}, "", nil, ""))
	require.Len(t, mc.SentEmails, 1)
	assert.Equal(t, "Invoice INV-7 from Brightline Facility Services", mc.SentEmails[0].Subject)

	from, reply := email.ResolveFrom(ctx, mc.SentEmails[0], "ServicePro", "noreply@example.test", "")
	assert.True(t, strings.HasPrefix(from, "Brightline Facility Services via ServicePro <"), from)
	assert.Equal(t, "office@brightline.test", reply)
}
