// Package gw_log は slog による構造化ロギングと、リクエストID/ユーザーIDの
// context 伝搬を提供する。Init 後は、どの層でも slog.InfoContext(ctx, ...) と
// 書くだけで request_id / user_id がログへ自動付与される（横ぐし検索の基盤）。
//
// 使い方:
//
//	gw_log.Init(slog.LevelInfo)                       // 起動時に1回
//	app.Use(gw_web.RequestId())                        // 最初のミドルウェアとして
//	slog.InfoContext(ctx, "something happened", ...)   // どこからでも
package gw_log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

type ctxKey int

const (
	requestIdKey ctxKey = iota
	userIdKey
)

// WithRequestId はリクエストIDを context に載せる。
func WithRequestId(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, requestIdKey, requestId)
}

// RequestIdFromContext は context からリクエストIDを取り出す（無ければ空文字）。
func RequestIdFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIdKey).(string)
	return v
}

// WithUserId は認証済みユーザーIDを context に載せる（認証ミドルウェアが呼ぶ）。
func WithUserId(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, userIdKey, userId)
}

// UserIdFromContext は context からユーザーIDを取り出す（無ければ空文字）。
func UserIdFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(userIdKey).(string)
	return v
}

// contextHandler は slog.Handler をラップし、Handle 時に context から
// request_id / user_id を読み取って属性として自動付与する。
// これにより slog.XxxContext 系の呼び出し全てに横ぐしキーが乗る。
//
// 横ぐしキーは WithGroup 適用後の logger からのログでも常に**トップレベル**に出る
// （グループ配下に沈むと検索フィールドが request_id と g.request_id に分散するため）。
// これを保証するため、グループ前の base ハンドラと WithAttrs/WithGroup の操作列を保持し、
// 注入時は base に横ぐしキーを付けてから操作列を再適用する。
type contextHandler struct {
	base       slog.Handler                      // WithAttrs/WithGroup 適用前のハンドラ
	buildSteps []func(slog.Handler) slog.Handler // 適用された WithAttrs/WithGroup の列
	assembled  slog.Handler                      // base に buildSteps を適用済みのハンドラ（注入不要時の高速パス）
}

func newContextHandler(base slog.Handler) contextHandler {
	return contextHandler{base: base, assembled: base}
}

func (h contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.assembled.Enabled(ctx, level)
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	var ids []slog.Attr
	if id := RequestIdFromContext(ctx); id != "" {
		ids = append(ids, slog.String("request_id", id))
	}
	if uid := UserIdFromContext(ctx); uid != "" {
		ids = append(ids, slog.String("user_id", uid))
	}
	if len(ids) == 0 {
		return h.assembled.Handle(ctx, r)
	}
	// base の直後（グループ適用前）に注入することでトップレベル属性を保証する
	hh := h.base.WithAttrs(ids)
	for _, op := range h.buildSteps {
		hh = op(hh)
	}
	return hh.Handle(ctx, r)
}

func (h contextHandler) with(op func(slog.Handler) slog.Handler) slog.Handler {
	buildSteps := make([]func(slog.Handler) slog.Handler, len(h.buildSteps), len(h.buildSteps)+1)
	copy(buildSteps, h.buildSteps)
	return contextHandler{base: h.base, buildSteps: append(buildSteps, op), assembled: op(h.assembled)}
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.with(func(hh slog.Handler) slog.Handler { return hh.WithAttrs(attrs) })
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return h.with(func(hh slog.Handler) slog.Handler { return hh.WithGroup(name) })
}

// Options は Init の調整用。
type Options struct {
	// Writer は出力先。nil なら os.Stdout（ローテートログ等の io.Writer も指定可）。
	Writer io.Writer
	// AddSource を true にすると発生箇所（file:line）を付与する。
	AddSource bool
}

// Init は JSON ハンドラ + context 注入ハンドラで slog のデフォルトロガーを設定する。
// 時刻は UTC の RFC3339Nano に正規化される。アプリ起動時に1回呼ぶこと。
func Init(level slog.Leveler, opts ...Options) *slog.Logger {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	var w io.Writer = os.Stdout
	if o.Writer != nil {
		w = o.Writer
	}
	base := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     level,
		AddSource: o.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.UTC().Format(time.RFC3339Nano))
				}
			}
			return a
		},
	})
	l := slog.New(newContextHandler(base))
	slog.SetDefault(l)
	return l
}

// NewHandler は任意の slog.Handler を context 注入ハンドラでラップして返す
// （独自の出力先やフォーマットを使いたい場合の低レベルAPI）。nil は破棄ハンドラ扱い。
func NewHandler(inner slog.Handler) slog.Handler {
	if inner == nil {
		inner = slog.DiscardHandler
	}
	return newContextHandler(inner)
}
