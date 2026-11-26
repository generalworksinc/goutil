package gw_web

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type jsonProbe struct {
	Name string         `json:"name"`
	Tags []string       `json:"tags"`
	M    map[string]int `json:"m"`
}

func newJSONProbeApp(t *testing.T, opts ...json.Options) *WebApp {
	t.Helper()
	app := NewAppWithSettings(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}, &AppSettings{JSONOptions: opts})
	app.Get("/out", func(ctx *WebCtx) error {
		return ctx.JSON(jsonProbe{Name: "a", M: map[string]int{"q": 1, "w": 2, "e": 3, "r": 4, "t": 5, "y": 6, "u": 7, "i": 8, "o": 9, "p": 10}})
	})
	app.Post("/in", func(ctx *WebCtx) error {
		var in jsonProbe
		if err := ctx.BindJSON(&in); err != nil {
			return err
		}
		return ctx.SendString("name=" + in.Name)
	})
	return app
}

func doJSONProbe(t *testing.T, app *WebApp, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.App.(*fiber.App).Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, string(b)
}

const sortedProbeMap = `"m":{"e":3,"i":8,"o":9,"p":10,"q":1,"r":4,"t":5,"u":7,"w":2,"y":6}`

func TestJSONOptionsUnsetKeepsV1Behavior(t *testing.T) {
	app := newJSONProbeApp(t)
	_, out := doJSONProbe(t, app, http.MethodGet, "/out", "")
	if want := `{"name":"a","tags":null,` + sortedProbeMap + `}`; out != want {
		t.Fatalf("response: want %s got %s", want, out)
	}
	_, in := doJSONProbe(t, app, http.MethodPost, "/in", `{"NAME":"z"}`)
	if in != "name=z" {
		t.Fatalf("v1 matches names case-insensitively, got %q", in)
	}
}

func TestJSONOptionsV2(t *testing.T) {
	app := newJSONProbeApp(t, json.DefaultOptionsV2(), json.Deterministic(true))
	for range 5 {
		_, out := doJSONProbe(t, app, http.MethodGet, "/out", "")
		if want := `{"name":"a","tags":[],` + sortedProbeMap + `}`; out != want {
			t.Fatalf("response: want %s got %s", want, out)
		}
	}
	_, in := doJSONProbe(t, app, http.MethodPost, "/in", `{"NAME":"z"}`)
	if in != "name=" {
		t.Fatalf("v2 matches names case-sensitively, got %q", in)
	}
	status, _ := doJSONProbe(t, app, http.MethodPost, "/in", `{"name":"a","name":"b"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("duplicate member names must be rejected, got status %d", status)
	}
}

func TestJSONOptionsV1WithOneOptionRemoved(t *testing.T) {
	app := newJSONProbeApp(t, jsonv1.DefaultOptionsV1(), json.FormatNilSliceAsNull(false))
	_, out := doJSONProbe(t, app, http.MethodGet, "/out", "")
	if want := `{"name":"a","tags":[],` + sortedProbeMap + `}`; out != want {
		t.Fatalf("response: want %s got %s", want, out)
	}
	_, in := doJSONProbe(t, app, http.MethodPost, "/in", `{"NAME":"z"}`)
	if in != "name=z" {
		t.Fatalf("remaining v1 options keep case-insensitive matching, got %q", in)
	}
}
