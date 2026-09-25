package gw_web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestTypedLocalsAreDistinctAndKeepLegacyLocals(t *testing.T) {
	userKey := NewLocalKey[*struct{ Name string }]()
	otherKey := NewLocalKey[*struct{ Name string }]()
	app := fiber.New()
	app.Get("/", func(fiberCtx fiber.Ctx) error {
		ctx := WebCtx{Ctx: fiberCtx}
		if _, ok := ctx.Local(otherKey); ok {
			t.Error("unset key unexpectedly returned a value")
		}
		user := &struct{ Name string }{Name: "alice"}
		ctx.SetLocal(userKey, user)
		ctx.Locals("Auth", "legacy")
		got, ok := ctx.Local(userKey)
		if !ok || got != user {
			t.Errorf("typed local mismatch: got=%v ok=%v", got, ok)
		}
		if _, ok := ctx.Local(otherKey); ok {
			t.Error("distinct key returned another key's value")
		}
		if got := ctx.Locals("Auth"); got != "legacy" {
			t.Errorf("legacy local changed: %v", got)
		}
		ctx.SetLocal(otherKey, nil)
		if got, ok := ctx.Local(otherKey); !ok || got != nil {
			t.Errorf("typed nil local was lost: got=%v ok=%v", got, ok)
		}
		return fiberCtx.SendStatus(http.StatusOK)
	})

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
}
