package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/imports"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// mappedImporter writes mapped records (T193) for one import job: every row
// in its own transaction, under the job's tenant (ctx). Re-importing is
// idempotent: each created customer, job and invoice is recorded in
// import_external_refs under (source, entity, external ID), claimed with
// INSERT … ON CONFLICT DO NOTHING before the row is written, so a second run
// (or a concurrent one) finds the claim and skips the row; sites are
// matched on their address.
type mappedImporter struct {
	db       *gorm.DB
	tenantID uuid.UUID
	userID   uuid.UUID
	jobID    uuid.UUID
	source   imports.Source
	entity   imports.Entity
	loc      *time.Location
	now      time.Time
	// customers caches resolved customer references (only rows that already
	// existed before this run's transactions, so a rollback can't stale it).
	customers map[string]uuid.UUID
}

func (s *ImportService) newMappedImporter(ctx context.Context, job *models.ImportJob) (*mappedImporter, error) {
	src, err := imports.ParseSource(job.Source)
	if err != nil {
		return nil, err
	}
	entity, err := imports.ParseEntity(job.Entity)
	if err != nil {
		return nil, err
	}
	settings := models.TenantSettingsInTx(s.db.WithContext(ctx), job.TenantID)
	return &mappedImporter{
		db: s.db, tenantID: job.TenantID, userID: job.UserID, jobID: job.ID,
		source: src, entity: entity, loc: loadLocationOrUTC(settings.Timezone), now: s.now(),
		customers: map[string]uuid.UUID{},
	}, nil
}

// rowError is a row failure safe to show the user.
type rowError struct{ imports.FieldError }

func failRow(field, format string, args ...any) error {
	return rowError{imports.FieldError{Field: field, Message: fmt.Sprintf(format, args...)}}
}

// importRecord validates and writes one record. The entity writers return
// whether they created the row's record; false means it was there already
// (imported before, or matched an existing customer) and the row is
// counted as skipped.
func (m *mappedImporter) importRecord(ctx context.Context, rec *imports.Record) (rowOutcome, []models.ImportError) {
	if rec.Values == nil {
		rec.Values = map[string]string{}
	}
	if errs := imports.Validate(m.source, m.entity, rec, m.now.In(m.loc)); len(errs) > 0 {
		return rowFailed, m.rowErrors(rec, errs...)
	}
	if rec.Values[imports.SkipKey] != "" {
		return rowSkipped, nil
	}

	// Every write of the row commits together; a skipped row (already
	// imported) commits nothing new, a matched one only its ref.
	var created bool
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		switch m.entity {
		case imports.EntityCustomers:
			created, err = m.customer(tx, rec)
		case imports.EntitySites:
			created, err = m.site(tx, rec)
		case imports.EntityJobs:
			created, err = m.job(tx, rec)
		case imports.EntityInvoices:
			created, err = m.invoice(tx, rec)
		default:
			err = fmt.Errorf("unknown entity %q", m.entity)
		}
		return err
	})
	var re rowError
	switch {
	case err == nil && created:
		return rowImported, nil
	case err == nil:
		return rowSkipped, nil
	case errors.As(err, &re):
		return rowFailed, m.rowErrors(rec, re.FieldError)
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return rowFailed, m.rowErrors(rec, imports.FieldError{Message: "this row conflicts with an existing record (duplicate email or number)"})
		}
		logging.Error(ctx, "[IMPORT-SERVICE] Mapped row failed", map[string]any{"error": err.Error(), "row": rec.Row, "job_id": m.jobID.String()})
		return rowFailed, m.rowErrors(rec, imports.FieldError{Message: "could not save this row"})
	}
}

func (m *mappedImporter) rowErrors(rec *imports.Record, errs ...imports.FieldError) []models.ImportError {
	data, err := json.Marshal(rec.Values)
	if err != nil {
		data = nil
	}
	out := make([]models.ImportError, 0, len(errs))
	for _, e := range errs {
		out = append(out, models.ImportError{RowNumber: rec.Row, Field: e.Field, Error: e.Message, RowData: string(data)})
	}
	return out
}

// --- external refs --------------------------------------------------------

// claimRef records (source, entity, extID) → recordID unless that key is
// already recorded. It returns the recorded ID and whether this call
// created it.
func (m *mappedImporter) claimRef(tx *gorm.DB, entity imports.Entity, extID string, recordID uuid.UUID) (uuid.UUID, bool, error) {
	var ids []uuid.UUID
	err := tx.Raw(`INSERT INTO import_external_refs (tenant_id, source, entity, external_id, record_id, import_job_id)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (tenant_id, source, entity, external_id) DO NOTHING
		RETURNING record_id`, m.tenantID, m.source, entity, extID, recordID, m.jobID).Scan(&ids).Error
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("claim import ref: %w", err)
	}
	if len(ids) == 1 {
		return ids[0], true, nil
	}
	existing, ok, err := m.lookupRef(tx, entity, extID)
	if err != nil {
		return uuid.Nil, false, err
	}
	if !ok {
		return uuid.Nil, false, fmt.Errorf("import ref %s/%s vanished", entity, extID)
	}
	return existing, false, nil
}

func (m *mappedImporter) lookupRef(tx *gorm.DB, entity imports.Entity, extID string) (uuid.UUID, bool, error) {
	var refs []models.ImportExternalRef
	if err := tx.Where("tenant_id = ? AND source = ? AND entity = ? AND external_id = ?", m.tenantID, m.source, entity, extID).
		Limit(1).Find(&refs).Error; err != nil {
		return uuid.Nil, false, fmt.Errorf("look up import ref: %w", err)
	}
	if len(refs) == 0 {
		return uuid.Nil, false, nil
	}
	return refs[0].RecordID, true, nil
}

// repointRef moves a ref to another record (the old one was deleted, or the
// row matched an existing record).
func (m *mappedImporter) repointRef(tx *gorm.DB, entity imports.Entity, extID string, recordID uuid.UUID) error {
	return tx.Model(&models.ImportExternalRef{}).
		Where("tenant_id = ? AND source = ? AND entity = ? AND external_id = ?", m.tenantID, m.source, entity, extID).
		Updates(map[string]any{"record_id": recordID, "import_job_id": m.jobID}).Error
}

// alive reports whether a tenant row (customers, jobs, invoices) exists and
// is not deleted.
func (m *mappedImporter) alive(tx *gorm.DB, table string, id uuid.UUID) (bool, error) {
	var n int64
	err := tx.Table(table).Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, m.tenantID).Count(&n).Error
	return n > 0, err
}

// claimRecord claims extID for a new record. It returns the ID to create
// and true, or the existing record's ID and false when an earlier import
// already created it (and it still exists).
func (m *mappedImporter) claimRecord(tx *gorm.DB, table string, extID string) (uuid.UUID, bool, error) {
	newID := uuid.New()
	id, claimed, err := m.claimRef(tx, m.entity, extID, newID)
	if err != nil || claimed {
		return newID, claimed, err
	}
	ok, err := m.alive(tx, table, id)
	if err != nil {
		return uuid.Nil, false, err
	}
	if ok {
		return id, false, nil
	}
	// Imported before, deleted since: import it again.
	return newID, true, m.repointRef(tx, m.entity, extID, newID)
}

// naturalKey is the external ID of a row whose file has no ID column.
func naturalKey(parts ...string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.Join(parts, "\x1f"))))
	return "key:" + hex.EncodeToString(h[:16])
}

// --- customers --------------------------------------------------------------

func importStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (m *mappedImporter) customerByEmail(tx *gorm.DB, email string) (*models.Customer, error) {
	var cs []models.Customer
	if err := tx.Where("tenant_id = ? AND LOWER(email) = LOWER(?)", m.tenantID, email).Limit(1).Find(&cs).Error; err != nil {
		return nil, fmt.Errorf("find customer by email: %w", err)
	}
	if len(cs) == 0 {
		return nil, nil
	}
	return &cs[0], nil
}

func (m *mappedImporter) customer(tx *gorm.DB, rec *imports.Record) (bool, error) {
	v := rec.Values
	email := v[imports.FEmail]
	extID := v[imports.FExternalID]
	if extID == "" {
		extID = "email:" + strings.ToLower(email)
	}
	id, isNew, err := m.claimRecord(tx, "customers", extID)
	if err != nil {
		return false, err
	}
	if !isNew {
		// Imported before: only add a site the earlier run didn't have.
		_, err := m.customerSite(tx, id, rec)
		return false, err
	}
	// The customer may already be in ServicePro (entered by hand, or from
	// another source): link to it instead of failing on the email.
	existing, err := m.customerByEmail(tx, email)
	if err != nil {
		return false, err
	}
	if existing != nil {
		if err := m.repointRef(tx, m.entity, extID, existing.ID); err != nil {
			return false, err
		}
		_, err := m.customerSite(tx, existing.ID, rec)
		return false, err
	}

	c := &models.Customer{
		ID:                     id,
		FirstName:              v[imports.FFirstName],
		LastName:               v[imports.FLastName],
		CompanyName:            importStr(v[imports.FCompanyName]),
		Email:                  email,
		PhonePrimary:           v[imports.FPhone],
		PhoneSecondary:         importStr(v[imports.FPhoneSecondary]),
		BillingAddressStreet:   v[imports.FBillingStreet],
		BillingAddressCity:     v[imports.FBillingCity],
		BillingAddressState:    v[imports.FBillingState],
		BillingAddressZip:      v[imports.FBillingZip],
		CustomerType:           models.CustomerType(v[imports.FCustomerType]),
		Status:                 models.CustomerStatus(v[imports.FStatus]),
		Notes:                  importStr(v[imports.FNotes]),
		PreferredContactMethod: models.ContactMethodAny,
	}
	// The customer's own service address columns are US-only (database
	// checks); any other address still becomes a site below.
	if street := v[imports.FStreet]; street != "" && v[imports.FCity] != "" &&
		imports.IsUSState(v[imports.FState]) && imports.IsUSZip(v[imports.FZip]) {
		c.ServiceAddressStreet = importStr(street)
		c.ServiceAddressCity = importStr(v[imports.FCity])
		c.ServiceAddressState = importStr(v[imports.FState])
		c.ServiceAddressZip = importStr(v[imports.FZip])
	}
	c.InheritTenantBillingDefaults(nil)
	if err := tx.Create(c).Error; err != nil {
		return false, err
	}
	if _, err := m.customerSite(tx, c.ID, rec); err != nil {
		return false, err
	}
	return true, nil
}

// customerSite adds the customer row's service address as a site (when it
// has one and the customer has no site at that address).
func (m *mappedImporter) customerSite(tx *gorm.DB, customerID uuid.UUID, rec *imports.Record) (bool, error) {
	v := rec.Values
	if v[imports.FStreet] == "" {
		return false, nil
	}
	_, created, err := m.ensureSite(tx, customerID, models.Site{
		Name: v[imports.FSiteName], AddressStreet: v[imports.FStreet], AddressCity: v[imports.FCity],
		AddressState: v[imports.FState], AddressZip: v[imports.FZip],
	})
	return created, err
}

// ensureSite returns the customer's site at the same street (and ZIP, when
// both have one), creating it from want when there is none.
func (m *mappedImporter) ensureSite(tx *gorm.DB, customerID uuid.UUID, want models.Site) (*models.Site, bool, error) {
	var sites []models.Site
	q := tx.Where("tenant_id = ? AND customer_id = ? AND LOWER(TRIM(address_street)) = LOWER(TRIM(?))", m.tenantID, customerID, want.AddressStreet)
	if want.AddressZip != "" {
		q = q.Where("(address_zip = '' OR address_zip = ?)", want.AddressZip)
	}
	if err := q.Order("created_at").Limit(1).Find(&sites).Error; err != nil {
		return nil, false, fmt.Errorf("find site: %w", err)
	}
	if len(sites) > 0 {
		return &sites[0], false, nil
	}
	site := want
	site.ID = uuid.New()
	site.CustomerID = customerID
	site.IsActive = true
	if strings.TrimSpace(site.Name) == "" {
		site.Name = site.AddressStreet
	}
	if err := tx.Create(&site).Error; err != nil {
		return nil, false, fmt.Errorf("create site: %w", err)
	}
	return &site, true, nil
}

// resolveCustomer finds the customer a site/job/invoice row names: by the
// old system's customer ID (imported earlier from the same source), else by
// email, else by exact company or contact name.
func (m *mappedImporter) resolveCustomer(tx *gorm.DB, rec *imports.Record) (uuid.UUID, error) {
	v := rec.Values
	if ext := v[imports.FCustomerExternalID]; ext != "" {
		key := "x:" + ext
		if id, ok := m.customers[key]; ok {
			return id, nil
		}
		id, ok, err := m.lookupRef(tx, imports.EntityCustomers, ext)
		if err != nil {
			return uuid.Nil, err
		}
		if ok {
			if alive, err := m.alive(tx, "customers", id); err != nil {
				return uuid.Nil, err
			} else if alive {
				m.customers[key] = id
				return id, nil
			}
		}
	}
	if email := v[imports.FCustomerEmail]; email != "" {
		key := "e:" + strings.ToLower(email)
		if id, ok := m.customers[key]; ok {
			return id, nil
		}
		c, err := m.customerByEmail(tx, email)
		if err != nil {
			return uuid.Nil, err
		}
		if c != nil {
			m.customers[key] = c.ID
			return c.ID, nil
		}
	}
	if name := strings.TrimSpace(v[imports.FCustomerName]); name != "" {
		key := "n:" + strings.ToLower(name)
		if id, ok := m.customers[key]; ok {
			return id, nil
		}
		var ids []uuid.UUID
		if err := tx.Model(&models.Customer{}).
			Where("tenant_id = ? AND (LOWER(company_name) = LOWER(?) OR LOWER(TRIM(first_name || ' ' || last_name)) = LOWER(?))", m.tenantID, name, name).
			Limit(3).Pluck("id", &ids).Error; err != nil {
			return uuid.Nil, fmt.Errorf("find customer by name: %w", err)
		}
		switch len(ids) {
		case 1:
			m.customers[key] = ids[0]
			return ids[0], nil
		case 0:
		default:
			return uuid.Nil, failRow(imports.FCustomerName, "several customers are named %q: add a customer email column to tell them apart", name)
		}
	}
	return uuid.Nil, failRow("", "no customer matches this row (%s): import customers first", describeCustomerRef(v))
}

func describeCustomerRef(v map[string]string) string {
	var parts []string
	for _, k := range []string{imports.FCustomerExternalID, imports.FCustomerEmail, imports.FCustomerName} {
		if v[k] != "" {
			parts = append(parts, v[k])
		}
	}
	return strings.Join(parts, ", ")
}

// --- sites -------------------------------------------------------------------

func (m *mappedImporter) site(tx *gorm.DB, rec *imports.Record) (bool, error) {
	v := rec.Values
	customerID, err := m.resolveCustomer(tx, rec)
	if err != nil {
		return false, err
	}
	_, created, err := m.ensureSite(tx, customerID, models.Site{
		Name: v[imports.FName], AddressStreet: v[imports.FStreet], AddressCity: v[imports.FCity],
		AddressState: v[imports.FState], AddressZip: v[imports.FZip],
		AccessNotes: importStr(v[imports.FAccessNotes]), ContactName: importStr(v[imports.FContactName]),
		ContactPhone: importStr(v[imports.FContactPhone]), ContactEmail: importStr(v[imports.FContactEmail]),
	})
	return created, err
}

// --- jobs --------------------------------------------------------------------

func (m *mappedImporter) job(tx *gorm.DB, rec *imports.Record) (bool, error) {
	v := rec.Values
	customerID, err := m.resolveCustomer(tx, rec)
	if err != nil {
		return false, err
	}
	number := v[imports.FExternalID]
	extID := number
	if extID == "" {
		extID = naturalKey(customerID.String(), v[imports.FTitle], v[imports.FScheduledStart])
	}
	id, isNew, err := m.claimRecord(tx, "jobs", extID)
	if err != nil || !isNew {
		return false, err
	}

	job := &models.Job{
		ID:            id,
		CustomerID:    customerID,
		Title:         v[imports.FTitle],
		Description:   v[imports.FDescription],
		JobType:       models.JobType(v[imports.FJobType]),
		Status:        models.JobStatus(v[imports.FStatus]),
		Priority:      models.JobPriorityNormal,
		PONumber:      importStr(v[imports.FPONumber]),
		InternalNotes: importStr(m.importNote(fmt.Sprintf("job %s", number), v[imports.FNotes])),
		CreatedBy:     m.userID,
	}
	// Keep the old job number when it is free in this tenant.
	if number != "" {
		var taken int64
		if err := tx.Model(&models.Job{}).Unscoped().Where("tenant_id = ? AND job_number = ?", m.tenantID, number).Count(&taken).Error; err != nil {
			return false, err
		}
		if taken == 0 {
			job.JobNumber = number
		}
	}
	if err := m.jobSchedule(job, v); err != nil {
		return false, err
	}
	if err := m.jobSite(tx, job, customerID, v); err != nil {
		return false, err
	}
	if err := tx.Create(job).Error; err != nil {
		return false, err
	}
	return true, nil
}

// importNote is the "Imported from …" line for internal notes, before any
// notes from the file.
func (m *mappedImporter) importNote(what, notes string) string {
	note := "Imported from " + m.source.Label()
	if strings.TrimSpace(what) != "" && !strings.HasSuffix(what, " ") {
		note += " (" + what + ")"
	}
	if notes != "" {
		note += "\n" + notes
	}
	return note
}

// jobSchedule sets the job's times from the normalised dates (tenant-local
// wall time). A date alone is scheduled at 08:00.
func (m *mappedImporter) jobSchedule(job *models.Job, v map[string]string) error {
	if s := v[imports.FScheduledStart]; s != "" {
		t, hasTime, err := imports.ParseLocal(s, m.loc)
		if err != nil {
			return failRow(imports.FScheduledStart, "scheduled start %q is not a date", s)
		}
		if !hasTime {
			t = t.Add(8 * time.Hour)
		}
		start := t.Unix()
		job.ScheduledStartAt = &start
	}
	if s := v[imports.FScheduledEnd]; s != "" && job.ScheduledStartAt != nil {
		t, hasTime, err := imports.ParseLocal(s, m.loc)
		if err == nil {
			if !hasTime {
				t = t.Add(17 * time.Hour)
			}
			if end := t.Unix(); end > *job.ScheduledStartAt {
				job.ScheduledEndAt = &end
				job.EstimatedDuration = int((end - *job.ScheduledStartAt) / 60)
			}
		}
	}
	if s := v[imports.FCompletedAt]; s != "" {
		if t, hasTime, err := imports.ParseLocal(s, m.loc); err == nil {
			if !hasTime {
				t = t.Add(17 * time.Hour)
			}
			done := t.Unix()
			job.ActualEndAt = &done
		}
	}
	return nil
}

// jobSite points the job at the customer's site for its address (created
// when missing); without an address, at the customer's only site.
func (m *mappedImporter) jobSite(tx *gorm.DB, job *models.Job, customerID uuid.UUID, v map[string]string) error {
	if street := v[imports.FStreet]; street != "" {
		site, _, err := m.ensureSite(tx, customerID, models.Site{
			AddressStreet: street, AddressCity: v[imports.FCity], AddressState: v[imports.FState], AddressZip: v[imports.FZip],
		})
		if err != nil {
			return err
		}
		job.SiteID, job.ServiceAddress = &site.ID, site.ServiceAddress()
		return nil
	}
	var sites []models.Site
	if err := tx.Where("tenant_id = ? AND customer_id = ?", m.tenantID, customerID).Limit(2).Find(&sites).Error; err != nil {
		return err
	}
	if len(sites) == 1 {
		job.SiteID, job.ServiceAddress = &sites[0].ID, sites[0].ServiceAddress()
		return nil
	}
	var c models.Customer
	if err := tx.Where("id = ? AND tenant_id = ?", customerID, m.tenantID).First(&c).Error; err != nil {
		return err
	}
	job.ServiceAddress = models.ServiceAddress{Street: c.BillingAddressStreet, City: c.BillingAddressCity, State: c.BillingAddressState, Zip: c.BillingAddressZip}
	return nil
}

// --- invoices ----------------------------------------------------------------

// invoice imports an open invoice as a sent invoice for its unpaid balance:
// one line carrying the balance, so ServicePro's totals and payments start
// from what is still owed (payments made in the old system are not
// re-created). The original total is kept in the notes.
func (m *mappedImporter) invoice(tx *gorm.DB, rec *imports.Record) (bool, error) {
	v := rec.Values
	customerID, err := m.resolveCustomer(tx, rec)
	if err != nil {
		return false, err
	}
	number := v[imports.FInvoiceNumber]
	id, isNew, err := m.claimRecord(tx, "invoices", number)
	if err != nil || !isNew {
		return false, err
	}
	var taken int64
	if err := tx.Model(&models.Invoice{}).Unscoped().Where("tenant_id = ? AND invoice_number = ?", m.tenantID, number).Count(&taken).Error; err != nil {
		return false, err
	}
	if taken > 0 {
		return false, failRow(imports.FInvoiceNumber, "invoice %s already exists in ServicePro", number)
	}

	balance, err := decimal.NewFromString(v[imports.FBalance])
	if err != nil {
		return false, failRow(imports.FBalance, "balance due is not an amount")
	}
	today := m.now.In(m.loc)
	issue := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if s := v[imports.FIssueDate]; s != "" {
		if t, _, err := imports.ParseLocal(s, time.UTC); err == nil {
			issue = t
		}
	}
	due := issue
	if s := v[imports.FDueDate]; s != "" {
		if t, _, err := imports.ParseLocal(s, time.UTC); err == nil && !t.Before(issue) {
			due = t
		}
	}

	desc := fmt.Sprintf("Balance carried over from %s invoice %s", m.source.Label(), number)
	if subj := v[imports.FSubject]; subj != "" {
		desc += " (" + subj + ")"
	}
	notes := fmt.Sprintf("Imported from %s invoice %s. Original total %s; balance due when imported %s.",
		m.source.Label(), number, v[imports.FTotal], balance.StringFixed(2))
	if n := v[imports.FNotes]; n != "" {
		notes += "\n" + n
	}
	sent := issue
	inv := &models.Invoice{
		ID:            id,
		InvoiceNumber: number,
		CustomerID:    customerID,
		Status:        models.InvoiceStatusSent,
		IssueDate:     issue,
		DueDate:       due,
		SentDate:      &sent,
		PONumber:      v[imports.FPONumber],
		Notes:         notes,
		CreatedBy:     m.userID,
		Lines: []models.InvoiceLine{{
			Description: desc, Quantity: decimal.NewFromInt(1), UnitPrice: balance, UnitOfMeasure: "each",
		}},
	}
	if err := insertInvoice(tx, inv, m.userID); err != nil {
		return false, err
	}
	return true, nil
}
