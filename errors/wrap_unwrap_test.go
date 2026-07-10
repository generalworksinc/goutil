package gw_errors

import (
	"errors"
	"testing"

	"github.com/morikuni/failure/v2"
)

var sentinel = errors.New("forbidden")

// Wrap 後も errors.Is / errors.As が透過すること（sentinel エラーの403変換等が依存）。
func TestWrapPreservesErrorsIs(t *testing.T) {
	if !errors.Is(Wrap(sentinel), sentinel) {
		t.Fatal("Wrap(err) は errors.Is を透過するべき")
	}
	// 二重 Wrap でも透過
	if !errors.Is(Wrap(Wrap(sentinel)), sentinel) {
		t.Fatal("Wrap(Wrap(err)) も errors.Is を透過するべき")
	}
	// params 付きでも透過
	if !errors.Is(Wrap(sentinel, "param1", 42), sentinel) {
		t.Fatal("Wrap(err, params...) も errors.Is を透過するべき")
	}
}

// 既存規約: code = 元エラー（CodeOf ベースの検出・復元）が維持されること。
func TestWrapKeepsCodeConvention(t *testing.T) {
	wrapped := Wrap(sentinel)
	if code := failure.CodeOf(wrapped); code != sentinel {
		t.Fatalf("CodeOf は元エラーを返すべき: got %v", code)
	}
	// 二重 Wrap しても code は変わらない（再ラップされない）
	if code := failure.CodeOf(Wrap(wrapped)); code != sentinel {
		t.Fatalf("二重 Wrap 後も CodeOf は元エラー: got %v", code)
	}
}

func TestWrapNilIsNil(t *testing.T) {
	if Wrap(nil) != nil {
		t.Fatal("Wrap(nil) は nil")
	}
}
