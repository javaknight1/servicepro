package models

import (
	"net/mail"
	"strings"

	"github.com/shopspring/decimal"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
)

// optionalText trims v and turns "" into nil (clearing the field).
func optionalText(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// CleanPONumber trims a PO number from a request; nil, "" and blanks mean
// "no PO" (nil).
func CleanPONumber(v *string) *string {
	if v == nil {
		return nil
	}
	return optionalText(*v)
}

// Apply copies the fields present in r onto b, validating them. It returns a
// 400 validation error naming the bad fields.
func (r *BillingRulesRequest) Apply(b *BillingRules) error {
	fields := map[string]string{}
	if r.PORequired != nil {
		b.PORequired = *r.PORequired
	}
	if r.DefaultPONumber != nil {
		b.DefaultPONumber = optionalText(*r.DefaultPONumber)
	}
	if r.BillingApproverName != nil {
		b.BillingApproverName = optionalText(*r.BillingApproverName)
	}
	if r.BillingApproverEmail != nil {
		email := optionalText(*r.BillingApproverEmail)
		if email != nil {
			if addr, err := mail.ParseAddress(*email); err != nil || addr.Address != *email {
				fields["billing_approver_email"] = "must be a valid email address"
			}
		}
		b.BillingApproverEmail = email
	}
	switch {
	case r.ClearNTELimit:
		b.NTELimit = nil
	case r.NTELimit != nil && r.NTELimit.IsNegative():
		fields["nte_limit"] = "must be zero or more"
	case r.NTELimit != nil && r.NTELimit.GreaterThanOrEqual(decimal.New(1, 10)):
		fields["nte_limit"] = "is too large"
	case r.NTELimit != nil:
		v := r.NTELimit.Round(2)
		b.NTELimit = &v
	}
	if r.ConsolidatedBilling != nil {
		b.ConsolidatedBilling = *r.ConsolidatedBilling
	}
	if len(fields) > 0 {
		return apperrors.Validation("Invalid billing rules", fields)
	}
	return nil
}
