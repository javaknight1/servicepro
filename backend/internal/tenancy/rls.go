package tenancy

// Row-level security plumbing (T154).
//
// Migration 000007 puts Postgres RLS policies on every table in
// DefaultTables. The policies read two transaction-local settings:
//
//	app.tenant_id   the caller's tenant (rows of other tenants are invisible
//	                and can't be written)
//	app.bypass_rls  'on' for trusted system code (WithSystem): webhooks,
//	                internal jobs, public token routes
//
// RLSConnector wraps the pgx database/sql driver so that every statement
// whose context carries a tenant (WithTenant) or the system marker
// (WithSystem) runs inside a transaction that has those settings:
//
//   - database/sql transactions (GORM's db.Transaction, its default
//     write transactions) get the settings right after BEGIN, from the
//     context that began them; a later statement in the same transaction
//     whose context names a different scope re-applies them first.
//   - a statement outside a transaction is wrapped in its own
//     BEGIN / set settings / statement / COMMIT. For queries, the COMMIT
//     runs when the rows are closed.
//   - a statement whose context has neither runs untouched: it sees no
//     tenant-owned rows at all (fail closed).
//
// Only transaction-local settings (set_config(..., true)) are used, so the
// plumbing is safe behind a transaction-mode pooler (Neon's PgBouncer): the
// settings never outlive the transaction and never leak to another client.
//
// The policies only bind roles that are subject to RLS: not superusers, not
// roles with BYPASSRLS and not the tables' owner (the tables are not
// FORCE ROW LEVEL SECURITY). The API must connect as a member of
// servicepro_app for the backstop to take effect; migrations, backups and
// seeds keep running as the owner.

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
)

// Names of the settings the RLS policies read (see migration 000007).
const (
	SettingTenantID  = "app.tenant_id"
	SettingBypassRLS = "app.bypass_rls" //nolint:gosec // a setting name, not a credential
)

// rlsScope is what a statement's context asks the database to enforce.
type rlsScope struct {
	tenant uuid.UUID
	system bool
}

func (s rlsScope) none() bool { return s.tenant == uuid.Nil && !s.system }

// scopeOf derives the scope from ctx. A tenant wins over the system marker,
// matching the GORM plugin (WithTenant after WithSystem narrows the scope).
func scopeOf(ctx context.Context) rlsScope {
	if id, ok := TenantID(ctx); ok {
		return rlsScope{tenant: id}
	}
	if IsSystem(ctx) {
		return rlsScope{system: true}
	}
	return rlsScope{}
}

// setSQL sets both settings for the current transaction. The values are
// formatted into the statement (a UUID's text form is only hex digits and
// dashes) so that it has no parameters: pgx then sends it on the simple
// protocol, which lets it share a round trip with BEGIN.
func (s rlsScope) setSQL() string {
	tenant, bypass := "", "off"
	if s.system {
		bypass = "on"
	} else {
		tenant = s.tenant.String()
	}
	return fmt.Sprintf("SELECT set_config('%s', '%s', true), set_config('%s', '%s', true)",
		SettingTenantID, tenant, SettingBypassRLS, bypass)
}

// RLSConnector wraps a pgx stdlib connector (stdlib.GetConnector) so every
// connection applies the tenant settings described above.
func RLSConnector(base driver.Connector) driver.Connector {
	return &rlsConnector{base: base}
}

type rlsConnector struct {
	base driver.Connector
}

func (c *rlsConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	pc, ok := conn.(*stdlib.Conn)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("tenancy: RLSConnector needs a pgx stdlib connection, got %T", conn)
	}
	return &rlsConn{Conn: pc}, nil
}

func (c *rlsConnector) Driver() driver.Driver { return c.base.Driver() }

// rlsConn embeds the pgx connection, so every optional driver interface
// (NamedValueChecker, SessionResetter, Pinger, ...) still reaches pgx; only
// statement execution and transactions are intercepted. Prepared statements
// (PrepareContext) are not: nothing in the app prepares explicitly.
//
// database/sql never uses a driver connection from two goroutines at once,
// so the fields need no locking.
type rlsConn struct {
	*stdlib.Conn

	// applied is the scope set in the current transaction; valid only while
	// hasApplied is true.
	applied    rlsScope
	hasApplied bool
}

func (c *rlsConn) inTx() bool { return c.Conn.Conn().PgConn().TxStatus() != 'I' }

// apply sets s for the current transaction.
func (c *rlsConn) apply(ctx context.Context, s rlsScope) error {
	if _, err := c.Conn.ExecContext(ctx, s.setSQL(), nil); err != nil {
		c.hasApplied = false
		return fmt.Errorf("tenancy: set RLS scope: %w", err)
	}
	c.applied, c.hasApplied = s, true
	return nil
}

// ensure makes the open transaction match s (a no-op when it already does).
func (c *rlsConn) ensure(ctx context.Context, s rlsScope) error {
	if s.none() || (c.hasApplied && c.applied == s) {
		return nil
	}
	if c.Conn.Conn().PgConn().TxStatus() == 'E' {
		// Aborted transaction: only ROLLBACK [TO SAVEPOINT] can run, and
		// set_config would fail and block it. Re-apply afterwards.
		c.hasApplied = false
		return nil
	}
	return c.apply(ctx, s)
}

// begin opens a statement-level transaction with s applied, in one round
// trip.
func (c *rlsConn) begin(ctx context.Context, s rlsScope) error {
	if _, err := c.Conn.ExecContext(ctx, "BEGIN; "+s.setSQL(), nil); err != nil {
		_ = c.end(context.WithoutCancel(ctx), false)
		return fmt.Errorf("tenancy: begin RLS-scoped statement: %w", err)
	}
	c.applied, c.hasApplied = s, true
	return nil
}

// end closes a statement-level transaction. A failed statement leaves the
// transaction aborted, where COMMIT is a rollback anyway; the explicit
// ROLLBACK just says so.
func (c *rlsConn) end(ctx context.Context, commit bool) error {
	c.hasApplied = false
	if !c.inTx() {
		return nil
	}
	stmt := "ROLLBACK"
	if commit {
		stmt = "COMMIT"
	}
	_, err := c.Conn.ExecContext(ctx, stmt, nil)
	return err
}

// isSavepointRollback reports whether query rolls back to a savepoint, which
// also reverts any set_config made after that savepoint.
func isSavepointRollback(query string) bool {
	q := strings.TrimSpace(query)
	return len(q) >= len("ROLLBACK TO") && strings.EqualFold(q[:len("ROLLBACK TO")], "ROLLBACK TO")
}

func (c *rlsConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	s := scopeOf(ctx)
	if c.inTx() {
		if err := c.ensure(ctx, s); err != nil {
			return nil, err
		}
		res, err := c.Conn.ExecContext(ctx, query, args)
		if isSavepointRollback(query) {
			c.hasApplied = false // re-apply before the next scoped statement
		}
		return res, err
	}
	if s.none() {
		return c.Conn.ExecContext(ctx, query, args)
	}
	if err := c.begin(ctx, s); err != nil {
		return nil, err
	}
	res, err := c.Conn.ExecContext(ctx, query, args)
	if endErr := c.end(context.WithoutCancel(ctx), err == nil); err == nil && endErr != nil {
		return nil, endErr
	}
	return res, err
}

func (c *rlsConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	s := scopeOf(ctx)
	if c.inTx() {
		if err := c.ensure(ctx, s); err != nil {
			return nil, err
		}
		return c.Conn.QueryContext(ctx, query, args)
	}
	if s.none() {
		return c.Conn.QueryContext(ctx, query, args)
	}
	if err := c.begin(ctx, s); err != nil {
		return nil, err
	}
	rows, err := c.Conn.QueryContext(ctx, query, args)
	if err != nil {
		_ = c.end(context.WithoutCancel(ctx), false)
		return nil, err
	}
	pr, ok := rows.(*stdlib.Rows)
	if !ok {
		_ = rows.Close()
		_ = c.end(context.WithoutCancel(ctx), false)
		return nil, fmt.Errorf("tenancy: unexpected rows type %T", rows)
	}
	return &rlsRows{Rows: pr, conn: c, ctx: context.WithoutCancel(ctx)}, nil
}

// rlsRows commits the statement-level transaction when the rows are closed.
// Embedding *stdlib.Rows keeps the column-type interfaces database/sql uses
// for scanning.
type rlsRows struct {
	*stdlib.Rows
	conn   *rlsConn
	ctx    context.Context
	closed bool
}

func (r *rlsRows) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	err := r.Rows.Close()
	return errors.Join(err, r.conn.end(r.ctx, err == nil))
}

// Begin overrides the embedded pgx Begin, which would bypass BeginTx below.
func (c *rlsConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

// BeginTx begins a transaction and applies the scope of the context that
// began it (GORM passes the statement context to Begin).
func (c *rlsConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	c.hasApplied = false
	if s := scopeOf(ctx); !s.none() {
		if err := c.apply(ctx, s); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	return &rlsTx{Tx: tx, conn: c}, nil
}

type rlsTx struct {
	driver.Tx
	conn *rlsConn
}

func (t *rlsTx) Commit() error {
	t.conn.hasApplied = false
	return t.Tx.Commit()
}

func (t *rlsTx) Rollback() error {
	t.conn.hasApplied = false
	return t.Tx.Rollback()
}

// ResetSession runs when database/sql hands the connection to a new caller.
func (c *rlsConn) ResetSession(ctx context.Context) error {
	c.hasApplied = false
	return c.Conn.ResetSession(ctx)
}
