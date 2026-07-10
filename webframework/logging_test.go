package gw_web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gw_log "github.com/generalworksinc/goutil/logging"
	"github.com/gofiber/fiber/v3"
)

// slog のデフォルトロガーを buffer 出力に差し替え、テスト後に復元する
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(gw_log.NewHandler(slog.NewJSONHandler(&buf, nil))))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return &buf
}

func newLoggingTestApp() *WebApp {
	return NewApp(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusInternalServerError).SendString(err.Error())
	})
}

func TestRequestIdUsesIncomingHeaderAndEchoesIt(t *testing.T) {
	buf := captureSlog(t)
	app := newLoggingTestApp()
	app.Use(RequestId())

	var seenInHandler string
	app.Get("/", func(ctx *WebCtx) error {
		seenInHandler = gw_log.RequestIdFromContext(ctx.Context())
		slog.InfoContext(ctx.Context(), "in handler")
		return ctx.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(HeaderRequestId, "incoming-id")
	resp, err := app.App.(*fiber.App).Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if got := resp.Header.Get(HeaderRequestId); got != "incoming-id" {
		t.Errorf("response header = %q, want incoming-id", got)
	}
	if seenInHandler != "incoming-id" {
		t.Errorf("handler ctx request id = %q, want incoming-id", seenInHandler)
	}
	if !strings.Contains(buf.String(), `"request_id":"incoming-id"`) {
		t.Errorf("slog output missing request_id: %s", buf.String())
	}
}

func TestRequestIdGeneratesUlidWhenNoHeader(t *testing.T) {
	app := newLoggingTestApp()
	app.Use(RequestId())
	app.Get("/", func(ctx *WebCtx) error { return ctx.SendString("ok") })

	resp, err := app.App.(*fiber.App).Test(httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	rid := resp.Header.Get(HeaderRequestId)
	if len(rid) != 26 { // ULID は26文字
		t.Errorf("generated request id = %q, want 26-char ULID", rid)
	}
}

func TestAccessLogEmitsStructuredLine(t *testing.T) {
	buf := captureSlog(t)
	app := newLoggingTestApp()
	app.Use(RequestId(), AccessLog())
	app.Get("/hello", func(ctx *WebCtx) error { return ctx.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/hello?q=1", http.NoBody)
	req.Header.Set(HeaderRequestId, "acc-1")
	if _, err := app.App.(*fiber.App).Test(req); err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	var accessLine map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err == nil && m["msg"] == "access" {
			accessLine = m
		}
	}
	if accessLine == nil {
		t.Fatalf("no access log line found in: %s", buf.String())
	}
	if accessLine["method"] != "GET" || accessLine["path"] != "/hello?q=1" {
		t.Errorf("unexpected method/path: %v / %v", accessLine["method"], accessLine["path"])
	}
	if accessLine["status"] != float64(200) {
		t.Errorf("status = %v, want 200", accessLine["status"])
	}
	if accessLine["request_id"] != "acc-1" {
		t.Errorf("request_id = %v, want acc-1", accessLine["request_id"])
	}
	if _, ok := accessLine["latency_ms"]; !ok {
		t.Error("latency_ms missing")
	}
}

func TestRequestIdFallsBackToAmznTraceId(t *testing.T) {
	app := newLoggingTestApp()
	app.Use(RequestId())
	app.Get("/", func(ctx *WebCtx) error { return ctx.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Amzn-Trace-Id", "Root=1-abc")
	resp, err := app.App.(*fiber.App).Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if got := resp.Header.Get(HeaderRequestId); got != "Root=1-abc" {
		t.Errorf("response header = %q, want Root=1-abc", got)
	}
}

// ハンドラが ctx.Status(404) 済みで非 fiber.Error を返すケース:
// CustomHTTPErrorHandler はセット済み status を上書きしないため、アクセスログも 404 を記録すべき
func TestAccessLogRespectsPresetStatusOnError(t *testing.T) {
	buf := captureSlog(t)
	app := newLoggingTestApp()
	app.Use(AccessLog())
	app.Get("/preset", func(ctx *WebCtx) error {
		ctx.Status(http.StatusNotFound)
		return errPlain
	})

	if _, err := app.App.(*fiber.App).Test(httptest.NewRequest(http.MethodGet, "/preset", http.NoBody)); err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if !strings.Contains(buf.String(), `"status":404`) {
		t.Errorf("access log should record preset 404: %s", buf.String())
	}
}

var errPlain = &plainError{}

type plainError struct{}

func (*plainError) Error() string { return "plain error" }

func TestAccessLogLevelsByStatus(t *testing.T) {
	buf := captureSlog(t)
	app := newLoggingTestApp()
	app.Use(AccessLog())
	app.Get("/notfound", func(ctx *WebCtx) error { return ctx.Status(http.StatusNotFound).SendString("nf") })
	app.Get("/boom", func(ctx *WebCtx) error { return fiber.ErrInternalServerError })

	for _, path := range []string{"/notfound", "/boom"} {
		if _, err := app.App.(*fiber.App).Test(httptest.NewRequest(http.MethodGet, path, http.NoBody)); err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
	}
	out := buf.String()
	if !strings.Contains(out, `"level":"WARN"`) {
		t.Errorf("expected WARN access log for 404: %s", out)
	}
	if !strings.Contains(out, `"level":"ERROR"`) {
		t.Errorf("expected ERROR access log for 500: %s", out)
	}
}
