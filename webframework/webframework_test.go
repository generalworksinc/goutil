package gw_web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestNormalizeLoggerFormatConvertsLegacyHeaderTag(t *testing.T) {
	format := "${method} ${header:X-Request-ID} ${header}"
	got := normalizeLoggerFormat(format)
	want := "${method} ${reqHeader:X-Request-ID} ${reqHeaders}"
	if got != want {
		t.Fatalf("unexpected format. want=%q got=%q", want, got)
	}
}

func TestSetLoggerAcceptsLegacyHeaderTag(t *testing.T) {
	app := NewApp(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusInternalServerError).SendString(err.Error())
	})
	var out bytes.Buffer
	format := "${header:X-Test}\n"

	app.SetLogger(&out, &format)
	app.Get("/", func(ctx *WebCtx) error {
		return ctx.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Test", "legacy-header")
	resp, err := app.App.(*fiber.App).Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if !strings.Contains(out.String(), "legacy-header") {
		t.Fatalf("expected logger output to include legacy header value, got %q", out.String())
	}
}
