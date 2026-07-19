package gw_gorm

import (
	"context"

	"gorm.io/gorm"
)

type scopeContextKey struct{}

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

// scopeFromContextは、WithScopeContextで保存したScopeをTenant Guard内部へ返します。
// Scopeは認可の適用機構が利用する情報であり、ControllerやRepositoryへ公開しません。
func scopeFromContext(ctx context.Context) (*Scope, bool) {
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
// 最小インターフェースだけを受け取り、gw_webなど特定のWeb frameworkへ依存しません。
func AttachScope(target interface {
	Context() context.Context
	SetContext(context.Context)
}, scope *Scope) {
	if target == nil {
		return
	}
	target.SetContext(WithScopeContext(target.Context(), scope))
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
