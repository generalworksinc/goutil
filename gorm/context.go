package gw_gorm

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type scopeContextKey struct{}

// ErrDBRequiredは、transaction対象のDBが指定されていない場合に返します。
var ErrDBRequired = errors.New("database is required")

// ContextCarrierは、context.Contextを保持するHTTP contextなどの最小インターフェースです。
// gw_gormは特定のWebフレームワークへ依存せず、gw_web.WebCtxなどがこの条件を満たします。
type ContextCarrier interface {
	Context() context.Context
	SetContext(context.Context)
}

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

// AttachScopeは、認証済みScopeをHTTP contextなどが保持するcontext.Contextへ設定します。
func AttachScope(target ContextCarrier, scope *Scope) {
	if target == nil {
		return
	}
	target.SetContext(WithScopeContext(target.Context(), scope))
}

// WithTxは、渡されたDBへcontextを一度だけ設定してtransactionを開始します。
// Scopeがなくてもtransaction自体は開始できますが、ガード対象モデルへの操作は
// UseTenantGuardによって拒否されます。全テナント操作では、呼び出し側が明示的に
// AllTenantsまたはWithoutTenantScopeを指定したDBを利用してください。
func WithTx(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	if db == nil {
		return ErrDBRequired
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}
	return db.Transaction(fn)
}

func contextFromDB(db *gorm.DB) context.Context {
	if db != nil && db.Statement != nil && db.Statement.Context != nil {
		return db.Statement.Context
	}
	return context.Background()
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
