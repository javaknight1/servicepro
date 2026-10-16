package models

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// A gorm default:true tag on a bool makes GORM omit an explicit false from
// the INSERT, so the database default (true) wins: a non-taxable invoice line
// was saved as taxable (audit 7.M3). These fields must carry no GORM default.
func TestBoolFieldsHaveNoTrueDefault(t *testing.T) {
	cases := []struct {
		model any
		field string
	}{
		{&InvoiceLine{}, "Taxable"},
		{&TaxRate{}, "IsActive"},
		{&PaymentTerm{}, "IsActive"},
		{&Schedule{}, "RemindersEnabled"},
		{&TenantUser{}, "IsActive"},
		{&JobMaterial{}, "Billable"},
	}
	for _, c := range cases {
		s, err := schema.Parse(c.model, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Fatal(err)
		}
		f := s.LookUpField(c.field)
		if f == nil {
			t.Fatalf("%T has no field %s", c.model, c.field)
		}
		if f.HasDefaultValue || f.DefaultValue != "" {
			t.Errorf("%s.%s has GORM default %q; an explicit false would be stored as true", s.Name, c.field, f.DefaultValue)
		}
	}
}
