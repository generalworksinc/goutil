package gw_authz

import (
	"errors"

	gw_gorm "github.com/generalworksinc/goutil/gorm"
	"gorm.io/gorm"
)

// ErrForbidden は「対象は存在する（見えている）が、この行為の権限がない」ことを表す。
// 呼び出し側は 403 に変換する。不在（nil, nil）とは区別される。
var ErrForbidden = errors.New("forbidden")

// OwnedResource は own/any 判定に必要な情報をモデルが宣言するインターフェース。
// モデル側に1行メソッドを2つ実装する（gw_gorm のマーカーと同じ「宣言」の流儀）:
//
//	func (t Todo) OwnedBy() string    { return t.UserId }
//	func (t Todo) TenantIdOf() string { return t.TenantId }
type OwnedResource interface {
	OwnedBy() string
	TenantIdOf() string
}

// FindEditable は id で1件ロードし、行為権限を合成判定して返す。
//
//	現物, err := gw_authz.FindEditable[models.Todo](tx, scope.Role.Name(), userId, id, "todo", "update")
//
// 判定順:
//  1. 可視範囲はガードが解決済み（他テナントはそもそもヒットせず (nil, nil)）
//  2. 所有者かつ act+"_own" が許可 → OK
//  3. act+"_any" が許可（例: admin が他人のリソースを操作）→ OK
//  4. どちらでもない → ErrForbidden
func FindEditable[T any, PT interface {
	*T
	OwnedResource
}](tx *gorm.DB, role, actorId, id, obj, act string) (*T, error) {
	ent, err := gw_gorm.FindOne[T](tx.Where("id = ?", id))
	if err != nil || ent == nil {
		return ent, err
	}
	r := PT(ent)
	dom := r.TenantIdOf()
	if r.OwnedBy() == actorId && Can(role, dom, obj, act+"_own") {
		return ent, nil
	}
	if Can(role, dom, obj, act+"_any") {
		return ent, nil
	}
	return nil, ErrForbidden
}
