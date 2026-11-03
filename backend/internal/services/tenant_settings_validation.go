package services

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Limits on tenant settings values (T235).
const (
	maxSettingsPrefixLen   = 20 // models.CounterPrefix falls back above this
	maxSettingsCityLen     = 100
	maxSettingsStateLen    = 50
	maxSettingsTechnicians = 10000
	maxOnboardingStep      = 3
)

// settingsLookup answers whether a payment term / tax rate ID is one the
// tenant may use (its own or a global row, active). TenantRepository
// implements it.
type settingsLookup interface {
	UsablePaymentTerm(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
	UsableTaxRate(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
}

// ErrInvalidTenantSettings is the 400 for a settings payload with bad values;
// Fields names each bad setting by its JSON name.
var ErrInvalidTenantSettings = apperrors.Validation("Some settings are not valid", nil)

// validateTenantSettings checks a settings payload before it is stored and
// normalises it in place (trimmed text, upper-case currency). Only values
// that differ from prev (what is stored now) are checked, so a value saved
// before validation existed doesn't block saving something else; those are
// still cleaned up on read (models.CounterPrefix, NormalizeCurrency, ...).
//
// tenantID is the tenant being updated (from the authenticated path
// membership, never the body). lookup may be nil when the tenant does not
// exist yet (create): default term/tax IDs are then rejected.
func validateTenantSettings(ctx context.Context, lookup settingsLookup, tenantID uuid.UUID, prev, next *models.TenantSettings) error {
	fields := map[string]string{}

	next.Currency = strings.ToUpper(strings.TrimSpace(next.Currency))
	if next.Currency != "" && next.Currency != strings.ToUpper(strings.TrimSpace(prev.Currency)) && !models.IsISO4217Currency(next.Currency) {
		fields["currency"] = "must be an ISO 4217 currency code, like USD"
	}

	for _, p := range []struct {
		key       string
		val, prev *string
	}{
		{"invoicePrefix", &next.InvoicePrefix, &prev.InvoicePrefix},
		{"quotePrefix", &next.QuotePrefix, &prev.QuotePrefix},
		{"jobPrefix", &next.JobPrefix, &prev.JobPrefix},
	} {
		*p.val = strings.TrimSpace(*p.val)
		if *p.val == strings.TrimSpace(*p.prev) {
			continue
		}
		if msg := checkText(*p.val, maxSettingsPrefixLen); msg != "" {
			fields[p.key] = msg
		}
	}

	next.Timezone = strings.TrimSpace(next.Timezone)
	if next.Timezone != "" && next.Timezone != strings.TrimSpace(prev.Timezone) && !validTimezone(next.Timezone) {
		fields["timezone"] = "must be an IANA time zone, like America/New_York"
	}

	next.BusinessCity = strings.TrimSpace(next.BusinessCity)
	if next.BusinessCity != strings.TrimSpace(prev.BusinessCity) {
		if msg := checkText(next.BusinessCity, maxSettingsCityLen); msg != "" {
			fields["businessCity"] = msg
		}
	}
	next.BusinessState = strings.TrimSpace(next.BusinessState)
	if next.BusinessState != strings.TrimSpace(prev.BusinessState) {
		if msg := checkText(next.BusinessState, maxSettingsStateLen); msg != "" {
			fields["businessState"] = msg
		}
	}

	switch next.Trade {
	case "", models.TenantTradeCommercialCleaning, models.TenantTradeFacilityMaintenance, models.TenantTradeOther:
	default:
		if next.Trade != prev.Trade {
			fields["trade"] = "must be commercial_cleaning, facility_maintenance or other"
		}
	}
	if next.TechnicianCount < 0 || next.TechnicianCount > maxSettingsTechnicians {
		fields["technicianCount"] = "must be between 0 and 10000"
	}
	if next.OnboardingStep < 0 || next.OnboardingStep > maxOnboardingStep {
		fields["onboardingStep"] = "must be between 0 and 3"
	}

	canLookUp := lookup != nil && tenantID != uuid.Nil
	if err := checkDefaultRow(canLookUp, "defaultPaymentTermId", next.DefaultPaymentTermID, prev.DefaultPaymentTermID,
		func(id uuid.UUID) (bool, error) { return lookup.UsablePaymentTerm(ctx, tenantID, id) }, fields); err != nil {
		return err
	}
	if err := checkDefaultRow(canLookUp, "defaultTaxRateId", next.DefaultTaxRateID, prev.DefaultTaxRateID,
		func(id uuid.UUID) (bool, error) { return lookup.UsableTaxRate(ctx, tenantID, id) }, fields); err != nil {
		return err
	}

	if len(fields) > 0 {
		e := ErrInvalidTenantSettings.WithMessage(ErrInvalidTenantSettings.Message)
		e.Fields = fields
		return e
	}
	return nil
}

// checkDefaultRow validates a default payment term / tax rate ID: unchanged
// or cleared is fine; a new one must be usable by the tenant. Another
// tenant's ID gets the same answer as one that doesn't exist.
func checkDefaultRow(canLookUp bool, key string, next, prev *uuid.UUID,
	usable func(uuid.UUID) (bool, error), fields map[string]string) error {
	if next == nil || *next == uuid.Nil {
		return nil
	}
	if prev != nil && *prev == *next {
		return nil
	}
	if !canLookUp {
		fields[key] = "can be set once the organization exists"
		return nil
	}
	ok, err := usable(*next)
	if err != nil {
		return err
	}
	if !ok {
		fields[key] = "was not found"
	}
	return nil
}

// checkText returns a problem with a free-text setting, or "".
func checkText(v string, maxLen int) string {
	if utf8.RuneCountInString(v) > maxLen {
		return "must be at most " + strconv.Itoa(maxLen) + " characters"
	}
	if strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return "must not contain control characters"
	}
	return ""
}

// validTimezone reports whether name is an IANA zone the server can load.
// "Local" (the server's own zone) is not a tenant setting.
func validTimezone(name string) bool {
	if name == "Local" || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}
