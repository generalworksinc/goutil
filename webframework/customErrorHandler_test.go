package gw_web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// collectErrorData を実リクエストの WebCtx で検証する。
func collectViaRoute(t *testing.T, opts ErrorHandlerOptions) ErrorData {
	t.Helper()
	var got ErrorData
	app := NewApp(func(c *WebCtx, err error) error { return c.Status(http.StatusInternalServerError).SendString("x") })
	app.Post("/probe", func(c *WebCtx) error {
		got = collectErrorData(c, errors.New("boom"), http.StatusInternalServerError, "msg", "stack", "user-1", "v1", opts)
		return c.SendString("ok")
	})
	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(`{"email":"a@example.test","password":"secret-password"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "probe-agent/1.0")
	resp, err := app.App.(*fiber.App).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	return got
}

func TestCollectErrorDataDefaultIncludesBodyAndClientInfo(t *testing.T) {
	data := collectViaRoute(t, ErrorHandlerOptions{})
	if !strings.Contains(data.Body, "secret-password") {
		t.Fatalf("default options must keep the request body for compatibility, got %q", data.Body)
	}
	if data.Ua != "probe-agent/1.0" || data.Ip == "" {
		t.Fatalf("default options must keep client info, got ua=%q ip=%q", data.Ua, data.Ip)
	}
	if data.Error != "boom" || data.Method != http.MethodPost || !strings.HasSuffix(data.Url, "/probe") {
		t.Fatalf("unexpected error data: %+v", data)
	}
}

func TestCollectErrorDataOmitsBodyAndClientInfo(t *testing.T) {
	data := collectViaRoute(t, ErrorHandlerOptions{OmitRequestBody: true, OmitClientInfo: true})
	if data.Body != "" {
		t.Fatalf("OmitRequestBody must drop the body, got %q", data.Body)
	}
	if data.Ip != "" || data.Ua != "" {
		t.Fatalf("OmitClientInfo must drop ip/ua, got ip=%q ua=%q", data.Ip, data.Ua)
	}
	if data.Error != "boom" || data.StackTrace != "stack" || data.UserId != "user-1" {
		t.Fatalf("other fields must be kept: %+v", data)
	}
}

func TestCustomHTTPErrorHandlerWithOptionsStillResponds(t *testing.T) {
	app := NewApp(CustomHTTPErrorHandlerWithOptions("v1", func(*WebCtx) string { return "" }, ErrorHandlerOptions{OmitRequestBody: true, OmitClientInfo: true}))
	app.Get("/fail", func(c *WebCtx) error { return errors.New("boom") })
	resp, err := app.App.(*fiber.App).Test(httptest.NewRequest(http.MethodGet, "/fail", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
