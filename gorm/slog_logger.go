package gw_gorm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// SlogLogger は GORM のログを slog（構造化JSON）で出力する logger.Interface 実装。
// Trace は slog.XxxContext 相当で呼ぶため、gw_log の context 注入ハンドラを
// 使っていれば SQL ログにも request_id / user_id が自動付与される。
type SlogLogger struct {
	// LogLevel は GORM 側の出力レベル（Silent/Error/Warn/Info）。
	LogLevel gormlogger.LogLevel
	// SlowThreshold を超えたクエリは WARN で slow_query として出力する（0 なら無効）。
	SlowThreshold time.Duration
}

// NewSlogLogger は SQL ログ用の SlogLogger を返す。
// debug=true なら全クエリを INFO で、false ならスロークエリ(200ms超)とエラーのみ出力する。
func NewSlogLogger(debug bool) *SlogLogger {
	if debug {
		return &SlogLogger{LogLevel: gormlogger.Info, SlowThreshold: 200 * time.Millisecond}
	}
	return &SlogLogger{LogLevel: gormlogger.Warn, SlowThreshold: 200 * time.Millisecond}
}

func (l *SlogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *SlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		slog.InfoContext(ctx, fmt.Sprintf(msg, data...), slog.String("source", "gorm"))
	}
}

func (l *SlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		slog.WarnContext(ctx, fmt.Sprintf(msg, data...), slog.String("source", "gorm"))
	}
}

func (l *SlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		slog.ErrorContext(ctx, fmt.Sprintf(msg, data...), slog.String("source", "gorm"))
	}
}

func (l *SlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	switch {
	// ErrRecordNotFound は「不在」であってエラーではないため ERROR にはしない
	// （Info レベルでは通常 SQL として記録される）
	case err != nil && l.LogLevel >= gormlogger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		l.trace(ctx, slog.LevelError, "sql error", fc, elapsed, slog.String("error", err.Error()))
	case l.SlowThreshold > 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn:
		l.trace(ctx, slog.LevelWarn, "slow query", fc, elapsed)
	case l.LogLevel >= gormlogger.Info:
		l.trace(ctx, slog.LevelInfo, "sql", fc, elapsed)
	}
}

// trace は slog 側のレベルで捨てられる場合に fc()（SQL文字列構築）を実行しない。
func (l *SlogLogger) trace(ctx context.Context, level slog.Level, msg string, fc func() (string, int64), elapsed time.Duration, extra ...slog.Attr) {
	if !slog.Default().Enabled(ctx, level) {
		return
	}
	sql, rows := fc()
	slog.LogAttrs(ctx, level, msg, sqlAttrs(sql, rows, elapsed, extra...)...)
}

func sqlAttrs(sql string, rows int64, elapsed time.Duration, extra ...slog.Attr) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("source", "gorm"),
		slog.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
		slog.Int64("rows", rows),
		slog.String("sql", sql),
	}
	return append(attrs, extra...)
}
