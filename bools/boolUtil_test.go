package gw_bools

import "testing"

func TestCNullBoolByJson(t *testing.T) {
	js := map[string]interface{}{"ok": true, "str": "x"}
	if v := CNullBoolByJson(js, "ok"); v == nil || *v != true {
		t.Fatalf("expected true, got %#v", v)
	}
	if v := CNullBoolByJson(js, "missing"); v != nil {
		t.Fatalf("expected nil for missing key")
	}
	if v := CNullBoolByJson(js, "str"); v != nil {
		t.Fatalf("expected nil for non-bool value")
	}
}
