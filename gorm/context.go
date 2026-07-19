package gw_gorm

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type scopeContextKey struct{}

// ErrDefaultDBRequiredは、明示DBと既定DBの両方が指定されていない場合に返します。
var ErrDefaultDBRequired = errors.New("default database is required")

// WithScopeContextは、認証・認可で確定したScopeをcontextへ保存します。
// 呼び出し後に元のScopeが変更されても影響を受けないよう、値を複製して保存します。
func WithScopeContext(ctx context.Context, scope *Scope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if scope == nil {
		return ctx
	}
	return context.WithValue(ctx, scopeContextKey{}, cloneScope(scope))
}

// ScopeFromContextは、WithScopeContextで保存したScopeを返します。
func ScopeFromContext(ctx context.Context) (*Scope, bool) {
	if ctx == nil {
		return nil, false
	}
	scope, ok := ctx.Value(scopeContextKey{}).(*Scope)
	if !ok || scope == nil {
		return nil, false
	}
	return cloneScope(scope), true
}

// PickConnectionIfEmptyは、明示DBがあればそれを優先し、なければ既定DBを選択します。
// 既定DBを使う場合はcontext内のScopeをGORMセッションへ適用します。
// Scopeがなければ通常の既定DBを返し、ガード対象モデルの可否はUseTenantGuardが
// fail-closedで判断します。
func PickConnectionIfEmpty(ctx context.Context, dbAccess, defaultDB *gorm.DB) *gorm.DB {
	if dbAccess != nil {
		if ctx != nil {
			return dbAccess.WithContext(ctx)
		}
		return dbAccess
	}
	if defaultDB == nil {
		return nil
	}
	if scope, ok := ScopeFromContext(ctx); ok {
		db := WithScope(defaultDB, scope)
		if ctx != nil {
			db = db.WithContext(ctx)
		}
		return db
	}
	if ctx != nil {
		return defaultDB.WithContext(ctx)
	}
	return defaultDB
}

// WithTxは、context内のScopeを維持した既定DBからtransactionを開始します。
// Scopeがなくてもtransaction自体は開始できますが、ガード対象モデルへの操作は
// UseTenantGuardによって拒否されます。全テナント操作では、呼び出し側が明示的に
// AllTenantsまたはWithoutTenantScopeを指定したDBを利用してください。
func WithTx(ctx context.Context, defaultDB *gorm.DB, fn func(*gorm.DB) error) error {
	if defaultDB == nil {
		return ErrDefaultDBRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db := PickConnectionIfEmpty(ctx, nil, defaultDB)
	return db.Transaction(fn)
}

func cloneScope(scope *Scope) *Scope {
	if scope == nil {
		return nil
	}
	return &Scope{
		TenantIds:  append([]string(nil), scope.TenantIds...),
		OrgIds:     append([]string(nil), scope.OrgIds...),
		AllTenants: scope.AllTenants,
	}
}
