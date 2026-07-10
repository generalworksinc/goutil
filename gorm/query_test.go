package gw_gorm

import (
	"strings"
	"testing"
)

// DryRun（DummyDialector）では行が返らないため、ここで検証できるのは
// 「不在は (nil, nil)」と「ガード拒否等のエラーは素通しされる」の2経路。
// ヒット時の挙動（RowsAffected>0 で実体が返る）は実DBを使うテンプレートのE2Eで担保する。
func TestFindOneNotFoundIsNotError(t *testing.T) {
	db := openTestDB(t)
	scoped := WithScope(db, singleScope())

	got, err := FindOne[guardedTodo](scoped.Where("id = ?", "missing"))
	if err != nil {
		t.Fatalf("not found must not be an error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil entity, got %+v", got)
	}
}

func TestFindOnePropagatesUnexpectedError(t *testing.T) {
	db := openTestDB(t)
	// スコープ未設定でガード対象モデルを検索 → "tenant scope is required" がそのまま返る
	got, err := FindOne[guardedTodo](db.Where("id = ?", "x"))
	if err == nil || !strings.Contains(err.Error(), "tenant scope is required") {
		t.Fatalf("expected guard error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil entity on error, got %+v", got)
	}
}
