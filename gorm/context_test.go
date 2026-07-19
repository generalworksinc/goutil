package gw_gorm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestScopeContextCopiesScope(t *testing.T) {
	scope := &Scope{TenantIds: []string{"tenant-a"}, OrgIds: []string{"org-a"}}
	ctx := WithScopeContext(context.Background(), scope)
	scope.TenantIds[0] = "changed-tenant"
	scope.OrgIds[0] = "changed-org"

	actual, ok := scopeFromContext(ctx)
	if !ok || !actual.CanSeeTenant("tenant-a") || !actual.CanSeeOrg("org-a") {
		t.Fatalf("scope=%+v ok=%v", actual, ok)
	}
	actual.TenantIds[0] = "mutated-return-value"
	again, ok := scopeFromContext(ctx)
	if !ok || !again.CanSeeTenant("tenant-a") {
		t.Fatalf("stored scope was mutated through return value: %+v", again)
	}
}

func TestScopeContextHandlesNilInputs(t *testing.T) {
	ctx := WithScopeContext(nil, nil)
	if ctx == nil {
		t.Fatal("nil context must be normalized")
	}
	if scope, ok := scopeFromContext(ctx); ok || scope != nil {
		t.Fatalf("scope=%+v ok=%v", scope, ok)
	}
	if scope, ok := scopeFromContext(nil); ok || scope != nil {
		t.Fatalf("nil context scope=%+v ok=%v", scope, ok)
	}
}

type testContextCarrier struct {
	ctx context.Context
}

func (c *testContextCarrier) Context() context.Context       { return c.ctx }
func (c *testContextCarrier) SetContext(ctx context.Context) { c.ctx = ctx }

func TestAttachScopeSupportsContextCarrier(t *testing.T) {
	carrier := &testContextCarrier{ctx: context.Background()}
	AttachScope(carrier, singleScope())
	actual, ok := scopeFromContext(carrier.Context())
	if !ok || !actual.CanSeeTenant("t1") || !actual.CanSeeOrg("o1") {
		t.Fatalf("scope=%+v ok=%v", actual, ok)
	}
	AttachScope(nil, singleScope())
}

func TestWithTxFailsClosedWithoutScope(t *testing.T) {
	db := openTransactionTestDB(t)
	err := WithTx(context.Background(), db, func(tx *gorm.DB) error {
		return tx.Create(&guardedTodo{Id: "without-scope", TenantId: "t1", OrganizationId: "o1"}).Error
	})
	if err == nil || !strings.Contains(err.Error(), "tenant scope is required") {
		t.Fatalf("expected tenant guard error, got %v", err)
	}
}

func TestWithTxPreservesScope(t *testing.T) {
	db := openTransactionTestDB(t)
	ctx := WithScopeContext(context.Background(), singleScope())
	err := WithTx(ctx, db, func(tx *gorm.DB) error {
		actual, ok := scopeFrom(tx)
		if !ok || !actual.CanSeeTenant("t1") || !actual.CanSeeOrg("o1") {
			t.Fatalf("transaction scope=%+v ok=%v", actual, ok)
		}
		return tx.Create(&guardedTodo{Id: "with-scope", TenantId: "t1", OrganizationId: "o1"}).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := WithScope(db, singleScope()).Model(&guardedTodo{}).Where("id = ?", "with-scope").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("committed row count=%d", count)
	}
}

func TestWithTxValidatesDBAndPropagatesCallbackError(t *testing.T) {
	if err := WithTx(context.Background(), nil, func(*gorm.DB) error { return nil }); !errors.Is(err, ErrDBRequired) {
		t.Fatalf("expected ErrDBRequired, got %v", err)
	}

	db := openTransactionTestDB(t)
	want := errors.New("callback failed")
	ctx := WithScopeContext(context.Background(), singleScope())
	err := WithTx(ctx, db, func(tx *gorm.DB) error {
		if err := tx.Create(&guardedTodo{Id: "rolled-back", TenantId: "t1", OrganizationId: "o1"}).Error; err != nil {
			return err
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("callback error=%v", err)
	}
	var count int64
	if err := WithScope(db, singleScope()).Model(&guardedTodo{}).Where("id = ?", "rolled-back").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("transaction must roll back, count=%d", count)
	}
}

func openTransactionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := UseTenantGuard(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&guardedTodo{}); err != nil {
		t.Fatal(err)
	}
	return db
}
