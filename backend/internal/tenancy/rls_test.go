package tenancy_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/internal/tenancy/rlstest"
)

// rlsTenant is one tenant seeded through the owner connection.
type rlsTenant struct {
	id       uuid.UUID
	customer uuid.UUID
	taxRate  uuid.UUID
	ctx      context.Context
}

func seedRLSTenant(t *testing.T, owner *gorm.DB) *rlsTenant {
	t.Helper()
	tt := &rlsTenant{id: uuid.New(), customer: uuid.New(), taxRate: uuid.New()}
	short := tt.id.String()[:8]
	user := uuid.New()
	require.NoError(t, owner.Exec(`INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`,
		user, "owner-"+short+"@example.com").Error)
	require.NoError(t, owner.Exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, ?, ?, ?)`,
		tt.id, "Tenant "+short, "t-"+short, user).Error)
	require.NoError(t, owner.Exec(`INSERT INTO customers (id, tenant_id, first_name, last_name, email, phone_primary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip)
		 VALUES (?, ?, 'Pat', ?, ?, '555-555-5555', '1 Main St', 'Springfield', 'IL', '62701')`,
		tt.customer, tt.id, "Customer "+short, "pat-"+short+"@example.com").Error)
	require.NoError(t, owner.Exec(`INSERT INTO tax_rates (id, tenant_id, name, rate) VALUES (?, ?, ?, 0.05)`,
		tt.taxRate, tt.id, "Tax "+short).Error)
	tt.ctx = tenancy.WithTenant(context.Background(), tt.id)
	return tt
}

func customerIDs(t *testing.T, db *gorm.DB) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	// Raw SQL with no tenant filter: the plugin never rewrites raw SQL, so
	// only the RLS policies stand between the caller and other tenants.
	require.NoError(t, db.Raw(`SELECT id FROM customers`).Scan(&ids).Error)
	return ids
}

func TestRLS_PoliciesCoverDefaultTables(t *testing.T) {
	env := rlstest.Setup(t)

	var rlsTables []string
	require.NoError(t, env.Owner.Raw(`
		SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relrowsecurity`).Scan(&rlsTables).Error)

	var want []string
	for table := range tenancy.DefaultTables() {
		want = append(want, table)
	}
	sort.Strings(want)
	sort.Strings(rlsTables)
	assert.Equal(t, want, rlsTables, "migration 000007 must cover exactly tenancy.DefaultTables()")

	var shared []string
	require.NoError(t, env.Owner.Raw(`SELECT tablename FROM pg_policies WHERE policyname = 'tenant_shared_read'`).Scan(&shared).Error)
	for table, mode := range tenancy.DefaultTables() {
		assert.Equal(t, mode == tenancy.SharedRead, contains(shared, table), "shared-read policy on %s", table)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestRLS_RawSQLIsTenantScoped(t *testing.T) {
	env := rlstest.Setup(t)
	a, b := seedRLSTenant(t, env.Owner), seedRLSTenant(t, env.Owner)
	db := env.OpenApp(t, true)

	t.Run("no tenant in context sees nothing", func(t *testing.T) {
		assert.Empty(t, customerIDs(t, db.WithContext(context.Background())))
	})

	t.Run("tenant sees only its own rows", func(t *testing.T) {
		assert.Equal(t, []uuid.UUID{a.customer}, customerIDs(t, db.WithContext(a.ctx)))
		assert.Equal(t, []uuid.UUID{b.customer}, customerIDs(t, db.WithContext(b.ctx)))

		var n int64
		require.NoError(t, db.WithContext(a.ctx).Raw(`SELECT count(*) FROM customers WHERE id = ?`, b.customer).Scan(&n).Error)
		assert.Zero(t, n)
	})

	t.Run("tenant cannot update or delete another tenant's rows", func(t *testing.T) {
		res := db.WithContext(a.ctx).Exec(`UPDATE customers SET notes = 'touched'`)
		require.NoError(t, res.Error)
		assert.Equal(t, int64(1), res.RowsAffected, "only A's own customer")

		res = db.WithContext(a.ctx).Exec(`DELETE FROM customers WHERE id = ?`, b.customer)
		require.NoError(t, res.Error)
		assert.Zero(t, res.RowsAffected)

		var notes *string
		require.NoError(t, env.Owner.Raw(`SELECT notes FROM customers WHERE id = ?`, b.customer).Scan(&notes).Error)
		assert.Nil(t, notes, "B's customer is untouched")
	})

	t.Run("tenant cannot insert or move rows into another tenant", func(t *testing.T) {
		err := db.WithContext(a.ctx).Exec(`INSERT INTO customers (tenant_id, first_name, last_name, email, phone_primary,
				billing_address_street, billing_address_city, billing_address_state, billing_address_zip)
			VALUES (?, 'Eve', 'Intruder', 'eve@example.com', '555-555-5555', '1 Main St', 'Springfield', 'IL', '62701')`,
			b.id).Error
		assert.ErrorContains(t, err, "row-level security")

		err = db.WithContext(a.ctx).Exec(`UPDATE customers SET tenant_id = ? WHERE id = ?`, b.id, a.customer).Error
		assert.ErrorContains(t, err, "row-level security")
	})

	t.Run("system context bypasses", func(t *testing.T) {
		sys := tenancy.WithSystem(context.Background())
		assert.ElementsMatch(t, []uuid.UUID{a.customer, b.customer}, customerIDs(t, db.WithContext(sys)))
	})
}

func TestRLS_SharedReadTables(t *testing.T) {
	env := rlstest.Setup(t)
	a, b := seedRLSTenant(t, env.Owner), seedRLSTenant(t, env.Owner)
	global := uuid.New()
	require.NoError(t, env.Owner.Exec(`INSERT INTO tax_rates (id, name, rate) VALUES (?, ?, 0.07)`,
		global, "Global "+global.String()[:8]).Error)
	db := env.OpenApp(t, true)

	var ids []uuid.UUID
	require.NoError(t, db.WithContext(a.ctx).Raw(`SELECT id FROM tax_rates`).Scan(&ids).Error)
	assert.Contains(t, ids, a.taxRate)
	assert.Contains(t, ids, global, "global defaults are readable")
	assert.NotContains(t, ids, b.taxRate)

	res := db.WithContext(a.ctx).Exec(`UPDATE tax_rates SET rate = 0.5 WHERE id = ?`, global)
	require.NoError(t, res.Error)
	assert.Zero(t, res.RowsAffected, "a tenant can't change a global default")

	res = db.WithContext(tenancy.WithSystem(context.Background())).Exec(`UPDATE tax_rates SET rate = 0.08 WHERE id = ?`, global)
	require.NoError(t, res.Error)
	assert.Equal(t, int64(1), res.RowsAffected, "system code can")
}

// Without the GORM plugin, an unfiltered model query is still scoped.
func TestRLS_GormWithoutPlugin(t *testing.T) {
	env := rlstest.Setup(t)
	a, b := seedRLSTenant(t, env.Owner), seedRLSTenant(t, env.Owner)
	db := env.OpenApp(t, false)

	var customers []models.Customer
	require.NoError(t, db.WithContext(a.ctx).Find(&customers).Error)
	require.Len(t, customers, 1)
	assert.Equal(t, a.customer, customers[0].ID)

	var c models.Customer
	err := db.WithContext(a.ctx).First(&c, "id = ?", b.customer).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var n int64
	require.NoError(t, db.WithContext(a.ctx).Model(&models.Customer{}).Count(&n).Error)
	assert.Equal(t, int64(1), n)

	// Row() keeps the statement transaction open until the row is scanned.
	var id uuid.UUID
	require.NoError(t, db.WithContext(b.ctx).Raw(`SELECT id FROM customers`).Row().Scan(&id))
	assert.Equal(t, b.customer, id)
}

func currentSetting(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	var v *string
	require.NoError(t, db.Raw(`SELECT current_setting(?, true)`, name).Scan(&v).Error)
	if v == nil {
		return ""
	}
	return *v
}

func TestRLS_Transactions(t *testing.T) {
	env := rlstest.Setup(t)
	a, b := seedRLSTenant(t, env.Owner), seedRLSTenant(t, env.Owner)
	db := env.OpenApp(t, true)
	sys := tenancy.WithSystem(context.Background())

	t.Run("transaction carries the tenant, nested savepoints too", func(t *testing.T) {
		err := db.WithContext(a.ctx).Transaction(func(tx *gorm.DB) error {
			assert.Equal(t, a.id.String(), currentSetting(t, tx, tenancy.SettingTenantID))
			assert.Equal(t, []uuid.UUID{a.customer}, customerIDs(t, tx))
			return tx.Transaction(func(inner *gorm.DB) error {
				assert.Equal(t, []uuid.UUID{a.customer}, customerIDs(t, inner))
				return nil
			})
		})
		require.NoError(t, err)
	})

	t.Run("system transaction narrowed to a tenant", func(t *testing.T) {
		err := db.WithContext(sys).Transaction(func(tx *gorm.DB) error {
			assert.Len(t, customerIDs(t, tx), 2)
			assert.Equal(t, []uuid.UUID{b.customer}, customerIDs(t, tx.WithContext(b.ctx)))
			assert.Len(t, customerIDs(t, tx.WithContext(sys)), 2)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("scope re-applied after rolling back to a savepoint", func(t *testing.T) {
		err := db.WithContext(sys).Transaction(func(tx *gorm.DB) error {
			require.NoError(t, tx.SavePoint("sp").Error) // under the system scope
			assert.Equal(t, []uuid.UUID{a.customer}, customerIDs(t, tx.WithContext(a.ctx)))
			// Reverts the server to the system settings while the statement
			// context (and the driver's idea of the scope) is still A's.
			require.NoError(t, tx.WithContext(a.ctx).RollbackTo("sp").Error)
			assert.Equal(t, []uuid.UUID{a.customer}, customerIDs(t, tx.WithContext(a.ctx)))
			return nil
		})
		require.NoError(t, err)

		errInner := errors.New("inner fails")
		err = db.WithContext(sys).Transaction(func(tx *gorm.DB) error {
			innerErr := tx.Transaction(func(inner *gorm.DB) error {
				assert.Equal(t, []uuid.UUID{a.customer}, customerIDs(t, inner.WithContext(a.ctx)))
				return errInner
			})
			require.ErrorIs(t, innerErr, errInner)
			assert.Len(t, customerIDs(t, tx), 2)
			assert.Equal(t, []uuid.UUID{a.customer}, customerIDs(t, tx.WithContext(a.ctx)))
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("failed savepoint under another scope rolls back cleanly", func(t *testing.T) {
		err := db.WithContext(sys).Transaction(func(tx *gorm.DB) error {
			innerErr := tx.WithContext(a.ctx).Transaction(func(inner *gorm.DB) error {
				// Fails under B's scope, aborting the transaction; the
				// ROLLBACK TO SAVEPOINT then runs under A's.
				return inner.WithContext(b.ctx).Exec(`SELECT 1/0`).Error
			})
			require.Error(t, innerErr)
			assert.Len(t, customerIDs(t, tx), 2)
			assert.Equal(t, []uuid.UUID{b.customer}, customerIDs(t, tx.WithContext(b.ctx)))
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("settings never leak to the next statement", func(t *testing.T) {
		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.SetMaxOpenConns(1) // force reuse of the same connection
		defer sqlDB.SetMaxOpenConns(4)

		assert.NotEmpty(t, customerIDs(t, db.WithContext(sys)))
		assert.Equal(t, "", currentSetting(t, db.WithContext(context.Background()), tenancy.SettingBypassRLS))
		assert.Empty(t, customerIDs(t, db.WithContext(context.Background())))
		require.NoError(t, db.WithContext(a.ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Exec(`UPDATE customers SET notes = 'x'`).Error
		}))
		assert.Equal(t, "", currentSetting(t, db.WithContext(context.Background()), tenancy.SettingTenantID))
		assert.Empty(t, customerIDs(t, db.WithContext(context.Background())))
	})
}
