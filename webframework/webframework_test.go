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

func TestOpenAPIRegistersAppDocAndConvertsPathParams(t *testing.T) {
	app := NewApp(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusInternalServerError).SendString(err.Error())
	})
	app.GetDoc("/users/:id", RouteDoc{Summary: "Get user"}, func(ctx *WebCtx) error {
		return ctx.SendString("ok")
	})

	doc := app.OpenAPI()
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths should be map[string]any, got %T", doc["paths"])
	}
	pathDoc, ok := paths["/users/{id}"].(map[string]any)
	if !ok {
		t.Fatalf("expected OpenAPI path /users/{id}, got %#v", paths)
	}
	getDoc, ok := pathDoc["get"].(map[string]any)
	if !ok {
		t.Fatalf("expected get operation, got %#v", pathDoc)
	}
	if getDoc["summary"] != "Get user" {
		t.Fatalf("unexpected summary: %#v", getDoc["summary"])
	}
}

func TestOpenAPIRegistersGroupDocWithPrefix(t *testing.T) {
	app := NewApp(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusInternalServerError).SendString(err.Error())
	})
	group := app.Group("/api")
	group.PostDoc("users/:id", RouteDoc{Summary: "Update user"}, func(ctx *WebCtx) error {
		return ctx.SendString("ok")
	})

	doc := app.OpenAPI()
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths should be map[string]any, got %T", doc["paths"])
	}
	pathDoc, ok := paths["/api/users/{id}"].(map[string]any)
	if !ok {
		t.Fatalf("expected OpenAPI path /api/users/{id}, got %#v", paths)
	}
	postDoc, ok := pathDoc["post"].(map[string]any)
	if !ok {
		t.Fatalf("expected post operation, got %#v", pathDoc)
	}
	if postDoc["summary"] != "Update user" {
		t.Fatalf("unexpected summary: %#v", postDoc["summary"])
	}
}
