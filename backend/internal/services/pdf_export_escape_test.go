package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func TestBuildPDFHTML_EscapesUserContent(t *testing.T) {
	s := &unifiedExportService{}
	payload := `<iframe src='file:///proc/self/environ'></iframe>`
	html := s.buildPDFHTML(
		[]models.Customer{{FirstName: `<img src=x onerror=alert(1)>`, LastName: "Doe", Email: "a@b.test"}},
		&models.ExportConfig{ExportType: models.ExportTypeCustomers},
		&models.PDFExportOptions{Title: "<script>x</script>", HeaderText: payload},
	)

	for _, raw := range []string{"<iframe", "<img", "<script>"} {
		assert.False(t, strings.Contains(html, raw), "unescaped %q in PDF HTML", raw)
	}
	assert.Contains(t, html, "&lt;iframe")
	assert.Contains(t, html, "a@b.test")
	assert.Contains(t, html, "Total Customers:</strong> 1")
}

func TestWkhtmltopdfArgs_NoLocalFileAccess(t *testing.T) {
	args := (&PDFService{}).buildWkhtmltopdfArgs("in.html", "out.pdf", &PDFGenerationOptions{})
	joined := strings.Join(args, " ")
	assert.NotContains(t, joined, "--enable-local-file-access")
	assert.Contains(t, joined, "--disable-local-file-access")
	assert.Contains(t, joined, "--disable-javascript")
}
