package quickbooks

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	qbo "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
)

const (
	statusErrorsLimit = 50
	statusRecentLimit = 10
)

// Status is the Integrations settings view of the tenant's QuickBooks.
func (s *Service) Status(ctx context.Context, tenantID uuid.UUID) (*models.QuickBooksStatusResponse, error) {
	out := &models.QuickBooksStatusResponse{
		Configured: s.Configured(), Status: models.QuickBooksDisconnected,
		Errors: []models.QuickBooksSyncRecordResponse{}, Recent: []models.QuickBooksSyncRecordResponse{},
	}
	if !out.Configured {
		return out, nil
	}
	conn, err := s.loadConnection(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return out, nil
	}
	out.Status = conn.Status
	if conn.Status == models.QuickBooksDisconnected {
		return out, nil
	}
	out.Environment = conn.Environment
	out.CompanyName = deref(conn.CompanyName)
	out.ConnectedAt = conn.ConnectedAt
	if conn.SyncFrom != nil {
		d := conn.SyncFrom.Format("2006-01-02")
		out.SyncFrom = &d
	}
	out.LastSyncAt = conn.LastSyncAt
	out.LastSyncError = deref(conn.LastSyncError)
	out.IncomeAccount = ref(conn.IncomeAccountID, conn.IncomeAccountName)
	out.Item = ref(conn.ItemID, conn.ItemName)
	out.TaxCode = ref(conn.TaxCodeID, conn.TaxCodeName)
	out.DepositAccount = ref(conn.DepositAccountID, conn.DepositAccountName)

	db := s.db.WithContext(tenantCtx(ctx, tenantID))
	var counts []struct {
		Entity string
		Status string
		N      int64
	}
	if err := db.Raw(`SELECT entity, status, COUNT(*) AS n FROM quickbooks_sync_map WHERE tenant_id = ? GROUP BY entity, status`,
		tenantID).Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, c := range counts {
		switch {
		case c.Status == models.QuickBooksSyncError:
			out.Counts.Errors += c.N
		case c.Status == models.QuickBooksSyncPending:
			out.Counts.Pending += c.N
		case c.Entity == models.QuickBooksEntityCustomer:
			out.Counts.Customers += c.N
		case c.Entity == models.QuickBooksEntityInvoice:
			out.Counts.Invoices += c.N
		case c.Entity == models.QuickBooksEntityPayment:
			out.Counts.Payments += c.N
		}
	}

	var errs, recent []models.QuickBooksSyncRecord
	if err := db.Where("tenant_id = ? AND status = ?", tenantID, models.QuickBooksSyncError).
		Order("updated_at DESC").Limit(statusErrorsLimit).Find(&errs).Error; err != nil {
		return nil, err
	}
	if err := db.Where("tenant_id = ? AND status = ? AND last_synced_at IS NOT NULL", tenantID, models.QuickBooksSyncSynced).
		Order("last_synced_at DESC").Limit(statusRecentLimit).Find(&recent).Error; err != nil {
		return nil, err
	}
	labels, err := s.labels(ctx, tenantID, append(append([]models.QuickBooksSyncRecord{}, errs...), recent...))
	if err != nil {
		return nil, err
	}
	for _, r := range errs {
		out.Errors = append(out.Errors, recordResponse(r, labels))
	}
	for _, r := range recent {
		out.Recent = append(out.Recent, recordResponse(r, labels))
	}
	return out, nil
}

func recordResponse(r models.QuickBooksSyncRecord, labels map[uuid.UUID]string) models.QuickBooksSyncRecordResponse {
	label := labels[r.LocalID]
	if label == "" {
		label = strings.ToUpper(r.Entity[:1]) + r.Entity[1:] + " (deleted)"
	}
	return models.QuickBooksSyncRecordResponse{
		ID: r.ID, Entity: r.Entity, LocalID: r.LocalID, Label: label, Status: r.Status, Attempts: r.Attempts,
		LastError: deref(r.LastError), LastSyncedAt: r.LastSyncedAt, UpdatedAt: r.UpdatedAt,
	}
}

// labels describes records for the activity and error lists, e.g.
// "Invoice INV-1042 · Harborview Property Management · $9,882.00".
func (s *Service) labels(ctx context.Context, tenantID uuid.UUID, recs []models.QuickBooksSyncRecord) (map[uuid.UUID]string, error) {
	ids := map[string][]uuid.UUID{}
	for _, r := range recs {
		ids[r.Entity] = append(ids[r.Entity], r.LocalID)
	}
	out := map[uuid.UUID]string{}
	db := s.db.WithContext(tenantCtx(ctx, tenantID))
	type row struct {
		ID       uuid.UUID
		Number   string
		Company  *string
		First    string
		Last     string
		Amount   decimal.Decimal
		Currency string
	}
	name := func(r row) string {
		if r.Company != nil && strings.TrimSpace(*r.Company) != "" {
			return *r.Company
		}
		return strings.TrimSpace(r.First + " " + r.Last)
	}
	if len(ids[models.QuickBooksEntityCustomer]) > 0 {
		var rows []row
		if err := db.Raw(`SELECT id, company_name AS company, first_name AS first, last_name AS last
			FROM customers WHERE tenant_id = ? AND id IN ?`, tenantID, ids[models.QuickBooksEntityCustomer]).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out[r.ID] = "Customer " + name(r)
		}
	}
	if len(ids[models.QuickBooksEntityInvoice]) > 0 {
		var rows []row
		if err := db.Raw(`SELECT i.id, i.invoice_number AS number, i.total_amount AS amount, i.currency,
				c.company_name AS company, c.first_name AS first, c.last_name AS last
			FROM invoices i JOIN customers c ON c.id = i.customer_id AND c.tenant_id = i.tenant_id
			WHERE i.tenant_id = ? AND i.id IN ?`, tenantID, ids[models.QuickBooksEntityInvoice]).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out[r.ID] = joinLabel("Invoice "+r.Number, name(r), formatMoney(r.Amount, r.Currency))
		}
	}
	if len(ids[models.QuickBooksEntityPayment]) > 0 {
		var rows []row
		if err := db.Raw(`SELECT p.id, i.invoice_number AS number, p.amount, i.currency,
				c.company_name AS company, c.first_name AS first, c.last_name AS last
			FROM invoice_payments p JOIN invoices i ON i.id = p.invoice_id
			JOIN customers c ON c.id = i.customer_id AND c.tenant_id = i.tenant_id
			WHERE i.tenant_id = ? AND p.id IN ?`, tenantID, ids[models.QuickBooksEntityPayment]).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out[r.ID] = joinLabel("Payment "+formatMoney(r.Amount, r.Currency)+" on "+r.Number, name(r))
		}
	}
	return out, nil
}

// formatMoney renders an amount like $9,882.00 (other currencies get their
// code as a suffix).
func formatMoney(d decimal.Decimal, currency string) string {
	s := d.Abs().StringFixed(2)
	whole, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	sign := ""
	if d.IsNegative() {
		sign = "-"
	}
	if currency == "" || strings.EqualFold(currency, "USD") {
		return sign + "$" + b.String() + "." + frac
	}
	return sign + b.String() + "." + frac + " " + strings.ToUpper(currency)
}

// optionsTimeout bounds the QuickBooks lookups of the settings form.
const optionsTimeout = 20 * time.Second

// Options lists the income accounts, tax codes and deposit accounts of the
// connected company.
func (s *Service) Options(ctx context.Context, tenantID uuid.UUID) (*models.QuickBooksOptionsResponse, error) {
	if _, err := s.connected(ctx, tenantID); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, optionsTimeout)
	defer cancel()
	w := &worker{s: s, tenantID: tenantID, ctx: tenantCtx(ctx, tenantID)}
	out := &models.QuickBooksOptionsResponse{}
	toRefs := func(list []qbo.Ref) []models.QuickBooksRef {
		refs := make([]models.QuickBooksRef, 0, len(list))
		for _, r := range list {
			refs = append(refs, models.QuickBooksRef{ID: r.Value, Name: r.Name})
		}
		return refs
	}
	err := w.call(func(sess qbo.Session) error {
		income, err := s.client.ListAccounts(ctx, sess, qbo.AccountTypeIncome)
		if err != nil {
			return err
		}
		taxes, err := s.client.ListTaxCodes(ctx, sess)
		if err != nil {
			return err
		}
		bank, err := s.client.ListAccounts(ctx, sess, qbo.AccountTypeBank)
		if err != nil {
			return err
		}
		other, err := s.client.ListAccounts(ctx, sess, qbo.AccountTypeOtherCurrentAsset)
		if err != nil {
			return err
		}
		out.IncomeAccounts, out.TaxCodes, out.DepositAccounts = toRefs(income), toRefs(taxes), toRefs(append(bank, other...))
		return nil
	})
	if err != nil {
		return nil, s.apiFailure(ctx, tenantID, "options", err)
	}
	return out, nil
}

// apiFailure maps a QuickBooks failure on an interactive request.
func (s *Service) apiFailure(ctx context.Context, tenantID uuid.UUID, op string, err error) error {
	if errors.Is(err, ErrReauthRequired) || errors.Is(err, ErrNotConnected) || errors.Is(err, ErrNotConfigured) || errors.Is(err, ErrInvalidSetting) {
		return err
	}
	logging.Error(ctx, "[QUICKBOOKS] API call failed", map[string]any{"tenant_id": tenantID, "op": op, "error": err.Error()})
	return ErrQuickBooksUnavailable
}

func findRef(list []qbo.Ref, id string) (qbo.Ref, bool) {
	for _, r := range list {
		if r.Value == id {
			return r, true
		}
	}
	return qbo.Ref{}, false
}

// UpdateSettings saves the mapping: the income account (ServicePro finds or
// creates a service item "ServicePro - <account>" on it for invoice lines),
// an optional tax code and an optional deposit account for payments
// (QuickBooks defaults to Undeposited Funds). Failed records are retried.
func (s *Service) UpdateSettings(ctx context.Context, tenantID uuid.UUID, req models.QuickBooksSettingsRequest) (*models.QuickBooksStatusResponse, error) {
	if _, err := s.connected(ctx, tenantID); err != nil {
		return nil, err
	}
	ictx, cancel := context.WithTimeout(ctx, optionsTimeout)
	defer cancel()
	w := &worker{s: s, tenantID: tenantID, ctx: tenantCtx(ictx, tenantID)}
	updates := map[string]any{}
	err := w.call(func(sess qbo.Session) error {
		if id := strings.TrimSpace(req.IncomeAccountID); id != "" {
			income, err := s.client.ListAccounts(ictx, sess, qbo.AccountTypeIncome)
			if err != nil {
				return err
			}
			acct, ok := findRef(income, id)
			if !ok {
				return ErrInvalidSetting
			}
			name := itemName(acct.Name)
			item, err := s.client.FindItemByName(ictx, sess, name)
			if err != nil {
				return err
			}
			if item == nil {
				item, err = s.client.CreateItem(ictx, sess, qbo.Item{Name: name, Type: "Service", IncomeAccountRef: qbo.Ref{Value: acct.Value}},
					"sp-item-"+tenantID.String()[:8]+"-"+acct.Value)
				if err != nil {
					return err
				}
			}
			updates["income_account_id"], updates["income_account_name"] = acct.Value, acct.Name
			updates["item_id"], updates["item_name"] = item.ID, name
		}
		updates["tax_code_id"], updates["tax_code_name"] = nil, nil
		if id := strings.TrimSpace(req.TaxCodeID); id != "" {
			codes, err := s.client.ListTaxCodes(ictx, sess)
			if err != nil {
				return err
			}
			code, ok := findRef(codes, id)
			if !ok {
				return ErrInvalidSetting
			}
			updates["tax_code_id"], updates["tax_code_name"] = code.Value, code.Name
		}
		updates["deposit_account_id"], updates["deposit_account_name"] = nil, nil
		if id := strings.TrimSpace(req.DepositAccountID); id != "" {
			bank, err := s.client.ListAccounts(ictx, sess, qbo.AccountTypeBank)
			if err != nil {
				return err
			}
			other, err := s.client.ListAccounts(ictx, sess, qbo.AccountTypeOtherCurrentAsset)
			if err != nil {
				return err
			}
			acct, ok := findRef(append(bank, other...), id)
			if !ok {
				return ErrInvalidSetting
			}
			updates["deposit_account_id"], updates["deposit_account_name"] = acct.Value, acct.Name
		}
		return nil
	})
	if err != nil {
		return nil, s.apiFailure(ctx, tenantID, "settings", err)
	}
	db := s.db.WithContext(tenantCtx(ctx, tenantID))
	if err := db.Model(&models.QuickBooksConnection{}).Where("tenant_id = ?", tenantID).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.retryErrors(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.Status(ctx, tenantID)
}
