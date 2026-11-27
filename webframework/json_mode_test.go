package gw_web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type modeProbe struct {
	Name string `json:"name"`
}

func newModeProbeApp(t *testing.T, settings *AppSettings) *WebApp {
	t.Helper()
	app := NewAppWithSettings(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}, settings)
	app.Get("/out", func(ctx *WebCtx) error {
		return ctx.JSON(struct {
			Tags []string `json:"tags"`
		}{})
	})
	read := func(decode func(*WebCtx, []byte, any) error) WebHandler {
		return func(ctx *WebCtx) error {
			var in modeProbe
			if err := decode(ctx, ctx.Body(), &in); err != nil {
				return err
			}
			return ctx.SendString("name=" + in.Name)
		}
	}
	app.Post("/decode", read(func(ctx *WebCtx, b []byte, v any) error { return ctx.DecodeJSON(b, v) }))
	app.Post("/strict", read(func(ctx *WebCtx, b []byte, v any) error { return ctx.DecodeJSONStrict(b, v) }))
	return app
}

type modeResult struct {
	out, decodeUpper string
	strict           map[string]int
}

var strictBodies = map[string]string{
	"ok":       `{"name":"a"}`,
	"unknown":  `{"name":"a","x":1}`,
	"trailing": `{"name":"a"} {"name":"b"}`,
	"dupKey":   `{"name":"a","name":"b"}`,
}

func probeMode(t *testing.T, app *WebApp) modeResult {
	t.Helper()
	r := modeResult{strict: map[string]int{}}
	_, r.out = doJSONProbe(t, app, http.MethodGet, "/out", "")
	_, r.decodeUpper = doJSONProbe(t, app, http.MethodPost, "/decode", `{"NAME":"z"}`)
	for k, body := range strictBodies {
		r.strict[k], _ = doJSONProbe(t, app, http.MethodPost, "/strict", body)
	}
	return r
}

func wantV1(t *testing.T, r modeResult) {
	t.Helper()
	if r.out != `{"tags":null}` || r.decodeUpper != "name=z" {
		t.Fatalf("v1: out=%s decode(NAME)=%s", r.out, r.decodeUpper)
	}
	want := map[string]int{"ok": 200, "unknown": 400, "trailing": 400, "dupKey": 200}
	for k, v := range want {
		if r.strict[k] != v {
			t.Errorf("v1 strict %s: want %d got %d", k, v, r.strict[k])
		}
	}
}

func wantV2(t *testing.T, r modeResult) {
	t.Helper()
	if r.out != `{"tags":[]}` || r.decodeUpper != "name=" {
		t.Fatalf("v2: out=%s decode(NAME)=%s", r.out, r.decodeUpper)
	}
	want := map[string]int{"ok": 200, "unknown": 400, "trailing": 400, "dupKey": 400}
	for k, v := range want {
		if r.strict[k] != v {
			t.Errorf("v2 strict %s: want %d got %d", k, v, r.strict[k])
		}
	}
}

func TestJSONModeDefaultIsV1(t *testing.T) {
	for name, settings := range map[string]*AppSettings{
		"nil":    nil,
		"zero":   {},
		"JSONV1": {JSON: JSONV1},
	} {
		t.Run(name, func(t *testing.T) { wantV1(t, probeMode(t, newModeProbeApp(t, settings))) })
	}
}

func TestJSONModeV2(t *testing.T) {
	wantV2(t, probeMode(t, newModeProbeApp(t, &AppSettings{JSON: JSONV2})))
}

func TestJSONModeRejectsInvalidSettings(t *testing.T) {
	for name, settings := range map[string]*AppSettings{
		"未知のモード": {JSON: JSONMode(7)},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil || !strings.HasPrefix(r.(string), "gw_web: ") {
					t.Fatalf("panic を期待: %v", r)
				}
			}()
			newModeProbeApp(t, settings)
		})
	}
}

// AppOption が Fiber の State を Reset しても、DecodeJSONStrict は app の設定で読める
func TestDecodeJSONStrictSurvivesAppOptionStateReset(t *testing.T) {
	app := NewAppWithSettings(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}, &AppSettings{JSON: JSONV2}, func(a *fiber.App) { a.State().Reset() })
	app.Post("/strict", func(ctx *WebCtx) error {
		var in modeProbe
		if err := ctx.DecodeJSONStrict(ctx.Body(), &in); err != nil {
			return err
		}
		return ctx.SendString("name=" + in.Name)
	})
	if status, body := doJSONProbe(t, app, http.MethodPost, "/strict", `{"name":"a"}`); status != http.StatusOK || body != "name=a" {
		t.Fatalf("status=%d body=%s", status, body)
	}
}
