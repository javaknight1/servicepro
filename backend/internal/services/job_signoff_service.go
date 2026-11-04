package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/image/draw"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

// Customer sign-off and the service report (T176). The customer signs on
// the technician's phone; the signature is stored as a PNG next to the
// job's photos and the service report PDF is built from the job (work done,
// notes, materials without prices, time on site, photos, signature) and
// emailed to the customer's contacts.

const (
	// MaxSignatureBytes caps the decoded signature PNG.
	MaxSignatureBytes = 512 << 10
	// MaxSignatureSide bounds each declared side of the signature image
	// (a phone canvas at devicePixelRatio 3 is about 1200 x 600).
	MaxSignatureSide = 3000
	// minSignatureWidth/Height refuse a canvas too small to sign on.
	minSignatureWidth, minSignatureHeight = 100, 40
	// minSignatureInk is how many dark pixels make a signature (an empty
	// canvas has none).
	minSignatureInk = 60
	// storedSignatureMaxWidth: wider signatures are scaled down.
	storedSignatureMaxWidth = 1200
	// maxReportPhotos bounds the photos placed in the report.
	maxReportPhotos = 12
	// reportEmailClaimTTL: a send claim older than this is presumed dead
	// (the instance crashed mid-send) and may be taken over.
	reportEmailClaimTTL = 10 * time.Minute
	signoffPrefix       = "job-signoffs"
	maxSignerNameLength = 100
)

// Statuses in which a visit can be signed off: started or done.
var signableStatuses = map[models.JobStatus]bool{
	models.JobStatusInProgress: true, models.JobStatusOnHold: true, models.JobStatusCompleted: true,
	models.JobStatusInvoiced: true, models.JobStatusPaid: true,
}

var (
	// ErrSignoffJobNotFound: the job isn't in the tenant, or the caller is
	// limited to assigned jobs and isn't assigned to it.
	ErrSignoffJobNotFound = apperrors.NotFound("job_not_found", "Job not found")
	// ErrSignoffNotFound: the job hasn't been signed off.
	ErrSignoffNotFound = apperrors.NotFound("signoff_not_found", "This job hasn't been signed off")
	// ErrSignoffNotStarted: only a started or finished visit can be signed.
	ErrSignoffNotStarted = apperrors.Conflict("job_not_started", "Start the visit before getting a signature")
	// ErrSignoffLocked: the report went out; the sign-off is the record.
	ErrSignoffLocked = apperrors.Conflict("signoff_locked", "The service report has been emailed, so this sign-off can't be replaced")
	// ErrSignatureType: the signature isn't a PNG.
	ErrSignatureType = apperrors.New(http.StatusUnsupportedMediaType, "unsupported_signature_type", "The signature must be a PNG image")
	// ErrSignatureTooLarge: over MaxSignatureBytes.
	ErrSignatureTooLarge = apperrors.New(http.StatusRequestEntityTooLarge, "signature_too_large", "The signature image is larger than 512 KB")
	// ErrSignatureDimensions: the PNG declares too many or too few pixels.
	ErrSignatureDimensions = apperrors.BadRequest("signature_dimensions", "The signature image must be between 100 x 40 and 3000 x 3000 pixels")
	// ErrSignatureUnreadable: not base64, or the PNG doesn't decode.
	ErrSignatureUnreadable = apperrors.BadRequest("invalid_signature", "The signature could not be read")
	// ErrSignatureEmpty: the canvas has (almost) no ink.
	ErrSignatureEmpty = apperrors.BadRequest("signature_empty", "Ask the customer to sign before submitting")
	// ErrSignerName: blank after trimming.
	ErrSignerName = apperrors.Validation("Enter the signer's name", map[string]string{"signer_name": "required"})
	// ErrReportNoRecipients: nobody to email the report to.
	ErrReportNoRecipients = apperrors.Conflict("no_recipients", "There is no email address for this customer or site (or the customer opted out of email)")
	// ErrReportEmailInProgress: another request is sending it right now.
	ErrReportEmailInProgress = apperrors.Conflict("email_in_progress", "The service report is being emailed")
	// ErrReportEmailUnavailable: no email provider configured.
	ErrReportEmailUnavailable = apperrors.New(http.StatusServiceUnavailable, "email_unavailable", "Email is not configured")
	// ErrReportEmailFailed: the provider refused the message.
	ErrReportEmailFailed = apperrors.New(http.StatusBadGateway, "email_failed", "The service report couldn't be emailed. Try again.")
)

// JobSignoffServiceInterface is the sign-off use case the handler calls.
// The tenant (and any technician row filter) always comes from ctx.
type JobSignoffServiceInterface interface {
	Get(ctx context.Context, jobID uuid.UUID) (*models.JobSignoffEnvelope, error)
	Capture(ctx context.Context, jobID, userID uuid.UUID, req *models.CaptureJobSignoffRequest, ip, userAgent string) (*models.JobSignoffResponse, error)
	SignatureLink(ctx context.Context, jobID uuid.UUID) (string, error)
	// ReportLink returns a presigned link to the service report: the
	// emailed one once sent, otherwise freshly generated from the job.
	ReportLink(ctx context.Context, jobID uuid.UUID) (string, error)
	EmailReport(ctx context.Context, jobID uuid.UUID, resend bool) (*models.EmailJobReportResponse, error)
}

// JobSignoffService stores sign-offs and builds and emails service reports.
type JobSignoffService struct {
	repo    repository.JobSignoffRepositoryInterface
	storage storage.Client
	pdf     *pdf.Generator
	email   email.Client
	now     func() time.Time
}

// NewJobSignoffService creates the service. emailClient may be nil (then
// emailing answers 503).
func NewJobSignoffService(repo repository.JobSignoffRepositoryInterface, storageClient storage.Client, generator *pdf.Generator, emailClient email.Client) *JobSignoffService {
	return &JobSignoffService{repo: repo, storage: storageClient, pdf: generator, email: emailClient, now: time.Now}
}

func (s *JobSignoffService) visibleJob(ctx context.Context, jobID uuid.UUID) (uuid.UUID, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	ok, err := s.repo.JobVisible(ctx, tenantID, jobID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("check job: %w", err)
	}
	if !ok {
		return uuid.Nil, ErrSignoffJobNotFound
	}
	return tenantID, nil
}

func (s *JobSignoffService) get(ctx context.Context, tenantID, jobID uuid.UUID) (*models.JobSignoff, error) {
	so, err := s.repo.Get(ctx, tenantID, jobID)
	if errors.Is(err, repository.ErrJobSignoffNotFound) {
		return nil, ErrSignoffNotFound
	}
	return so, err
}

// recipients are the customer's email (unless opted out), the site
// contact (likewise) and the signer's email (given at signing, so asked
// for), deduplicated case-insensitively.
func (s *JobSignoffService) recipients(ctx context.Context, tenantID, jobID uuid.UUID, so *models.JobSignoff) ([]string, error) {
	c, err := s.repo.Contacts(ctx, tenantID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobSignoffNotFound) {
			return nil, ErrSignoffJobNotFound
		}
		return nil, err
	}
	var candidates []string
	if !c.DoNotEmail {
		candidates = append(candidates, c.CustomerEmail, c.SiteEmail)
	}
	if so != nil && so.SignerEmail != nil {
		candidates = append(candidates, *so.SignerEmail)
	}
	seen := map[string]bool{}
	out := []string{}
	for _, addr := range candidates {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		parsed, err := mail.ParseAddress(addr)
		if err != nil || parsed.Address != addr {
			continue
		}
		key := strings.ToLower(addr)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, addr)
	}
	return out, nil
}

func (s *JobSignoffService) respond(ctx context.Context, tenantID uuid.UUID, so *models.JobSignoff) (*models.JobSignoffResponse, error) {
	r := so.ToResponse()
	url, err := s.storage.GetPresignedURL(ctx, so.SignatureKey, AttachmentLinkTTL)
	if err != nil {
		return nil, fmt.Errorf("presign signature: %w", err)
	}
	r.SignatureURL = url
	if r.SuggestedRecipients, err = s.recipients(ctx, tenantID, so.JobID, so); err != nil {
		return nil, err
	}
	return &r, nil
}

// Get returns the job's sign-off (nil when unsigned) and who the report
// would be emailed to.
func (s *JobSignoffService) Get(ctx context.Context, jobID uuid.UUID) (*models.JobSignoffEnvelope, error) {
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	so, err := s.get(ctx, tenantID, jobID)
	if err != nil && !errors.Is(err, ErrSignoffNotFound) {
		return nil, err
	}
	env := &models.JobSignoffEnvelope{}
	if so != nil {
		if env.Signoff, err = s.respond(ctx, tenantID, so); err != nil {
			return nil, err
		}
		env.SuggestedRecipients = env.Signoff.SuggestedRecipients
		return env, nil
	}
	if env.SuggestedRecipients, err = s.recipients(ctx, tenantID, jobID, nil); err != nil {
		return nil, err
	}
	return env, nil
}

// cleanText trims, drops control characters and caps at max runes.
func cleanText(v string, max int) string {
	v = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, v))
	if utf8.RuneCountInString(v) > max {
		v = string([]rune(v)[:max])
	}
	return v
}

func optionalText(v *string, max int) *string {
	if v == nil {
		return nil
	}
	c := cleanText(*v, max)
	if c == "" {
		return nil
	}
	return &c
}

// processedSignature is a signature ready to store.
type processedSignature struct {
	PNG           []byte
	Width, Height int
}

// decodeSignature checks and normalizes the signature: a PNG (sniffed from
// the bytes, never trusted from the data URL), at most MaxSignatureBytes,
// sides checked with image.DecodeConfig before decoding, with enough ink to
// be a signature. It is flattened onto white (no transparency to surprise a
// PDF viewer), scaled to at most storedSignatureMaxWidth and re-encoded,
// dropping any extra PNG chunks.
func decodeSignature(raw string) (*processedSignature, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "data:") {
		const prefix = "data:image/png;base64,"
		if !strings.HasPrefix(raw, prefix) {
			return nil, ErrSignatureType
		}
		raw = raw[len(prefix):]
	}
	if base64.StdEncoding.DecodedLen(len(raw)) > MaxSignatureBytes+3 {
		return nil, ErrSignatureTooLarge
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrSignatureUnreadable
	}
	if len(data) > MaxSignatureBytes {
		return nil, ErrSignatureTooLarge
	}
	if len(data) == 0 {
		return nil, ErrSignatureUnreadable
	}
	if http.DetectContentType(data) != contentTypePNG {
		return nil, ErrSignatureType
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrSignatureUnreadable
	}
	if cfg.Width < minSignatureWidth || cfg.Height < minSignatureHeight ||
		cfg.Width > MaxSignatureSide || cfg.Height > MaxSignatureSide {
		return nil, ErrSignatureDimensions
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrSignatureUnreadable
	}
	b := img.Bounds()
	flat := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(flat, flat.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, b.Min, draw.Over)

	ink := 0
	for i := 0; i+3 < len(flat.Pix) && ink < minSignatureInk; i += 4 {
		// Dark enough to be a pen stroke on white.
		if int(flat.Pix[i])+int(flat.Pix[i+1])+int(flat.Pix[i+2]) < 3*160 {
			ink++
		}
	}
	if ink < minSignatureInk {
		return nil, ErrSignatureEmpty
	}

	out := image.Image(flat)
	w, h := b.Dx(), b.Dy()
	if w > storedSignatureMaxWidth {
		h = max(1, h*storedSignatureMaxWidth/w)
		w = storedSignatureMaxWidth
		scaled := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(scaled, scaled.Bounds(), flat, flat.Bounds(), draw.Src, nil)
		out = scaled
	}
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, out); err != nil {
		return nil, fmt.Errorf("encode signature: %w", err)
	}
	return &processedSignature{PNG: buf.Bytes(), Width: w, Height: h}, nil
}

// Capture stores the customer's signature on the job. Signing again
// replaces the sign-off until its report has been emailed.
func (s *JobSignoffService) Capture(ctx context.Context, jobID, userID uuid.UUID, req *models.CaptureJobSignoffRequest, ip, userAgent string) (*models.JobSignoffResponse, error) {
	name := cleanText(req.SignerName, maxSignerNameLength)
	if name == "" {
		return nil, ErrSignerName
	}
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	status, err := s.repo.JobStatus(ctx, tenantID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobSignoffNotFound) {
			return nil, ErrSignoffJobNotFound
		}
		return nil, err
	}
	if !signableStatuses[status] {
		return nil, ErrSignoffNotStarted
	}
	sig, err := decodeSignature(req.Signature)
	if err != nil {
		return nil, err
	}

	var signerEmail *string
	if e := optionalText(req.SignerEmail, 255); e != nil {
		lower := strings.ToLower(*e)
		if parsed, err := mail.ParseAddress(lower); err != nil || parsed.Address != lower {
			return nil, apperrors.Validation("Enter a valid email", map[string]string{"signer_email": "invalid email"})
		}
		signerEmail = &lower
	}
	key := fmt.Sprintf("%s/%s/%s/%s_signature.png", signoffPrefix, tenantID, jobID, uuid.New())
	in := storage.NewUploadInput(key).WithContentType(contentTypePNG).WithCacheControl("private, max-age=3600")
	if _, err := s.storage.UploadBytes(ctx, sig.PNG, in); err != nil {
		return nil, fmt.Errorf("store signature: %w", err)
	}
	so := &models.JobSignoff{
		JobID: jobID, SignerName: name, SignerTitle: optionalText(req.SignerTitle, 100), SignerEmail: signerEmail,
		SignatureKey: key, SignatureWidth: sig.Width, SignatureHeight: sig.Height, SignedAt: s.now().UTC(),
		IPAddress: optionalText(&ip, 45), UserAgent: optionalText(&userAgent, 500), CapturedBy: &userID,
	}
	replaced, err := s.repo.Save(ctx, tenantID, so)
	if err != nil {
		if delErr := s.storage.Delete(ctx, key); delErr != nil {
			logging.Warn(ctx, "[JOB-SIGNOFF] Failed to remove orphaned signature", map[string]any{"error": delErr})
		}
		switch {
		case errors.Is(err, repository.ErrJobSignoffLocked):
			return nil, ErrSignoffLocked
		case errors.Is(err, repository.ErrJobSignoffNotFound):
			return nil, ErrSignoffJobNotFound
		}
		return nil, fmt.Errorf("save sign-off: %w", err)
	}
	if replaced != nil {
		for _, old := range []*string{&replaced.SignatureKey, replaced.ReportKey} {
			if old == nil || *old == "" {
				continue
			}
			if err := s.storage.Delete(ctx, *old); err != nil {
				logging.Warn(ctx, "[JOB-SIGNOFF] Failed to remove replaced object", map[string]any{"error": err})
			}
		}
	}
	logging.Info(ctx, "[JOB-SIGNOFF] Sign-off captured", map[string]any{
		"job_id": jobID.String(), "signoff_id": so.ID.String(), "replaced": replaced != nil,
	})
	saved, err := s.get(ctx, tenantID, jobID)
	if err != nil {
		return nil, err
	}
	return s.respond(ctx, tenantID, saved)
}

// SignatureLink returns a presigned link to the signature PNG.
func (s *JobSignoffService) SignatureLink(ctx context.Context, jobID uuid.UUID) (string, error) {
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return "", err
	}
	so, err := s.get(ctx, tenantID, jobID)
	if err != nil {
		return "", err
	}
	url, err := s.storage.GetPresignedURL(ctx, so.SignatureKey, AttachmentLinkTTL)
	if err != nil {
		return "", fmt.Errorf("presign signature: %w", err)
	}
	return url, nil
}

// ReportLink returns a presigned link to the service report: the stored
// copy once it has been emailed (the record the customer got), otherwise a
// fresh one built from the job as it is now.
func (s *JobSignoffService) ReportLink(ctx context.Context, jobID uuid.UUID) (string, error) {
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return "", err
	}
	so, err := s.get(ctx, tenantID, jobID)
	if err != nil {
		return "", err
	}
	key := ""
	if so.ReportEmailedAt != nil && so.ReportKey != nil {
		key = *so.ReportKey
	} else {
		if _, key, err = s.buildAndStore(ctx, tenantID, so); err != nil {
			return "", err
		}
	}
	url, err := s.storage.GetPresignedURL(ctx, key, AttachmentLinkTTL)
	if err != nil {
		return "", fmt.Errorf("presign report: %w", err)
	}
	return url, nil
}

func reportFilename(jobNumber string) string {
	return fmt.Sprintf("Service-Report-%s.pdf", strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return '-'
		}
		return r
	}, jobNumber))
}

// buildAndStore generates the report and stores it under the sign-off's
// report key (overwriting an earlier draft).
func (s *JobSignoffService) buildAndStore(ctx context.Context, tenantID uuid.UUID, so *models.JobSignoff) (*reportFile, string, error) {
	rep, err := s.build(ctx, tenantID, so)
	if err != nil {
		return nil, "", err
	}
	key := fmt.Sprintf("%s/%s/%s/%s_report.pdf", signoffPrefix, tenantID, so.JobID, so.ID)
	in := storage.NewUploadInput(key).WithContentType(contentTypePDF).
		WithCacheControl("private, max-age=60")
	if _, err := s.storage.UploadBytes(ctx, rep.Content, in); err != nil {
		return nil, "", fmt.Errorf("store report: %w", err)
	}
	if err := s.repo.SetReport(ctx, tenantID, so.JobID, key, s.now().UTC()); err != nil {
		return nil, "", fmt.Errorf("record report: %w", err)
	}
	return rep, key, nil
}

// reportFile is a generated report with what the email needs.
type reportFile struct {
	Content  []byte
	Filename string
	Source   *repository.JobReportSource
}

// build loads the job and renders the report.
func (s *JobSignoffService) build(ctx context.Context, tenantID uuid.UUID, so *models.JobSignoff) (*reportFile, error) {
	if s.pdf == nil {
		return nil, apperrors.New(http.StatusServiceUnavailable, "pdf_unavailable", "PDF generation is not configured")
	}
	src, err := s.repo.ReportSource(ctx, tenantID, so.JobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobSignoffNotFound) {
			return nil, ErrSignoffJobNotFound
		}
		return nil, fmt.Errorf("load report source: %w", err)
	}
	data := s.reportData(ctx, src, so)
	filename := reportFilename(src.Job.JobNumber)
	branding := s.pdf.GetConfig().Branding
	branding.CompanyName = data.CompanyName
	branding.FooterText = "Service report " + src.Job.JobNumber + " - " + data.CompanyName
	res, err := s.pdf.Generate(ctx, data, &pdf.GenerationOptions{
		Filename: filename, ReturnContent: true, SkipLocalStorage: true, SkipS3Upload: true, Branding: &branding,
	})
	if err != nil {
		return nil, fmt.Errorf("generate report: %w", err)
	}
	return &reportFile{Content: res.Content, Filename: filename, Source: src}, nil
}

func customerDisplayName(c *models.Customer) string {
	if c == nil {
		return ""
	}
	if c.CompanyName != nil && strings.TrimSpace(*c.CompanyName) != "" {
		return strings.TrimSpace(*c.CompanyName)
	}
	return strings.TrimSpace(c.FirstName + " " + c.LastName)
}

func formatDuration(d time.Duration) string {
	m := int(d.Round(time.Minute).Minutes())
	switch {
	case m < 60:
		return fmt.Sprintf("%d min", m)
	case m%60 == 0:
		return fmt.Sprintf("%d h", m/60)
	default:
		return fmt.Sprintf("%d h %d min", m/60, m%60)
	}
}

// onSite is when the crew was on site: the job's actual start and end, or
// failing those the first start / completion in the status history.
func onSite(src *repository.JobReportSource) (start, end *time.Time) {
	at := func(v int64) *time.Time { t := time.Unix(v, 0); return &t }
	if src.Job.ActualStartAt != nil {
		start = at(*src.Job.ActualStartAt)
	}
	if src.Job.ActualEndAt != nil {
		end = at(*src.Job.ActualEndAt)
	}
	for _, t := range src.Transitions {
		if start == nil && t.ToStatus == models.JobStatusInProgress {
			start = at(t.TransitionedAt)
		}
		if end == nil && t.ToStatus == models.JobStatusCompleted {
			end = at(t.TransitionedAt)
		}
	}
	if start != nil && end != nil && end.Before(*start) {
		end = nil
	}
	return start, end
}

// reportData turns the job into the report's content, times in the
// tenant's time zone. Prices are never included.
func (s *JobSignoffService) reportData(ctx context.Context, src *repository.JobReportSource, so *models.JobSignoff) *pdf.ProofOfServiceData {
	loc := loadLocationOrUTC(src.Tenant.Settings.Timezone)
	const clock = "3:04 PM"
	job := &src.Job
	d := &pdf.ProofOfServiceData{
		CompanyName: src.Tenant.Name, JobNumber: job.JobNumber, Title: job.Title,
		CustomerName: customerDisplayName(job.Customer), Description: job.Description,
		GeneratedAt: s.now().In(loc).Format("Jan 2, 2006 at 3:04 PM MST"),
	}
	if d.CompanyName == "" {
		d.CompanyName = email.PlatformName
	}
	var contact []string
	for _, v := range []string{src.Tenant.Phone, src.Tenant.Email} {
		if strings.TrimSpace(v) != "" {
			contact = append(contact, v)
		}
	}
	d.CompanyContact = strings.Join(contact, " · ")
	if src.Site != nil {
		d.SiteName = src.Site.Name
		d.SiteAddress = src.Site.FullAddress()
	} else {
		a := job.ServiceAddress
		site := models.Site{AddressStreet: a.Street, AddressCity: a.City, AddressState: a.State, AddressZip: a.Zip}
		d.SiteAddress = site.FullAddress()
	}
	if job.PONumber != nil {
		d.PONumber = *job.PONumber
	}
	if job.CompletionNotes != nil {
		d.CompletionNote = *job.CompletionNotes
	}

	start, end := onSite(src)
	switch {
	case job.ScheduledStartAt != nil:
		d.VisitDate = time.Unix(*job.ScheduledStartAt, 0).In(loc).Format("Mon, Jan 2, 2006")
	case start != nil:
		d.VisitDate = start.In(loc).Format("Mon, Jan 2, 2006")
	default:
		d.VisitDate = so.SignedAt.In(loc).Format("Mon, Jan 2, 2006")
	}
	if job.ScheduledStartAt != nil && job.ScheduledEndAt != nil {
		d.Scheduled = time.Unix(*job.ScheduledStartAt, 0).In(loc).Format(clock) + " - " +
			time.Unix(*job.ScheduledEndAt, 0).In(loc).Format(clock)
	}
	switch {
	case start != nil && end != nil:
		d.OnSite = fmt.Sprintf("%s - %s (%s)", start.In(loc).Format(clock), end.In(loc).Format(clock), formatDuration(end.Sub(*start)))
	case start != nil:
		d.OnSite = "Arrived " + start.In(loc).Format(clock)
	}

	var crew []string
	for i := range job.Assignments {
		if n := models.UserDisplayName(job.Assignments[i].User); n != "" {
			crew = append(crew, n)
		}
	}
	d.Technicians = strings.Join(crew, ", ")
	for _, n := range job.Notes {
		if n.IsInternal {
			continue
		}
		d.Notes = append(d.Notes, pdf.ProofOfServiceNote{
			Author: models.UserDisplayName(n.User), At: n.CreatedAt.In(loc).Format("Jan 2, 3:04 PM"), Text: n.Note,
		})
	}
	for _, m := range job.Materials {
		qty := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", m.Quantity), "0"), ".")
		if m.Unit != "" {
			qty += " " + m.Unit
		}
		d.Materials = append(d.Materials, pdf.ProofOfServiceMaterial{Name: m.Name, Quantity: qty})
	}

	for i, p := range src.Photos {
		if len(d.Photos) == maxReportPhotos {
			d.MorePhotos = len(src.Photos) - i
			break
		}
		key := p.ObjectKey
		if p.ThumbKey != nil {
			key = *p.ThumbKey
		}
		data, err := s.storage.DownloadBytes(ctx, key)
		if err != nil {
			logging.Warn(ctx, "[JOB-SIGNOFF] Photo left out of the report", map[string]any{"attachment_id": p.ID.String(), "error": err})
			d.MorePhotos++
			continue
		}
		caption := ""
		if p.Caption != nil {
			caption = *p.Caption
		}
		d.Photos = append(d.Photos, pdf.ProofOfServicePhoto{JPEG: data, Caption: caption, Taken: p.CreatedAt.In(loc).Format(clock)})
	}

	sig := &pdf.ProofOfServiceSignature{
		Name: so.SignerName, SignedAt: so.SignedAt.In(loc).Format("Mon, Jan 2, 2006 at 3:04 PM MST"),
	}
	if so.SignerTitle != nil {
		sig.Title = *so.SignerTitle
	}
	if n := models.UserDisplayName(so.Capturer); n != "" {
		sig.Device = "On " + n + "'s device"
	}
	if sigPNG, err := s.storage.DownloadBytes(ctx, so.SignatureKey); err == nil {
		sig.PNG = sigPNG
	} else {
		logging.Warn(ctx, "[JOB-SIGNOFF] Signature image missing from the report", map[string]any{"error": err})
	}
	d.Signature = sig
	return d
}

// EmailReport emails the service report to the job's contacts. It sends at
// most once: the send is claimed first, and a repeated request after a
// successful send returns sent=false. resend sends the stored report again.
func (s *JobSignoffService) EmailReport(ctx context.Context, jobID uuid.UUID, resend bool) (*models.EmailJobReportResponse, error) {
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	so, err := s.get(ctx, tenantID, jobID)
	if err != nil {
		return nil, err
	}
	current := func(sent bool) (*models.EmailJobReportResponse, error) {
		fresh, err := s.get(ctx, tenantID, jobID)
		if err != nil {
			return nil, err
		}
		r, err := s.respond(ctx, tenantID, fresh)
		if err != nil {
			return nil, err
		}
		return &models.EmailJobReportResponse{Sent: sent, Signoff: *r}, nil
	}
	if !resend && so.ReportEmailedAt != nil {
		return current(false)
	}
	recipients, err := s.recipients(ctx, tenantID, jobID, so)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, ErrReportNoRecipients
	}
	if s.email == nil {
		return nil, ErrReportEmailUnavailable
	}

	now := s.now().UTC()
	claimed, err := s.repo.ClaimEmail(ctx, tenantID, jobID, resend, now, now.Add(-reportEmailClaimTTL))
	if err != nil {
		return nil, fmt.Errorf("claim report email: %w", err)
	}
	if !claimed {
		fresh, err := s.get(ctx, tenantID, jobID)
		if err != nil {
			return nil, err
		}
		if !resend && fresh.ReportEmailedAt != nil {
			return current(false)
		}
		return nil, ErrReportEmailInProgress
	}
	release := func() {
		if err := s.repo.ReleaseEmail(ctx, tenantID, jobID); err != nil {
			logging.Error(ctx, "[JOB-SIGNOFF] Failed to release the report email claim", map[string]any{"error": err})
		}
	}

	// A resend sends the same document the customer already has; a first
	// send builds it from the job as it is now.
	var rep *reportFile
	if resend && so.ReportEmailedAt != nil && so.ReportKey != nil {
		content, err := s.storage.DownloadBytes(ctx, *so.ReportKey)
		if err == nil {
			src, srcErr := s.repo.ReportSource(ctx, tenantID, jobID)
			if srcErr == nil {
				rep = &reportFile{Content: content, Filename: reportFilename(src.Job.JobNumber), Source: src}
			}
		}
	}
	if rep == nil {
		if rep, _, err = s.buildAndStore(ctx, tenantID, so); err != nil {
			release()
			return nil, err
		}
	}

	msg := reportEmail(rep, so, recipients)
	sendCtx := tenantSenderFor(ctx, &rep.Source.Tenant)
	if _, err := s.email.Send(sendCtx, msg); err != nil {
		release()
		logging.Error(ctx, "[JOB-SIGNOFF] Service report email failed", map[string]any{"job_id": jobID.String(), "error": err})
		return nil, ErrReportEmailFailed.WithCause(err)
	}
	if err := s.repo.FinishEmail(ctx, tenantID, jobID, recipients, s.now().UTC()); err != nil {
		// Sent but not recorded: the claim stays until it goes stale, so
		// an immediate retry can't send it twice.
		logging.Error(ctx, "[JOB-SIGNOFF] Report emailed but not recorded", map[string]any{"job_id": jobID.String(), "error": err})
		return nil, fmt.Errorf("record report email: %w", err)
	}
	logging.Info(ctx, "[JOB-SIGNOFF] Service report emailed", map[string]any{
		"job_id": jobID.String(), "recipients": len(recipients), "resend": resend,
	})
	return current(true)
}

// reportEmail is the message carrying the report: who did what where, who
// signed, the PDF attached. Every value is HTML-escaped.
func reportEmail(rep *reportFile, so *models.JobSignoff, recipients []string) *email.EmailMessage {
	job := &rep.Source.Job
	loc := loadLocationOrUTC(rep.Source.Tenant.Settings.Timezone)
	business := rep.Source.Tenant.Name
	if business == "" {
		business = email.PlatformName
	}
	where := job.Title
	if rep.Source.Site != nil && rep.Source.Site.Name != "" {
		where = rep.Source.Site.Name
	}
	subject := fmt.Sprintf("Service report: %s (%s)", where, job.JobNumber)
	signer := so.SignerName
	if so.SignerTitle != nil && *so.SignerTitle != "" {
		signer += ", " + *so.SignerTitle
	}
	signed := so.SignedAt.In(loc).Format("Mon, Jan 2 at 3:04 PM MST")
	e := html.EscapeString
	body := fmt.Sprintf(`<html><body style="font-family: Arial, sans-serif; color: #171717; line-height: 1.5;">
<p>Hello,</p>
<p>%s completed <strong>%s</strong>%s.</p>
<p>%s signed off on %s.</p>
<p>The service report with the work done, photos and the signature is attached (%s).</p>
<p>Questions about this visit? Reply to this email to reach %s.</p>
</body></html>`,
		e(business), e(job.Title), siteSuffix(rep.Source), e(signer), e(signed), e(rep.Filename), e(business))
	text := fmt.Sprintf("%s completed %s%s.\n%s signed off on %s.\nThe service report is attached (%s).\n",
		business, job.Title, siteSuffixText(rep.Source), signer, signed, rep.Filename)
	msg := email.NewEmailMessage(recipients[0], subject, body).WithTextBody(text)
	msg.To = recipients
	msg.WithAttachment(email.Attachment{Filename: rep.Filename, ContentType: contentTypePDF, Content: rep.Content}).
		WithTag("type", "service_report")
	return msg
}

func siteSuffixText(src *repository.JobReportSource) string {
	if src.Site == nil || src.Site.Name == "" {
		return ""
	}
	return " at " + src.Site.Name
}

func siteSuffix(src *repository.JobReportSource) string {
	return html.EscapeString(siteSuffixText(src))
}
