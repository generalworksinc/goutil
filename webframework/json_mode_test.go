package gw_web

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
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
		"nil":          nil,
		"zero":         {},
		"JSONV1":       {JSON: JSONV1},
		"emptyOptions": {JSONOptions: []json.Options{}},
	} {
		t.Run(name, func(t *testing.T) { wantV1(t, probeMode(t, newModeProbeApp(t, settings))) })
	}
}

func TestJSONModeV2(t *testing.T) {
	wantV2(t, probeMode(t, newModeProbeApp(t, &AppSettings{JSON: JSONV2})))
}

func TestJSONModeV2MatchesDefaultOptionsV2(t *testing.T) {
	a := probeMode(t, newModeProbeApp(t, &AppSettings{JSON: JSONV2}))
	b := probeMode(t, newModeProbeApp(t, &AppSettings{JSONOptions: []json.Options{json.DefaultOptionsV2()}}))
	if a.out != b.out || a.decodeUpper != b.decodeUpper {
		t.Fatalf("JSON: JSONV2 と JSONOptions: DefaultOptionsV2() が違う: %+v / %+v", a, b)
	}
	for k := range strictBodies {
		if a.strict[k] != b.strict[k] {
			t.Errorf("strict %s: %d / %d", k, a.strict[k], b.strict[k])
		}
	}
}

// JSONOptions で細かく決めた設定は、DecodeJSON と DecodeJSONStrict にも同じく効く
func TestJSONOptionsReachDecodeHelpers(t *testing.T) {
	app := newModeProbeApp(t, &AppSettings{JSONOptions: []json.Options{jsonv1.DefaultOptionsV1(), json.FormatNilSliceAsNull(false)}})
	r := probeMode(t, app)
	if r.out != `{"tags":[]}` || r.decodeUpper != "name=z" {
		t.Fatalf("out=%s decode(NAME)=%s", r.out, r.decodeUpper)
	}
	if r.strict["unknown"] != 400 || r.strict["trailing"] != 400 || r.strict["ok"] != 200 {
		t.Fatalf("strict: %v", r.strict)
	}
}

func TestJSONModeRejectsInvalidSettings(t *testing.T) {
	for name, settings := range map[string]*AppSettings{
		"JSONV2 と JSONOptions": {JSON: JSONV2, JSONOptions: []json.Options{json.DefaultOptionsV2()}},
		"未知のモード":               {JSON: JSONMode(7)},
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
