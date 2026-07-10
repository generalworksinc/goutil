// Package gw_authz は casbin による「行為の可否」判定を提供する。
//
// 責務境界（重要）:
//   - 行為の可否（このロールはこの行為をしてよいか）→ このパッケージ
//   - 可視範囲（どの行が見えるか。tenant/organization の絞り込み）→ gw_gorm のテナントガード
//
// 可視範囲の計算・フィルタを casbin に入れてはならない（ポリシーと組織ツリーの二重管理になる）。
// 同じ理由で、ユーザー→ロールの割当ても casbin に持たせない（アプリの User.Role 等が真実の源）。
// g ルールはロール階層（上位ロールが下位の許可を継承）のみに使う。
package gw_authz

import (
	"errors"
	"net/http"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	gw_web "github.com/generalworksinc/goutil/webframework"
	"gorm.io/gorm"
)

var (
	enforcer     *casbin.SyncedEnforcer
	roleResolver func(c *gw_web.WebCtx) (role string, tenantId string, ok bool)
)

type Option func(*options)

type options struct {
	modelText string
}

// WithModel はデフォルトの RBAC with domains モデルを差し替える。
func WithModel(text string) Option {
	return func(o *options) { o.modelText = text }
}

// Init は casbin_rule テーブルからポリシーをロードして enforcer を初期化する。
// アプリ起動時（DB 初期化後）に1回呼ぶ。以後の判定はインメモリで行われ、
// AddPolicy / RemovePolicy による変更は DB へ自動保存される。
func Init(db *gorm.DB, opts ...Option) error {
	o := &options{modelText: defaultModelText}
	for _, opt := range opts {
		opt(o)
	}
	m, err := casbinmodel.NewModelFromString(o.modelText)
	if err != nil {
		return err
	}
	e, err := casbin.NewSyncedEnforcer(m, newGormAdapter(db))
	if err != nil {
		return err
	}
	enforcer = e
	return nil
}

// SetRoleResolver はリクエストコンテキストから（ロール名, テナントID）を取り出す関数を登録する。
// Require ミドルウェアが使用する。アプリ初期化時に1回登録すること。
func SetRoleResolver(fn func(c *gw_web.WebCtx) (role string, tenantId string, ok bool)) {
	roleResolver = fn
}

// Can はロール role が dom（テナントID）で obj に対する act を行えるかを判定する。
// ポリシーにマッチしなければ false（デフォルト拒否）。Init 前は常に false。
func Can(role, dom, obj, act string) bool {
	if enforcer == nil {
		return false
	}
	ok, err := enforcer.Enforce(role, dom, obj, act)
	return err == nil && ok
}

// Require はルート/グループ用のミドルウェア。SetRoleResolver で取り出したロールが
// obj/act を許可されていなければ 403 で止める。リソース実体に依存しない行為
// （create や管理系 API）のゲートに使う。実体依存の判定（own/any）は FindEditable を使うこと。
func Require(obj, act string) gw_web.WebHandler {
	return func(c *gw_web.WebCtx) error {
		if roleResolver == nil {
			return c.Status(http.StatusInternalServerError).SendString("authz role resolver is not configured")
		}
		role, tenantId, ok := roleResolver(c)
		if !ok || !Can(role, tenantId, obj, act) {
			return c.Status(http.StatusForbidden).SendString("Forbidden")
		}
		return c.Next()
	}
}

// AddPolicy は p ルール（role, dom, obj, act）を追加する（DB へ自動保存・重複は無視）。
func AddPolicy(role, dom, obj, act string) error {
	if enforcer == nil {
		return errAuthzNotInitialized
	}
	_, err := enforcer.AddPolicy(role, dom, obj, act)
	return err
}

// RemovePolicy は p ルールを削除する。
func RemovePolicy(role, dom, obj, act string) error {
	if enforcer == nil {
		return errAuthzNotInitialized
	}
	_, err := enforcer.RemovePolicy(role, dom, obj, act)
	return err
}

// AddRoleInheritance は g ルール（child が parent の許可を継承）を追加する。
func AddRoleInheritance(child, parent string) error {
	if enforcer == nil {
		return errAuthzNotInitialized
	}
	_, err := enforcer.AddGroupingPolicy(child, parent)
	return err
}

// Enforcer は高度な用途（フィルタ付き削除等）のために生の enforcer を返す。
func Enforcer() *casbin.SyncedEnforcer { return enforcer }

var errAuthzNotInitialized = errors.New("gw_authz: not initialized (call Init first)")
