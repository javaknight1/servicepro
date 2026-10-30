package pdf

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// TemplateTypeTableReport renders a titled table: the list exports
// (customers, jobs, invoices, quotes). It replaced the wkhtmltopdf export
// engine (T260, D13), so every PDF comes from gofpdf.
const TemplateTypeTableReport TemplateType = "table_report"

// TableReportData is a titled table of plain-text cells.
type TableReportData struct {
	Title       string     `json:"title"`
	Subtitle    string     `json:"subtitle,omitempty"`
	Summary     string     `json:"summary,omitempty"`
	GeneratedAt time.Time  `json:"generated_at"`
	Headers     []string   `json:"headers"`
	Rows        [][]string `json:"rows"`
}

// GetType returns the template type.
func (d *TableReportData) GetType() TemplateType {
	return TemplateTypeTableReport
}

// Validate checks the table is well formed.
func (d *TableReportData) Validate() error {
	if d.Title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidContent)
	}
	if len(d.Headers) == 0 {
		return fmt.Errorf("%w: at least one column is required", ErrInvalidContent)
	}
	for i, row := range d.Rows {
		if len(row) != len(d.Headers) {
			return fmt.Errorf("%w: row %d has %d cells, want %d", ErrInvalidContent, i+1, len(row), len(d.Headers))
		}
	}
	return nil
}

// renderTableReportTemplate draws the title block and the table, repeating
// the header row on every page and clipping cells to their column width.
func renderTableReportTemplate(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error {
	report, ok := data.(*TableReportData)
	if !ok {
		return fmt.Errorf("%w: expected TableReportData", ErrInvalidContent)
	}
	// Core fonts are cp1252; translate UTF-8 so accented names render.
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pr, pg, pb := hexToRGB(config.Branding.PrimaryColor)
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeHeading)
	pdf.SetTextColor(pr, pg, pb)
	pdf.MultiCell(0, 9, tr(report.Title), "", "L", false)

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
	if report.Subtitle != "" {
		pdf.MultiCell(0, 5, tr(report.Subtitle), "", "L", false)
	}
	meta := "Generated " + report.GeneratedAt.Format("January 2, 2006 3:04 PM")
	if report.Summary != "" {
		meta = report.Summary + "  |  " + meta
	}
	mr, mg, mb := hexToRGB(config.Branding.MutedColor)
	pdf.SetTextColor(mr, mg, mb)
	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
	pdf.MultiCell(0, 5, tr(meta), "", "L", false)
	pdf.Ln(4)

	left, _, right, _ := pdf.GetMargins()
	pageW, _ := pdf.GetPageSize()
	colW := (pageW - left - right) / float64(len(report.Headers))

	drawHeader := func() {
		pdf.SetFillColor(pr, pg, pb)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeSmall)
		for _, h := range report.Headers {
			pdf.CellFormat(colW, 7, fitText(pdf, tr(h), colW), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
	}
	drawHeader()

	const rowH = 6.0
	_, pageH := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	limit := pageH - bottom // bottom is the auto-page-break margin
	pdf.SetFillColor(249, 250, 251)
	for i, row := range report.Rows {
		if pdf.GetY()+rowH > limit {
			pdf.AddPage()
			drawHeader()
			pdf.SetFillColor(249, 250, 251)
		}
		fill := i%2 == 1
		for _, cell := range row {
			pdf.CellFormat(colW, rowH, fitText(pdf, tr(cell), colW), "1", 0, "L", fill, 0, "")
		}
		pdf.Ln(-1)
	}
	if len(report.Rows) == 0 {
		pdf.Ln(4)
		pdf.CellFormat(0, rowH, "No records.", "", 1, "L", false, 0, "")
	}
	return pdf.Error()
}

// fitText trims s with an ellipsis so it fits a cell of width w (with
// padding). s is already translated to the single-byte core-font encoding,
// so trimming by byte is safe.
func fitText(pdf *gofpdf.Fpdf, s string, w float64) string {
	maxW := w - 2*pdf.GetCellMargin()
	if pdf.GetStringWidth(s) <= maxW {
		return s
	}
	for len(s) > 0 && pdf.GetStringWidth(s+"...") > maxW {
		s = s[:len(s)-1]
	}
	return s + "..."
}
