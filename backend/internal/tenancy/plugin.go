package tenancy

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Mode says how a table is scoped.
type Mode int

const (
	// Owned rows always belong to exactly one tenant. Reads, updates and
	// deletes are filtered by tenant_id = <caller>; creates get the caller's
	// tenant_id.
	Owned Mode = iota + 1
	// SharedRead tables mix tenant rows with global defaults (tenant_id IS
	// NULL), e.g. payment terms and tax rates. Reads see the caller's rows
	// plus global rows; updates and deletes only touch the caller's rows.
	SharedRead
)

const column = "tenant_id"

// Plugin is a GORM plugin that scopes every query on registered tables to the
// tenant carried by the statement's context (see WithTenant).
//
// It fails closed: a query on a registered table whose context has neither a
// tenant nor the system marker returns ErrNoTenant. Raw SQL (db.Raw/db.Exec)
// is not rewritten; callers must filter tenant_id themselves.
type Plugin struct {
	tables map[string]Mode
}

// New returns a plugin for the given table→mode map.
func New(tables map[string]Mode) *Plugin {
	return &Plugin{tables: tables}
}

// Name implements gorm.Plugin.
func (p *Plugin) Name() string { return "tenancy" }

// Initialize implements gorm.Plugin.
func (p *Plugin) Initialize(db *gorm.DB) error {
	cb := db.Callback()
	if err := cb.Query().Before("gorm:query").Register("tenancy:query", p.scopeRead); err != nil {
		return err
	}
	if err := cb.Row().Before("gorm:row").Register("tenancy:row", p.scopeRead); err != nil {
		return err
	}
	if err := cb.Update().Before("gorm:update").Register("tenancy:update", p.scopeWrite); err != nil {
		return err
	}
	if err := cb.Delete().Before("gorm:delete").Register("tenancy:delete", p.scopeWrite); err != nil {
		return err
	}
	return cb.Create().Before("gorm:create").Register("tenancy:create", p.assignOnCreate)
}

// Mode returns how a table is scoped (0 when it isn't).
func (p *Plugin) Mode(table string) Mode { return p.tables[table] }

// target returns the base table of the statement and the name/alias to
// qualify the tenant column with.
func target(stmt *gorm.Statement) (base, qualifier string) {
	if stmt.TableExpr != nil && stmt.TableExpr.SQL != "" {
		fields := strings.Fields(stmt.TableExpr.SQL)
		if len(fields) > 0 {
			base = strings.Trim(fields[0], "`\"")
		}
		qualifier = stmt.Table
		if qualifier == "" {
			qualifier = base
		}
		return base, qualifier
	}
	if stmt.Table != "" {
		return stmt.Table, stmt.Table
	}
	if stmt.Schema != nil {
		return stmt.Schema.Table, stmt.Schema.Table
	}
	return "", ""
}

// resolve returns the mode for the statement's table and the caller's tenant.
// ok=false means "leave the statement alone".
func (p *Plugin) resolve(db *gorm.DB) (mode Mode, qualifier string, tenantID uuid.UUID, ok bool) {
	stmt := db.Statement
	if stmt.SQL.Len() > 0 { // raw SQL: not rewritten
		return 0, "", uuid.Nil, false
	}
	base, qualifier := target(stmt)
	mode = p.tables[base]
	if mode == 0 {
		return 0, "", uuid.Nil, false
	}
	ctx := stmt.Context
	if tid, has := TenantID(ctx); has {
		return mode, qualifier, tid, true
	}
	if IsSystem(ctx) {
		return 0, "", uuid.Nil, false
	}
	_ = db.AddError(fmt.Errorf("%w (table %s)", ErrNoTenant, base))
	return 0, "", uuid.Nil, false
}

func (p *Plugin) condition(db *gorm.DB, mode Mode, qualifier string, tenantID uuid.UUID, write bool) clause.Expression {
	col := db.Statement.Quote(clause.Column{Table: qualifier, Name: column})
	if mode == SharedRead && !write {
		return clause.Expr{SQL: "(" + col + " = ? OR " + col + " IS NULL)", Vars: []interface{}{tenantID}}
	}
	return clause.Expr{SQL: col + " = ?", Vars: []interface{}{tenantID}}
}

// addCondition ANDs cond with whatever WHERE the statement already has. The
// existing conditions are grouped first so a caller's OR can't escape the
// tenant filter ("a OR b AND tenant" would).
func addCondition(stmt *gorm.Statement, cond clause.Expression) {
	c, ok := stmt.Clauses["WHERE"]
	if !ok {
		stmt.AddClause(clause.Where{Exprs: []clause.Expression{cond}})
		return
	}
	where, _ := c.Expression.(clause.Where)
	exprs := []clause.Expression{cond}
	if len(where.Exprs) > 0 {
		// Always parenthesise the caller's conditions: GORM does not wrap a
		// single raw "a OR b" string, which would otherwise bind as
		// "a OR (b AND tenant)".
		existing := clause.Expr{SQL: "(?)", Vars: []interface{}{clause.AndConditions(where)}}
		exprs = []clause.Expression{existing, cond}
	}
	c.Expression = clause.Where{Exprs: exprs}
	stmt.Clauses["WHERE"] = c
}

func (p *Plugin) scopeRead(db *gorm.DB) {
	if db.Error != nil {
		return
	}
	mode, q, tid, ok := p.resolve(db)
	if !ok {
		return
	}
	addCondition(db.Statement, p.condition(db, mode, q, tid, false))
}

func (p *Plugin) scopeWrite(db *gorm.DB) {
	if db.Error != nil {
		return
	}
	mode, q, tid, ok := p.resolve(db)
	if !ok {
		return
	}
	if err := checkNoTenantChange(db, tid); err != nil {
		_ = db.AddError(err)
		return
	}
	addCondition(db.Statement, p.condition(db, mode, q, tid, true))
}

// checkNoTenantChange rejects updates that would move a row to another tenant.
func checkNoTenantChange(db *gorm.DB, tid uuid.UUID) error {
	switch dest := db.Statement.Dest.(type) {
	case map[string]interface{}:
		if v, ok := dest[column]; ok && !sameTenant(v, tid) {
			return ErrTenantMismatch
		}
		return nil
	}
	if db.Statement.Schema == nil {
		return nil
	}
	field := db.Statement.Schema.LookUpField(column)
	if field == nil {
		return nil
	}
	rv := reflect.Indirect(reflect.ValueOf(db.Statement.Dest))
	if rv.Kind() != reflect.Struct || rv.Type() != db.Statement.Schema.ModelType {
		return nil
	}
	v, zero := field.ValueOf(db.Statement.Context, rv)
	if zero {
		return nil
	}
	if !sameTenant(v, tid) {
		return ErrTenantMismatch
	}
	return nil
}

func sameTenant(v interface{}, tid uuid.UUID) bool {
	switch t := v.(type) {
	case uuid.UUID:
		return t == tid
	case *uuid.UUID:
		return t != nil && *t == tid
	case string:
		return t == tid.String()
	}
	return false
}

// assignOnCreate sets tenant_id on new rows of registered tables to the
// caller's tenant, refusing rows that name a different tenant.
func (p *Plugin) assignOnCreate(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil {
		return
	}
	stmt := db.Statement
	mode := p.tables[stmt.Schema.Table]
	if mode == 0 {
		return
	}
	field := stmt.Schema.LookUpField(column)
	if field == nil {
		return
	}
	ctx := stmt.Context
	tid, has := TenantID(ctx)
	if !has && !IsSystem(ctx) {
		_ = db.AddError(fmt.Errorf("%w (create %s)", ErrNoTenant, stmt.Schema.Table))
		return
	}

	apply := func(rv reflect.Value) {
		v, zero := field.ValueOf(ctx, rv)
		switch {
		case !has:
			// System code must name the tenant explicitly for owned rows.
			if zero && mode == Owned {
				_ = db.AddError(fmt.Errorf("%w (create %s without tenant_id)", ErrNoTenant, stmt.Schema.Table))
			}
		case zero:
			if err := setTenant(ctx, field, rv, tid); err != nil {
				_ = db.AddError(err)
			}
		case !sameTenant(v, tid):
			_ = db.AddError(ErrTenantMismatch)
		}
	}

	// An upsert (GORM's Save falls back to INSERT ... ON CONFLICT DO UPDATE)
	// must never overwrite a row that belongs to another tenant.
	if has {
		if c, ok := stmt.Clauses["ON CONFLICT"]; ok {
			if oc, ok := c.Expression.(clause.OnConflict); ok && !oc.DoNothing {
				col := stmt.Quote(clause.Column{Table: stmt.Schema.Table, Name: column})
				oc.Where.Exprs = append(oc.Where.Exprs, clause.Expr{SQL: col + " = ?", Vars: []interface{}{tid}})
				c.Expression = oc
				stmt.Clauses["ON CONFLICT"] = c
			}
		}
	}

	rv := stmt.ReflectValue
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			elem := reflect.Indirect(rv.Index(i))
			if elem.Kind() == reflect.Struct {
				apply(elem)
			}
		}
	case reflect.Struct:
		apply(rv)
	}
}

func setTenant(ctx context.Context, field *schema.Field, rv reflect.Value, tid uuid.UUID) error {
	if field.FieldType.Kind() == reflect.Pointer {
		id := tid
		return field.Set(ctx, rv, &id)
	}
	return field.Set(ctx, rv, tid)
}
