package quickbooks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	qbo "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
)

// Sync limits.
const (
	// MaxAttempts: a record that failed this many times stops retrying on
	// its own (the error stays listed until someone retries it).
	MaxAttempts = 8
	// leaseTTL outlives the 60 s Cloud Run request timeout, so a crashed
	// run's lease expires before the next hourly run.
	leaseTTL = 90 * time.Second
	// maxPerTenantRun bounds one tenant's records per pass (QuickBooks
	// allows 500 requests per minute per company).
	maxPerTenantRun = 150
	// sweepLimit bounds how many records one sweep enqueues per entity.
	sweepLimit = 500
	// defaultItemName is the item every new QuickBooks company has.
	defaultItemName = "Services"
)

// RunResult reports one sync pass.
type RunResult struct {
	Tenants  int  `json:"tenants" example:"2"`
	Enqueued int  `json:"enqueued" example:"5"`
	Synced   int  `json:"synced" example:"5"`
	Failed   int  `json:"failed" example:"0"`
	Skipped  int  `json:"skipped" example:"0"`
	More     bool `json:"more" example:"false"`
}

func (r *RunResult) add(o RunResult) {
	r.Enqueued += o.Enqueued
	r.Synced += o.Synced
	r.Failed += o.Failed
	r.Skipped += o.Skipped
	r.More = r.More || o.More
}

// Enqueue marks one record for the next sync, if the tenant has QuickBooks
// connected (one statement; a no-op otherwise). Called after an invoice is
// sent or a payment recorded; failures only delay the record to the sweep.
func (s *Service) Enqueue(ctx context.Context, tenantID uuid.UUID, entity string, localID uuid.UUID) error {
	if !s.Configured() {
		return nil
	}
	return s.db.WithContext(tenantCtx(ctx, tenantID)).Exec(`
		INSERT INTO quickbooks_sync_map (tenant_id, entity, local_id, status, next_attempt_at)
		SELECT ?, ?, ?, 'pending', ?
		WHERE EXISTS (SELECT 1 FROM quickbooks_connections WHERE tenant_id = ? AND status = 'connected')
		ON CONFLICT (tenant_id, entity, local_id) DO UPDATE
		   SET status = 'pending', attempts = 0, next_attempt_at = EXCLUDED.next_attempt_at, updated_at = NOW()
		 WHERE quickbooks_sync_map.status <> 'pending'`,
		tenantID, entity, localID, s.now(), tenantID).Error
}

// EnqueueAfterCommit enqueues best effort and logs a failure (the sweep
// picks the record up anyway).
func (s *Service) EnqueueAfterCommit(ctx context.Context, tenantID uuid.UUID, entity string, localID uuid.UUID) {
	if s == nil {
		return
	}
	if err := s.Enqueue(ctx, tenantID, entity, localID); err != nil {
		logging.Warn(ctx, "[QUICKBOOKS] Enqueue failed; the next sweep will pick it up", map[string]any{
			"tenant_id": tenantID, "entity": entity, "local_id": localID, "error": err.Error(),
		})
	}
}

// sweep enqueues records the event hooks didn't: invoices issued since the
// connection's sync_from (new, or edited since their last sync), edited
// synced customers, and payments on synced-period invoices.
func (s *Service) sweep(ctx context.Context, conn *models.QuickBooksConnection) (int, error) {
	db := s.db.WithContext(tenantCtx(ctx, conn.TenantID))
	from := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	if conn.SyncFrom != nil {
		from = *conn.SyncFrom
	}
	total, now := 0, s.now()
	res := db.Exec(`
		INSERT INTO quickbooks_sync_map (tenant_id, entity, local_id, status, next_attempt_at)
		SELECT i.tenant_id, 'invoice', i.id, 'pending', ?::timestamptz
		  FROM invoices i
		  LEFT JOIN quickbooks_sync_map m ON m.tenant_id = i.tenant_id AND m.entity = 'invoice' AND m.local_id = i.id
		 WHERE i.tenant_id = ? AND i.deleted_at IS NULL AND i.issue_date >= ?
		   AND i.status NOT IN ('draft', 'cancelled')
		   AND (m.id IS NULL OR (m.status = 'synced' AND m.local_updated_at IS NOT NULL AND i.updated_at > m.local_updated_at))
		 ORDER BY i.issue_date, i.created_at
		 LIMIT ?
		ON CONFLICT (tenant_id, entity, local_id) DO UPDATE
		   SET status = 'pending', next_attempt_at = EXCLUDED.next_attempt_at, updated_at = NOW()`,
		now, conn.TenantID, from, sweepLimit)
	if res.Error != nil {
		return total, res.Error
	}
	total += int(res.RowsAffected)
	res = db.Exec(`
		UPDATE quickbooks_sync_map m
		   SET status = 'pending', next_attempt_at = ?, updated_at = NOW()
		  FROM customers c
		 WHERE m.tenant_id = ? AND m.entity = 'customer' AND m.status = 'synced'
		   AND c.id = m.local_id AND c.tenant_id = m.tenant_id
		   AND m.local_updated_at IS NOT NULL AND c.updated_at > m.local_updated_at`,
		now, conn.TenantID)
	if res.Error != nil {
		return total, res.Error
	}
	total += int(res.RowsAffected)
	res = db.Exec(`
		INSERT INTO quickbooks_sync_map (tenant_id, entity, local_id, status, next_attempt_at)
		SELECT i.tenant_id, 'payment', p.id, 'pending', ?::timestamptz
		  FROM invoice_payments p
		  JOIN invoices i ON i.id = p.invoice_id
		  LEFT JOIN quickbooks_sync_map m ON m.tenant_id = i.tenant_id AND m.entity = 'payment' AND m.local_id = p.id
		 WHERE i.tenant_id = ? AND i.deleted_at IS NULL AND i.issue_date >= ? AND m.id IS NULL
		 ORDER BY p.created_at
		 LIMIT ?
		ON CONFLICT (tenant_id, entity, local_id) DO NOTHING`,
		now, conn.TenantID, from, sweepLimit)
	if res.Error != nil {
		return total, res.Error
	}
	total += int(res.RowsAffected)
	return total, nil
}

// acquireLease takes the tenant's sync lease; false when another run holds
// it (or the tenant isn't connected).
func (s *Service) acquireLease(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	now := s.now()
	res := s.db.WithContext(tenantCtx(ctx, tenantID)).Model(&models.QuickBooksConnection{}).
		Where("tenant_id = ? AND status = ? AND (sync_locked_until IS NULL OR sync_locked_until < ?)", tenantID, models.QuickBooksConnected, now).
		Update("sync_locked_until", now.Add(leaseTTL))
	return res.RowsAffected == 1, res.Error
}

func (s *Service) releaseLease(ctx context.Context, tenantID uuid.UUID, lastErr *string) {
	updates := map[string]any{"sync_locked_until": nil, "last_sync_at": s.now()}
	if lastErr != nil {
		updates["last_sync_error"] = strPtr(*lastErr) // "" clears it
	}
	err := s.db.WithContext(tenantCtx(context.WithoutCancel(ctx), tenantID)).Model(&models.QuickBooksConnection{}).
		Where("tenant_id = ?", tenantID).Updates(updates).Error
	if err != nil {
		logging.Error(ctx, "[QUICKBOOKS] Could not release sync lease", map[string]any{"tenant_id": tenantID, "error": err.Error()})
	}
}

// ProcessPending is the scheduled job: it syncs each connected tenant (the
// least recently synced first) until budget is spent.
func (s *Service) ProcessPending(ctx context.Context, budget time.Duration) (RunResult, error) {
	var result RunResult
	if !s.Configured() {
		return result, nil
	}
	deadline := s.now().Add(budget)
	var tenants []uuid.UUID
	err := s.db.WithContext(tenancy.WithSystem(ctx)).Model(&models.QuickBooksConnection{}).
		Where("status = ?", models.QuickBooksConnected).
		Order("last_sync_at NULLS FIRST").Pluck("tenant_id", &tenants).Error
	if err != nil {
		return result, err
	}
	for _, tenantID := range tenants {
		if !s.now().Before(deadline) {
			result.More = true
			break
		}
		r, err := s.SyncTenant(ctx, tenantID, deadline)
		if err != nil {
			logging.Error(ctx, "[QUICKBOOKS] Tenant sync failed", map[string]any{"tenant_id": tenantID, "error": err.Error()})
			continue
		}
		result.Tenants++
		result.add(r)
	}
	return result, nil
}

// SyncTenant sweeps and pushes one tenant's pending records until deadline.
// Another run already syncing the tenant makes this a no-op.
func (s *Service) SyncTenant(ctx context.Context, tenantID uuid.UUID, deadline time.Time) (RunResult, error) {
	var result RunResult
	conn, err := s.connected(ctx, tenantID)
	if err != nil {
		return result, err
	}
	ok, err := s.acquireLease(ctx, tenantID)
	if err != nil || !ok {
		return result, err
	}
	var connErr *string
	defer func() { s.releaseLease(ctx, tenantID, connErr) }()

	if result.Enqueued, err = s.sweep(ctx, conn); err != nil {
		return result, err
	}
	w := &worker{s: s, conn: conn, tenantID: tenantID, ctx: tenantCtx(ctx, tenantID)}
	empty := ""
	connErr = &empty
	for n := 0; ; n++ {
		if n >= maxPerTenantRun || !s.now().Before(deadline) {
			result.More = true
			return result, nil
		}
		rec, err := w.next()
		if err != nil {
			return result, err
		}
		if rec == nil {
			return result, nil
		}
		_, err = w.process(rec)
		result.Synced, result.Failed, result.Skipped = w.res.Synced, w.res.Failed, w.res.Skipped
		switch {
		case errors.Is(err, ErrReauthRequired):
			msg := "QuickBooks authorization expired or was revoked. Reconnect QuickBooks to resume syncing."
			connErr = &msg
			return result, nil
		case err != nil && qbo.IsRateLimited(err):
			result.More = true
			return result, nil
		}
	}
}

type outcome int

const (
	outcomeSynced outcome = iota + 1
	outcomeSkipped
)

// worker pushes one tenant's records within a lease.
type worker struct {
	s        *Service
	conn     *models.QuickBooksConnection
	tenantID uuid.UUID
	ctx      context.Context
	sess     *qbo.Session
	// done holds records already handled this pass (incl. failures), so a
	// failed record isn't picked again before its backoff is written.
	done map[uuid.UUID]bool
	// res counts records handled, dependencies synced inline included.
	res RunResult
}

func (w *worker) db() *gorm.DB { return w.s.db.WithContext(w.ctx) }

// next returns the next due record: customers first, then invoices, then
// payments, oldest first.
func (w *worker) next() (*models.QuickBooksSyncRecord, error) {
	var rec models.QuickBooksSyncRecord
	q := w.db().Where("tenant_id = ? AND status IN ? AND next_attempt_at <= ? AND attempts < ?",
		w.tenantID, []string{models.QuickBooksSyncPending, models.QuickBooksSyncError}, w.s.now(), MaxAttempts)
	if len(w.done) > 0 {
		ids := make([]uuid.UUID, 0, len(w.done))
		for id := range w.done {
			ids = append(ids, id)
		}
		q = q.Where("id NOT IN ?", ids)
	}
	err := q.Order("CASE entity WHEN 'customer' THEN 0 WHEN 'invoice' THEN 1 ELSE 2 END, created_at").First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

// session returns the cached session or a fresh one.
func (w *worker) session(force bool) (qbo.Session, error) {
	if w.sess != nil && !force {
		return *w.sess, nil
	}
	stale := ""
	if w.sess != nil {
		stale = w.sess.AccessToken
	}
	sess, err := w.s.session(w.ctx, w.tenantID, force, stale)
	if err != nil {
		return sess, err
	}
	w.sess = &sess
	return sess, nil
}

// call runs fn with a session, refreshing once on a 401.
func (w *worker) call(fn func(qbo.Session) error) error {
	sess, err := w.session(false)
	if err != nil {
		return err
	}
	err = fn(sess)
	if qbo.IsUnauthorized(err) {
		if sess, err = w.session(true); err != nil {
			return err
		}
		err = fn(sess)
	}
	return err
}

// process pushes one record and records the outcome on its row.
func (w *worker) process(rec *models.QuickBooksSyncRecord) (outcome, error) {
	if w.done == nil {
		w.done = map[uuid.UUID]bool{}
	}
	w.done[rec.ID] = true
	var out outcome
	var err error
	switch rec.Entity {
	case models.QuickBooksEntityCustomer:
		_, out, err = w.syncCustomer(rec)
	case models.QuickBooksEntityInvoice:
		_, out, err = w.syncInvoice(rec)
	case models.QuickBooksEntityPayment:
		out, err = w.syncPayment(rec)
	default:
		err = fmt.Errorf("unknown entity %q", rec.Entity)
	}
	if errors.Is(err, errLocalMissing) {
		out, err = outcomeSkipped, w.forget(rec)
	}
	switch {
	case errors.Is(err, ErrReauthRequired):
	case err != nil:
		w.fail(rec, err)
		if !qbo.IsRateLimited(err) {
			w.res.Failed++
		}
	case out == outcomeSkipped:
		w.res.Skipped++
	default:
		w.res.Synced++
	}
	return out, err
}

// forget handles a record deleted in ServicePro: never synced → drop the
// row; synced → leave QuickBooks as it is and stop syncing it.
func (w *worker) forget(rec *models.QuickBooksSyncRecord) error {
	q := w.db().Where("id = ? AND tenant_id = ?", rec.ID, w.tenantID)
	if rec.QBOID == nil {
		return q.Delete(&models.QuickBooksSyncRecord{}).Error
	}
	return q.Model(&models.QuickBooksSyncRecord{}).Updates(map[string]any{
		"status": models.QuickBooksSyncSynced, "attempts": 0, "last_error": nil, "local_updated_at": nil,
	}).Error
}

// isLocalProblem: errors only the user can fix (not worth fast retries).
func isLocalProblem(err error) bool {
	return errors.Is(err, errMissingItem) || errors.Is(err, errDependency)
}

// errLocalMissing: the local record is gone (deleted).
var errLocalMissing = errors.New("record no longer exists in ServicePro")

// userError turns an error into the message shown on the record.
func userError(err error) string {
	var apiErr *qbo.APIError
	switch {
	case errors.As(err, &apiErr):
		msg := apiErr.Message
		if apiErr.Detail != "" && apiErr.Detail != apiErr.Message {
			msg += ": " + apiErr.Detail
		}
		if apiErr.Code != "" {
			msg += " (QuickBooks error " + apiErr.Code + ")"
		}
		return truncate(msg, 1000)
	case isLocalProblem(err):
		return err.Error()
	default:
		return "Couldn't reach QuickBooks; will retry"
	}
}

// fail records a failed attempt with exponential backoff (1 min doubling,
// at most 12 h). Throttling and outages stay pending; other errors show as
// errors.
func (w *worker) fail(rec *models.QuickBooksSyncRecord, err error) {
	if errors.Is(err, ErrReauthRequired) {
		return // connection-level; the record waits unchanged
	}
	attempts := rec.Attempts + 1
	if qbo.IsRateLimited(err) {
		attempts = rec.Attempts
	}
	backoff := time.Minute << min(attempts, 10)
	backoff = min(backoff, 12*time.Hour)
	status := models.QuickBooksSyncError
	if (qbo.IsTransient(err) && !errors.Is(err, errMissingItem)) || errors.Is(err, errDependency) {
		status = models.QuickBooksSyncPending
	}
	msg := userError(err)
	if !isLocalProblem(err) {
		logging.Warn(w.ctx, "[QUICKBOOKS] Record sync failed", map[string]any{
			"tenant_id": w.tenantID, "entity": rec.Entity, "local_id": rec.LocalID, "attempts": attempts, "error": err.Error(),
		})
	}
	uerr := w.db().Model(&models.QuickBooksSyncRecord{}).Where("id = ? AND tenant_id = ?", rec.ID, w.tenantID).
		Updates(map[string]any{"status": status, "attempts": attempts, "next_attempt_at": w.s.now().Add(backoff), "last_error": msg}).Error
	if uerr != nil {
		logging.Error(w.ctx, "[QUICKBOOKS] Could not record sync failure", map[string]any{"tenant_id": w.tenantID, "error": uerr.Error()})
	}
}

// succeed stores the QuickBooks identity of a synced record.
func (w *worker) succeed(rec *models.QuickBooksSyncRecord, e *qbo.Entity, hash string, localUpdated *time.Time) error {
	now := w.s.now()
	updates := map[string]any{
		"status": models.QuickBooksSyncSynced, "qbo_id": e.ID, "sync_token": e.SyncToken, "attempts": 0,
		"request_id": nil, "last_error": nil, "last_synced_at": now, "payload_hash": hash, "next_attempt_at": now,
		"local_updated_at": localUpdated,
	}
	if err := w.db().Model(&models.QuickBooksSyncRecord{}).Where("id = ? AND tenant_id = ?", rec.ID, w.tenantID).Updates(updates).Error; err != nil {
		return err
	}
	rec.Status, rec.QBOID, rec.SyncToken, rec.RequestID = models.QuickBooksSyncSynced, &e.ID, &e.SyncToken, nil
	return nil
}

// markUnchanged marks a record whose payload didn't change as synced
// without calling QuickBooks.
func (w *worker) markUnchanged(rec *models.QuickBooksSyncRecord, localUpdated *time.Time) error {
	rec.Status = models.QuickBooksSyncSynced
	return w.db().Model(&models.QuickBooksSyncRecord{}).Where("id = ? AND tenant_id = ?", rec.ID, w.tenantID).
		Updates(map[string]any{"status": models.QuickBooksSyncSynced, "attempts": 0, "last_error": nil, "local_updated_at": localUpdated}).Error
}

// requestID returns the record's persisted Intuit requestid for a create,
// minting and saving one first. reused reports a previous attempt with the
// same id (it may have reached QuickBooks).
func (w *worker) requestID(rec *models.QuickBooksSyncRecord) (id string, reused bool, err error) {
	if rec.RequestID != nil && *rec.RequestID != "" {
		return *rec.RequestID, true, nil
	}
	id = w.s.newID()
	if err := w.db().Model(&models.QuickBooksSyncRecord{}).Where("id = ? AND tenant_id = ?", rec.ID, w.tenantID).
		Update("request_id", id).Error; err != nil {
		return "", false, err
	}
	rec.RequestID = &id
	return id, false, nil
}

// record loads (or creates, as pending) the map row of a dependency.
func (w *worker) record(entity string, localID uuid.UUID) (*models.QuickBooksSyncRecord, error) {
	err := w.db().Exec(`
		INSERT INTO quickbooks_sync_map (tenant_id, entity, local_id, status, next_attempt_at)
		VALUES (?, ?, ?, 'pending', ?)
		ON CONFLICT (tenant_id, entity, local_id) DO NOTHING`, w.tenantID, entity, localID, w.s.now()).Error
	if err != nil {
		return nil, err
	}
	var rec models.QuickBooksSyncRecord
	err = w.db().Where("tenant_id = ? AND entity = ? AND local_id = ?", w.tenantID, entity, localID).First(&rec).Error
	return &rec, err
}

// errDependency: a record this one needs (its customer or invoice) failed.
var errDependency = errors.New("waiting for a linked record that failed to sync (see its error)")

// dependency returns the QuickBooks ID of a record this one points at,
// syncing it first if needed (the lease makes this safe).
func (w *worker) dependency(entity string, localID uuid.UUID) (string, error) {
	rec, err := w.record(entity, localID)
	if err != nil {
		return "", err
	}
	if rec.Status == models.QuickBooksSyncSynced && rec.QBOID != nil {
		return *rec.QBOID, nil
	}
	if w.done[rec.ID] {
		return "", errDependency
	}
	if _, err := w.process(rec); err != nil {
		if errors.Is(err, ErrReauthRequired) || qbo.IsRateLimited(err) {
			return "", err
		}
		return "", errDependency
	}
	if rec.QBOID == nil {
		return "", errDependency
	}
	return *rec.QBOID, nil
}

// syncCustomer creates or updates a customer.
func (w *worker) syncCustomer(rec *models.QuickBooksSyncRecord) (string, outcome, error) {
	var c models.Customer
	if err := w.db().Where("id = ?", rec.LocalID).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, errLocalMissing
		}
		return "", 0, err
	}
	payload := customerPayload(&c)
	hash := payloadHash(payload)
	updated := c.UpdatedAt

	if rec.QBOID != nil {
		if rec.PayloadHash != nil && *rec.PayloadHash == hash {
			return *rec.QBOID, outcomeSkipped, w.markUnchanged(rec, &updated)
		}
		e, err := w.update(qbo.EntityKindCustomer, rec, func(s qbo.Session, cur qbo.Entity, reqID string) (*qbo.Entity, error) {
			e, err := w.s.client.UpdateCustomer(w.ctx, s, cur, payload, reqID)
			if qbo.IsDuplicateName(err) {
				keep := payload
				keep.DisplayName = "" // keep the name it has in QuickBooks
				return w.s.client.UpdateCustomer(w.ctx, s, cur, keep, reqID)
			}
			return e, err
		})
		if err != nil {
			return "", 0, err
		}
		return e.ID, outcomeSynced, w.succeed(rec, e, hash, &updated)
	}

	reqID, _, err := w.requestID(rec)
	if err != nil {
		return "", 0, err
	}
	var e *qbo.Entity
	err = w.call(func(s qbo.Session) error {
		// Link to a customer that already exists under this name (made by
		// the accountant, or by an attempt whose answer we lost).
		found, err := w.s.client.FindCustomerByDisplayName(w.ctx, s, payload.DisplayName)
		if err != nil || found != nil {
			e = found
			return err
		}
		e, err = w.s.client.CreateCustomer(w.ctx, s, payload, reqID)
		if qbo.IsDuplicateName(err) {
			// The name belongs to a vendor or employee.
			alt := payload
			alt.DisplayName = truncate(payload.DisplayName, maxDisplayName-12) + " (customer)"
			e, err = w.s.client.CreateCustomer(w.ctx, s, alt, reqID+"-c")
		}
		return err
	})
	if err != nil {
		return "", 0, err
	}
	return e.ID, outcomeSynced, w.succeed(rec, e, hash, &updated)
}

// update runs a sparse update with the stored SyncToken; when QuickBooks
// says it's stale (edited there), it re-reads the token and tries once more.
func (w *worker) update(kind string, rec *models.QuickBooksSyncRecord, fn func(qbo.Session, qbo.Entity, string) (*qbo.Entity, error)) (*qbo.Entity, error) {
	cur := qbo.Entity{ID: *rec.QBOID, SyncToken: deref(rec.SyncToken)}
	var e *qbo.Entity
	err := w.call(func(s qbo.Session) error {
		var err error
		e, err = fn(s, cur, w.s.newID())
		if qbo.IsStale(err) {
			fresh, gerr := w.s.client.Get(w.ctx, s, kind, cur.ID)
			if gerr != nil {
				return gerr
			}
			cur.SyncToken = fresh.SyncToken
			e, err = fn(s, cur, w.s.newID())
		}
		return err
	})
	return e, err
}

// errMissingItem: no item to put on invoice lines.
var errMissingItem = errors.New("choose an income account in Settings → Integrations → QuickBooks")

// itemID returns the configured item, else the company's default
// "Services" item (cached on the connection).
func (w *worker) itemID() (string, error) {
	if id := deref(w.conn.ItemID); id != "" {
		return id, nil
	}
	var found *qbo.Entity
	err := w.call(func(s qbo.Session) error {
		var err error
		found, err = w.s.client.FindItemByName(w.ctx, s, defaultItemName)
		return err
	})
	if err != nil {
		return "", err
	}
	if found == nil {
		return "", errMissingItem
	}
	if err := w.db().Model(&models.QuickBooksConnection{}).Where("tenant_id = ?", w.tenantID).
		Updates(map[string]any{"item_id": found.ID, "item_name": defaultItemName}).Error; err != nil {
		return "", err
	}
	w.conn.ItemID, w.conn.ItemName = &found.ID, strPtr(defaultItemName)
	return found.ID, nil
}

// syncInvoice creates or updates an invoice (and its customer first).
func (w *worker) syncInvoice(rec *models.QuickBooksSyncRecord) (string, outcome, error) {
	var inv models.Invoice
	if err := w.db().Where("id = ?", rec.LocalID).First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, errLocalMissing
		}
		return "", 0, err
	}
	var lines []models.InvoiceLine
	if err := w.db().Where("invoice_id = ?", inv.ID).Order("sort_order, created_at").Find(&lines).Error; err != nil {
		return "", 0, err
	}
	customerID, err := w.dependency(models.QuickBooksEntityCustomer, inv.CustomerID)
	if err != nil {
		return "", 0, err
	}
	item, err := w.itemID()
	if err != nil {
		return "", 0, err
	}
	payload := invoicePayload(&inv, lines, invoiceRefs{CustomerID: customerID, ItemID: item, TaxCodeID: deref(w.conn.TaxCodeID)})
	hash := payloadHash(payload)
	updated := inv.UpdatedAt

	if rec.QBOID != nil {
		if rec.PayloadHash != nil && *rec.PayloadHash == hash {
			return *rec.QBOID, outcomeSkipped, w.markUnchanged(rec, &updated)
		}
		e, err := w.update(qbo.EntityKindInvoice, rec, func(s qbo.Session, cur qbo.Entity, reqID string) (*qbo.Entity, error) {
			return w.s.client.UpdateInvoice(w.ctx, s, cur, payload, reqID)
		})
		if err != nil {
			return "", 0, err
		}
		return e.ID, outcomeSynced, w.succeed(rec, e, hash, &updated)
	}

	reqID, reused, err := w.requestID(rec)
	if err != nil {
		return "", 0, err
	}
	var e *qbo.Entity
	err = w.call(func(s qbo.Session) error {
		if reused {
			// A previous attempt may have created it; Intuit only replays a
			// requestid for a limited time, so look it up as well.
			found, err := w.s.client.FindInvoiceByDocNumber(w.ctx, s, payload.DocNumber)
			if err != nil || found != nil {
				e = found
				return err
			}
		}
		var err error
		e, err = w.s.client.CreateInvoice(w.ctx, s, payload, reqID)
		return err
	})
	if err != nil {
		return "", 0, err
	}
	return e.ID, outcomeSynced, w.succeed(rec, e, hash, &updated)
}

// syncPayment creates a payment applied to its (synced) invoice. Payments
// are created once and never updated.
func (w *worker) syncPayment(rec *models.QuickBooksSyncRecord) (outcome, error) {
	if rec.QBOID != nil {
		return outcomeSkipped, w.markUnchanged(rec, rec.LocalUpdatedAt)
	}
	var p models.InvoicePayment
	if err := w.db().Where("id = ?", rec.LocalID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errLocalMissing
		}
		return 0, err
	}
	// invoice_payments has no tenant_id: its invoice must be this tenant's.
	var inv models.Invoice
	if err := w.db().Where("id = ?", p.InvoiceID).First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errLocalMissing
		}
		return 0, err
	}
	invoiceID, err := w.dependency(models.QuickBooksEntityInvoice, inv.ID)
	if err != nil {
		return 0, err
	}
	customerID, err := w.dependency(models.QuickBooksEntityCustomer, inv.CustomerID)
	if err != nil {
		return 0, err
	}
	payload := paymentPayload(&p, inv.InvoiceNumber, customerID, invoiceID, deref(w.conn.DepositAccountID))
	reqID, _, err := w.requestID(rec)
	if err != nil {
		return 0, err
	}
	var e *qbo.Entity
	err = w.call(func(s qbo.Session) error {
		var err error
		e, err = w.s.client.CreatePayment(w.ctx, s, payload, reqID)
		return err
	})
	if err != nil {
		return 0, err
	}
	updated := p.UpdatedAt
	return outcomeSynced, w.succeed(rec, e, payloadHash(payload), &updated)
}

// Retry puts a failed record back in the queue now.
func (s *Service) Retry(ctx context.Context, tenantID, recordID uuid.UUID) error {
	res := s.db.WithContext(tenantCtx(ctx, tenantID)).Model(&models.QuickBooksSyncRecord{}).
		Where("id = ? AND tenant_id = ? AND status <> ?", recordID, tenantID, models.QuickBooksSyncSynced).
		Updates(map[string]any{"status": models.QuickBooksSyncPending, "attempts": 0, "next_attempt_at": s.now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// retryErrors re-queues every failed record of the tenant (after the
// mapping settings change).
func (s *Service) retryErrors(ctx context.Context, tenantID uuid.UUID) error {
	return s.db.WithContext(tenantCtx(ctx, tenantID)).Model(&models.QuickBooksSyncRecord{}).
		Where("tenant_id = ? AND status = ?", tenantID, models.QuickBooksSyncError).
		Updates(map[string]any{"status": models.QuickBooksSyncPending, "attempts": 0, "next_attempt_at": s.now()}).Error
}

// SyncNow runs a short sync for one tenant ("Sync now").
func (s *Service) SyncNow(ctx context.Context, tenantID uuid.UUID, budget time.Duration) (RunResult, error) {
	if _, err := s.connected(ctx, tenantID); err != nil {
		return RunResult{}, err
	}
	r, err := s.SyncTenant(ctx, tenantID, s.now().Add(budget))
	if err != nil {
		return r, err
	}
	r.Tenants = 1
	return r, nil
}

// joinLabel joins non-empty label parts.
func joinLabel(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}
