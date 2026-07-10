package gw_web

// このファイルはロギング系ミドルウェアのみを置く。fiber の wrapping
// （WebApp.Use / WebCtx.Context / SetContext 等）は webframework.go に集約する。

import (
	"log/slog"
	"time"

	gw_log "github.com/generalworksinc/goutil/logging"
	gw_uuid "github.com/generalworksinc/goutil/uuid"
)

const HeaderRequestId = "X-Request-Id"

// RequestId はリクエストIDを解決して context とレスポンスヘッダに載せるミドルウェア。
// 最初のミドルウェアとして登録すること（以降の全ログ・SQLに request_id が乗る前提になる）。
// 優先順位: X-Request-Id → X-Amzn-Trace-Id（ALB経由）→ ULID新規採番。
func RequestId() WebHandler {
	return func(ctx *WebCtx) error {
		rid := ctx.Get(HeaderRequestId)
		if rid == "" {
			rid = ctx.Get("X-Amzn-Trace-Id")
		}
		if rid == "" {
			rid = gw_uuid.GetUlid()
		}
		ctx.SetContext(gw_log.WithRequestId(ctx.Context(), rid))
		ctx.Set(HeaderRequestId, rid)
		return ctx.Next()
	}
}

// AccessLog はレスポンス完了後に1リクエスト1行のアクセスログを slog で出力するミドルウェア。
// request_id / user_id は gw_log の context 注入ハンドラが自動付与する。
// ハンドラがエラーを返した場合もログを出してからエラーハンドラへ引き継ぐ。
func AccessLog() WebHandler {
	return func(ctx *WebCtx) error {
		start := time.Now()
		err := ctx.Next()
		// 実際に返る status を記録する（エラー時の推定規則は errorStatusCode に集約）
		status := errorStatusCode(err, ctx.StatusCode())
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		slog.LogAttrs(ctx.Context(), level, "access",
			slog.String("method", ctx.Method()),
			slog.String("path", ctx.OriginalURL()),
			slog.Int("status", status),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("ip", ctx.IP()),
			slog.String("ua", ctx.UserAgent()),
		)
		return err
	}
}
