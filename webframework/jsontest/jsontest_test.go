package jsontest_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gw_web "github.com/generalworksinc/goutil/webframework"
	"github.com/generalworksinc/goutil/webframework/jsontest"
	"github.com/gofiber/fiber/v3"
)

func newApp(settings *gw_web.AppSettings) *gw_web.WebApp {
	app := gw_web.NewAppWithSettings(func(c *gw_web.WebCtx, err error) error { return err }, settings)
	app.Get("/", func(c *gw_web.WebCtx) error {
		return c.JSON(struct {
			Tags []string `json:"tags"`
		}{})
	})
	return app
}

func body(t *testing.T, app *gw_web.WebApp) string {
	t.Helper()
	resp, err := app.App.(*fiber.App).Test(httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const v1Body, v2Body = `{"tags":null}`, `{"tags":[]}`

func TestBuildOverridesOnlyAppsBuiltInside(t *testing.T) {
	v1Settings := &gw_web.AppSettings{JSON: gw_web.JSONV1}
	v2 := jsontest.Build(gw_web.JSONV2, func() *gw_web.WebApp { return newApp(v1Settings) })
	after := newApp(v1Settings)
	if got := body(t, v2); got != v2Body {
		t.Fatalf("Build の中で作った app は v2: got %s", got)
	}
	if got := body(t, after); got != v1Body {
		t.Fatalf("Build の後で作った app は設定どおり v1: got %s", got)
	}
}

func TestBuildV1OverridesV2Settings(t *testing.T) {
	app := jsontest.Build(gw_web.JSONV1, func() *gw_web.WebApp { return newApp(&gw_web.AppSettings{JSON: gw_web.JSONV2}) })
	if got := body(t, app); got != v1Body {
		t.Fatalf("got %s", got)
	}
}

func TestBuildNestedRestoresOuterMode(t *testing.T) {
	var inner, outer *gw_web.WebApp
	jsontest.Build(gw_web.JSONV2, func() int {
		inner = jsontest.Build(gw_web.JSONV1, func() *gw_web.WebApp { return newApp(nil) })
		outer = newApp(nil)
		return 0
	})
	if body(t, inner) != v1Body || body(t, outer) != v2Body {
		t.Fatalf("inner=%s outer=%s", body(t, inner), body(t, outer))
	}
}

func TestBuildRestoresAfterPanic(t *testing.T) {
	func() {
		defer func() { _ = recover() }()
		jsontest.Build(gw_web.JSONV2, func() int { panic("x") })
	}()
	if got := body(t, newApp(nil)); got != v1Body {
		t.Fatalf("panic の後も上書きが残っている: %s", got)
	}
}
