package tenancy

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type widget struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key"`
	TenantID uuid.UUID `gorm:"type:uuid"`
	Name     string
}

func (widget) TableName() string { return "widgets" }

type term struct {
	ID       uuid.UUID  `gorm:"type:uuid;primary_key"`
	TenantID *uuid.UUID `gorm:"type:uuid"`
	Name     string
}

func (term) TableName() string { return "terms" }

func setup(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&widget{}, &term{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Use(New(map[string]Mode{"widgets": Owned, "terms": SharedRead})); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPluginScopesOwnedTable(t *testing.T) {
	db := setup(t)
	a, b := uuid.New(), uuid.New()
	ctxA, ctxB := WithTenant(context.Background(), a), WithTenant(context.Background(), b)

	wa := widget{ID: uuid.New(), Name: "a"}
	if err := db.WithContext(ctxA).Create(&wa).Error; err != nil {
		t.Fatal(err)
	}
	if wa.TenantID != a {
		t.Fatalf("tenant not assigned on create: %v", wa.TenantID)
	}
	if err := db.WithContext(ctxB).Create(&widget{ID: uuid.New(), Name: "b"}).Error; err != nil {
		t.Fatal(err)
	}

	// Cross-tenant create is refused.
	if err := db.WithContext(ctxB).Create(&widget{ID: uuid.New(), TenantID: a}).Error; !errors.Is(err, ErrTenantMismatch) {
		t.Fatalf("want mismatch, got %v", err)
	}

	var got []widget
	// A caller OR must not escape the tenant filter.
	if err := db.WithContext(ctxB).Where("name = ? OR name = ?", "a", "b").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "b" {
		t.Fatalf("tenant B saw %+v", got)
	}

	var one widget
	if err := db.WithContext(ctxB).First(&one, "id = ?", wa.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-tenant get: %v", err)
	}
	res := db.WithContext(ctxB).Model(&widget{}).Where("id = ?", wa.ID).Update("name", "pwned")
	if res.Error != nil || res.RowsAffected != 0 {
		t.Fatalf("cross-tenant update affected %d (%v)", res.RowsAffected, res.Error)
	}
	res = db.WithContext(ctxB).Delete(&widget{}, "id = ?", wa.ID)
	if res.Error != nil || res.RowsAffected != 0 {
		t.Fatalf("cross-tenant delete affected %d (%v)", res.RowsAffected, res.Error)
	}
	var n int64
	db.WithContext(ctxA).Table("widgets").Count(&n)
	if n != 1 {
		t.Fatalf("Table() count for A = %d", n)
	}

	// Moving a row to another tenant is refused.
	if err := db.WithContext(ctxA).Model(&widget{}).Where("id = ?", wa.ID).Updates(map[string]interface{}{"tenant_id": b}).Error; !errors.Is(err, ErrTenantMismatch) {
		t.Fatalf("want mismatch, got %v", err)
	}

	// Fail closed without tenant; system context is allowed.
	if err := db.Find(&got).Error; !errors.Is(err, ErrNoTenant) {
		t.Fatalf("want ErrNoTenant, got %v", err)
	}
	if err := db.WithContext(WithSystem(context.Background())).Find(&got).Error; err != nil || len(got) != 2 {
		t.Fatalf("system read: %v %d", err, len(got))
	}
}

func TestPluginSharedRead(t *testing.T) {
	db := setup(t)
	a, b := uuid.New(), uuid.New()
	sys := WithSystem(context.Background())
	if err := db.WithContext(sys).Create(&term{ID: uuid.New(), Name: "global"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(WithTenant(context.Background(), a)).Create(&term{ID: uuid.New(), Name: "a"}).Error; err != nil {
		t.Fatal(err)
	}
	var got []term
	if err := db.WithContext(WithTenant(context.Background(), b)).Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "global" {
		t.Fatalf("tenant B saw %+v", got)
	}
	// Global rows are read-only for tenants.
	res := db.WithContext(WithTenant(context.Background(), b)).Model(&term{}).Where("name = ?", "global").Update("name", "x")
	if res.Error != nil || res.RowsAffected != 0 {
		t.Fatalf("tenant updated global row: %d %v", res.RowsAffected, res.Error)
	}
}
