package gw_web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

const compressTestPath = "/compress-test"

func compressTestErrorHandler(ctx *WebCtx, err error) error {
	return ctx.Status(http.StatusInternalServerError).SendString(err.Error())
}

func registerCompressTestRoute(app *WebApp) {
	app.Get(compressTestPath, func(ctx *WebCtx) error {
		return ctx.SendString(strings.Repeat("a", 4096))
	})
}

func compressTestRequest(t *testing.T, app *WebApp, accept string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, compressTestPath, http.NoBody)
	req.Header.Set(fiber.HeaderAcceptEncoding, "gzip")
	if accept != "" {
		req.Header.Set(fiber.HeaderAccept, accept)
	}
	resp, err := app.App.(*fiber.App).Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	return resp
}

func TestNewAppWithSettingsCompressesNormalRequest(t *testing.T) {
	app := NewAppWithSettings(compressTestErrorHandler, &AppSettings{
		CompressSkip: SkipCompressForStreaming,
	})
	registerCompressTestRoute(app)

	resp := compressTestRequest(t, app, "")
	defer resp.Body.Close()
	if got := resp.Header.Get(fiber.HeaderContentEncoding); got != "gzip" {
		t.Fatalf("Content-Encoding: want %q, got %q", "gzip", got)
	}
}

func TestNewAppWithSettingsSkipsCompressionForSSE(t *testing.T) {
	app := NewAppWithSettings(compressTestErrorHandler, &AppSettings{
		CompressSkip: SkipCompressForStreaming,
	})
	registerCompressTestRoute(app)

	resp := compressTestRequest(t, app, "text/event-stream")
	defer resp.Body.Close()
	if got := resp.Header.Get(fiber.HeaderContentEncoding); got != "" {
		t.Fatalf("Content-Encoding: want empty, got %q", got)
	}
}

func TestSkipCompressForStreamingWebSocketUpgrade(t *testing.T) {
	app := fiber.New()
	var got bool
	app.Get("/", func(c fiber.Ctx) error {
		got = SkipCompressForStreaming(&WebCtx{Ctx: c})
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(fiber.HeaderUpgrade, "WebSocket")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	if !got {
		t.Fatal("SkipCompressForStreaming returned false for websocket upgrade")
	}
}

func TestNewAppKeepsCompressionEnabled(t *testing.T) {
	app := NewApp(compressTestErrorHandler)
	registerCompressTestRoute(app)

	resp := compressTestRequest(t, app, "")
	defer resp.Body.Close()
	if got := resp.Header.Get(fiber.HeaderContentEncoding); got != "gzip" {
		t.Fatalf("Content-Encoding: want %q, got %q", "gzip", got)
	}
}

// CompressSkip 未指定でも WS/SSE はデフォルトでスキップされる。
func TestNewAppSkipsCompressionForSSEByDefault(t *testing.T) {
	app := NewApp(compressTestErrorHandler)
	registerCompressTestRoute(app)

	resp := compressTestRequest(t, app, "text/event-stream")
	defer resp.Body.Close()
	if got := resp.Header.Get(fiber.HeaderContentEncoding); got != "" {
		t.Fatalf("Content-Encoding: want empty, got %q", got)
	}
}
