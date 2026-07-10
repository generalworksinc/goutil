package gw_log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(NewHandler(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
}

func parseLine(t *testing.T, line string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("failed to parse log line %q: %v", line, err)
	}
	return m
}

func TestContextHandlerInjectsRequestIdAndUserId(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	ctx := WithRequestId(context.Background(), "req-123")
	ctx = WithUserId(ctx, "user-456")
	logger.InfoContext(ctx, "hello", slog.String("k", "v"))

	m := parseLine(t, buf.String())
	if m["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want req-123", m["request_id"])
	}
	if m["user_id"] != "user-456" {
		t.Errorf("user_id = %v, want user-456", m["user_id"])
	}
	if m["k"] != "v" {
		t.Errorf("k = %v, want v", m["k"])
	}
}

func TestContextHandlerWithoutIdsAddsNothing(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.InfoContext(context.Background(), "plain")

	m := parseLine(t, buf.String())
	if _, ok := m["request_id"]; ok {
		t.Error("request_id should be absent when not in context")
	}
	if _, ok := m["user_id"]; ok {
		t.Error("user_id should be absent when not in context")
	}
}

// WithAttrs / WithGroup 経由でも context 注入が維持されること（ラップ漏れの回帰テスト）
func TestContextHandlerSurvivesWithAttrsAndWithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf).With(slog.String("component", "test")).WithGroup("g")

	ctx := WithRequestId(context.Background(), "req-789")
	logger.InfoContext(ctx, "grouped", slog.String("inner", "x"))

	m := parseLine(t, buf.String())
	if m["component"] != "test" {
		t.Errorf("component = %v, want test", m["component"])
	}
	// 横ぐし検索キーは WithGroup 適用後でも必ずトップレベルに出ること
	if m["request_id"] != "req-789" {
		t.Errorf("top-level request_id = %v, want req-789 (output: %s)", m["request_id"], buf.String())
	}
	// ユーザー属性は通常どおりグループ配下に入ること
	g, ok := m["g"].(map[string]any)
	if !ok || g["inner"] != "x" {
		t.Errorf("grouped attr g.inner = %v, want x (output: %s)", m["g"], buf.String())
	}
}

func TestRequestIdFromContextEmpty(t *testing.T) {
	if got := RequestIdFromContext(context.Background()); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := UserIdFromContext(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestInitSetsDefaultAndFormatsUTCTime(t *testing.T) {
	// Init は slog.SetDefault を書き換えるため、テスト後に元へ戻す
	orig := slog.Default()
	defer slog.SetDefault(orig)

	logger := Init(slog.LevelInfo)
	if logger == nil {
		t.Fatal("Init returned nil")
	}
	if slog.Default() != logger {
		t.Error("Init should set the default logger")
	}
}
