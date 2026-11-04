package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// TemplateTypeProofOfService is the service report emailed to the customer
// after a signed visit (T176, mockup 09): what was done, where and when,
// materials used (never prices), photos and the customer's signature.
const TemplateTypeProofOfService TemplateType = "proof_of_service"

// ProofOfServiceData is the report's content. Times arrive formatted in the
// tenant's time zone; the template only lays them out.
type ProofOfServiceData struct {
	CompanyName    string
	CompanyContact string // "phone · email"
	JobNumber      string
	Title          string
	CustomerName   string
	SiteName       string
	SiteAddress    string
	PONumber       string
	VisitDate      string // "Mon, Nov 2, 2026"
	Scheduled      string // "5:30 PM - 7:00 PM"
	OnSite         string // "5:32 PM - 6:58 PM (1 h 26 min)"
	Technicians    string
	Description    string
	CompletionNote string
	Notes          []ProofOfServiceNote
	Materials      []ProofOfServiceMaterial
	Photos         []ProofOfServicePhoto
	// MorePhotos counts photos left out of the report.
	MorePhotos  int
	Signature   *ProofOfServiceSignature
	GeneratedAt string
}

// ProofOfServiceNote is a customer-visible job note.
type ProofOfServiceNote struct {
	Author string
	At     string
	Text   string
}

// ProofOfServiceMaterial is a material used: name and quantity, no price.
type ProofOfServiceMaterial struct {
	Name     string
	Quantity string
}

// ProofOfServicePhoto is a JPEG (a stored thumbnail) with its caption.
type ProofOfServicePhoto struct {
	JPEG    []byte
	Caption string
	Taken   string
}

// ProofOfServiceSignature is the customer's sign-off.
type ProofOfServiceSignature struct {
	PNG      []byte
	Name     string
	Title    string
	SignedAt string
	Device   string
}

// GetType returns the template type.
func (d *ProofOfServiceData) GetType() TemplateType { return TemplateTypeProofOfService }

// Validate checks the report has a job.
func (d *ProofOfServiceData) Validate() error {
	if d.JobNumber == "" {
		return fmt.Errorf("%w: job number required", ErrInvalidContent)
	}
	if d.CompanyName == "" {
		return fmt.Errorf("%w: company name required", ErrInvalidContent)
	}
	return nil
}

// Report colours (mockup palette): orange primary, neutral text, green done.
var (
	posOrange = [3]int{234, 88, 12}
	posText   = [3]int{23, 23, 23}
	posMuted  = [3]int{115, 115, 115}
	posBorder = [3]int{229, 229, 229}
	posTint   = [3]int{255, 247, 237}
)

type posWriter struct {
	pdf    *gofpdf.Fpdf
	tr     func(string) string
	family string
	left   float64
	width  float64
	bottom float64
	imgN   int
}

func (w *posWriter) color(c [3]int) { w.pdf.SetTextColor(c[0], c[1], c[2]) }

// ensure starts a new page when h more millimetres don't fit.
func (w *posWriter) ensure(h float64) {
	if w.pdf.GetY()+h > w.bottom {
		w.pdf.AddPage()
	}
}

// heading starts a section, on a new page unless keep more millimetres
// (the heading plus what must stay with it) fit.
func (w *posWriter) heading(text string, keep float64) {
	w.ensure(16 + keep)
	w.pdf.Ln(4)
	w.pdf.SetFont(w.family, "B", 12)
	w.color(posText)
	w.pdf.CellFormat(w.width, 7, w.tr(text), "", 1, "L", false, 0, "")
	w.pdf.SetDrawColor(posBorder[0], posBorder[1], posBorder[2])
	y := w.pdf.GetY()
	w.pdf.Line(w.left, y, w.left+w.width, y)
	w.pdf.Ln(2)
}

func (w *posWriter) para(text string, style string, size float64, c [3]int) {
	w.pdf.SetFont(w.family, style, size)
	w.color(c)
	w.pdf.MultiCell(w.width, 5, w.tr(text), "", "L", false)
}

// field draws "Label  value" rows in two columns of the summary box.
func (w *posWriter) field(x, y, colW float64, label, value string) {
	w.pdf.SetXY(x, y)
	w.pdf.SetFont(w.family, "", 8)
	w.color(posMuted)
	w.pdf.CellFormat(colW, 4, w.tr(strings.ToUpper(label)), "", 2, "L", false, 0, "")
	w.pdf.SetFont(w.family, "B", 10)
	w.color(posText)
	w.pdf.SetX(x)
	w.pdf.MultiCell(colW, 5, w.tr(value), "", "L", false)
}

// image registers data and returns its name and natural size, or "" when
// it can't be read (a bad photo never fails the whole report).
func (w *posWriter) image(data []byte, kind string) (string, float64, float64) {
	w.imgN++
	name := fmt.Sprintf("pos-img-%d", w.imgN)
	info := w.pdf.RegisterImageOptionsReader(name, gofpdf.ImageOptions{ImageType: kind}, bytes.NewReader(data))
	if w.pdf.Err() || info == nil {
		w.pdf.ClearError()
		return "", 0, 0
	}
	return name, info.Width(), info.Height()
}

// fit scales (iw, ih) into a box of bw x bh.
func fit(iw, ih, bw, bh float64) (float64, float64) {
	if iw <= 0 || ih <= 0 {
		return bw, bh
	}
	s := bw / iw
	if ih*s > bh {
		s = bh / ih
	}
	return iw * s, ih * s
}

func renderProofOfServiceTemplate(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error {
	d, ok := data.(*ProofOfServiceData)
	if !ok {
		return fmt.Errorf("%w: expected ProofOfServiceData", ErrInvalidContent)
	}
	left, _, right, _ := pdf.GetMargins()
	pageW, pageH := pdf.GetPageSize()
	_, bottomMargin := pdf.GetAutoPageBreak()
	w := &posWriter{
		pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor(""), family: config.Fonts.DefaultFamily,
		left: left, width: pageW - left - right, bottom: pageH - bottomMargin,
	}
	pdf.SetTitle("Service report "+d.JobNumber, true)
	// Footer on every page: the generator sets its footer only after the
	// template has run, which reaches the last page alone.
	footer := config.Branding.FooterText
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont(w.family, "", 8)
		w.color(posMuted)
		pdf.CellFormat(0, 10, w.tr(footer), "", 0, "C", false, 0, "")
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	pdf.SetAuthor(d.CompanyName, true)

	// Header: company left, "Service report" + job number right.
	pdf.SetXY(left, 18)
	pdf.SetFont(w.family, "B", 16)
	w.color(posOrange)
	pdf.CellFormat(w.width/2, 8, w.tr(d.CompanyName), "", 0, "L", false, 0, "")
	pdf.SetFont(w.family, "B", 16)
	w.color(posText)
	pdf.CellFormat(w.width/2, 8, "Service report", "", 1, "R", false, 0, "")
	pdf.SetFont(w.family, "", 9)
	w.color(posMuted)
	pdf.CellFormat(w.width/2, 5, w.tr(d.CompanyContact), "", 0, "L", false, 0, "")
	pdf.CellFormat(w.width/2, 5, w.tr(d.JobNumber), "", 1, "R", false, 0, "")
	pdf.Ln(5)

	// Title and where.
	pdf.SetFont(w.family, "B", 15)
	w.color(posText)
	pdf.MultiCell(w.width, 7, w.tr(d.Title), "", "L", false)
	where := []string{}
	for _, p := range []string{d.CustomerName, d.SiteName, d.SiteAddress} {
		if strings.TrimSpace(p) != "" {
			where = append(where, p)
		}
	}
	if len(where) > 0 {
		w.para(strings.Join(where, " · "), "", 10, posMuted)
	}
	pdf.Ln(3)

	// Summary box: visit, on site, technician, PO.
	boxY := pdf.GetY()
	colW := w.width/2 - 6
	pdf.SetFillColor(posTint[0], posTint[1], posTint[2])
	pdf.SetDrawColor(posBorder[0], posBorder[1], posBorder[2])
	pdf.Rect(left, boxY, w.width, 30, "FD")
	visit := d.VisitDate
	if d.Scheduled != "" {
		visit += " · " + d.Scheduled + " scheduled"
	}
	onSite := d.OnSite
	if onSite == "" {
		onSite = "Not recorded"
	}
	w.field(left+4, boxY+3, colW, "Visit", visit)
	w.field(left+w.width/2+2, boxY+3, colW, "On site", onSite)
	tech := d.Technicians
	if tech == "" {
		tech = "-"
	}
	w.field(left+4, boxY+16, colW, "Technician", tech)
	if d.PONumber != "" {
		w.field(left+w.width/2+2, boxY+16, colW, "PO number", d.PONumber)
	}
	pdf.SetXY(left, boxY+32)

	if strings.TrimSpace(d.Description) != "" || strings.TrimSpace(d.CompletionNote) != "" {
		w.heading("Work done", 10)
		if strings.TrimSpace(d.Description) != "" {
			w.para(d.Description, "", 10, posText)
		}
		if strings.TrimSpace(d.CompletionNote) != "" {
			pdf.Ln(1)
			w.para("Completion notes: "+d.CompletionNote, "", 10, posText)
		}
	}

	if len(d.Notes) > 0 {
		w.heading("Technician notes", 10)
		for _, n := range d.Notes {
			w.ensure(12)
			w.para(n.Text, "", 10, posText)
			meta := strings.Trim(strings.Join([]string{n.Author, n.At}, " · "), " ·")
			if meta != "" {
				w.para(meta, "", 8, posMuted)
			}
			pdf.Ln(1.5)
		}
	}

	if len(d.Materials) > 0 {
		w.heading("Materials used", 6)
		pdf.SetFont(w.family, "", 10)
		for _, m := range d.Materials {
			w.ensure(6)
			w.color(posText)
			pdf.CellFormat(w.width-40, 6, fitText(pdf, w.tr(m.Name), w.width-40), "B", 0, "L", false, 0, "")
			w.color(posMuted)
			pdf.CellFormat(40, 6, w.tr(m.Quantity), "B", 1, "R", false, 0, "")
		}
	}

	if len(d.Photos) > 0 {
		label := fmt.Sprintf("Photos (%d)", len(d.Photos)+d.MorePhotos)
		w.heading(label, 50)
		const cols = 3
		gap := 4.0
		cellW := (w.width - gap*(cols-1)) / cols
		cellH := cellW * 0.75
		for i, p := range d.Photos {
			col := i % cols
			if col == 0 {
				w.ensure(cellH + 12)
			}
			x := left + float64(col)*(cellW+gap)
			y := pdf.GetY()
			if name, iw, ih := w.image(p.JPEG, "JPG"); name != "" {
				fw, fh := fit(iw, ih, cellW, cellH)
				pdf.ImageOptions(name, x+(cellW-fw)/2, y+(cellH-fh)/2, fw, fh, false, gofpdf.ImageOptions{}, 0, "")
			} else {
				pdf.SetDrawColor(posBorder[0], posBorder[1], posBorder[2])
				pdf.Rect(x, y, cellW, cellH, "D")
			}
			caption := strings.Trim(strings.Join([]string{p.Caption, p.Taken}, " · "), " ·")
			pdf.SetXY(x, y+cellH+1)
			pdf.SetFont(w.family, "", 8)
			w.color(posMuted)
			pdf.CellFormat(cellW, 4, fitText(pdf, w.tr(caption), cellW), "", 0, "L", false, 0, "")
			if col == cols-1 || i == len(d.Photos)-1 {
				pdf.SetXY(left, y+cellH+7)
			} else {
				pdf.SetY(y)
			}
		}
		if d.MorePhotos > 0 {
			w.para(fmt.Sprintf("%d more photos are on the job.", d.MorePhotos), "I", 8, posMuted)
		}
	}

	w.heading("Customer sign-off", 36)
	if s := d.Signature; s != nil {
		y := pdf.GetY()
		boxW, boxH := 80.0, 30.0
		pdf.SetDrawColor(posBorder[0], posBorder[1], posBorder[2])
		pdf.SetDashPattern([]float64{1, 1}, 0)
		pdf.Rect(left, y, boxW, boxH, "D")
		pdf.SetDashPattern([]float64{}, 0)
		if name, iw, ih := w.image(s.PNG, "PNG"); name != "" {
			fw, fh := fit(iw, ih, boxW-6, boxH-6)
			pdf.ImageOptions(name, left+(boxW-fw)/2, y+(boxH-fh)/2, fw, fh, false, gofpdf.ImageOptions{}, 0, "")
		}
		x := left + boxW + 8
		textW := w.width - boxW - 8
		pdf.SetXY(x, y+4)
		signer := s.Name
		if s.Title != "" {
			signer += ", " + s.Title
		}
		pdf.SetFont(w.family, "B", 11)
		w.color(posText)
		pdf.MultiCell(textW, 6, w.tr(signer), "", "L", false)
		pdf.SetX(x)
		pdf.SetFont(w.family, "", 9)
		w.color(posMuted)
		pdf.MultiCell(textW, 5, w.tr("Signed "+s.SignedAt), "", "L", false)
		if s.Device != "" {
			pdf.SetX(x)
			pdf.MultiCell(textW, 5, w.tr(s.Device), "", "L", false)
		}
		pdf.SetXY(left, y+boxH+3)
	} else {
		w.para("Not signed.", "I", 10, posMuted)
	}

	pdf.Ln(4)
	w.para("Report generated "+d.GeneratedAt+".", "", 8, posMuted)
	return nil
}
