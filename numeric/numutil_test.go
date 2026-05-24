package gw_numeric

import (
	"encoding/json"
	"testing"
)

func TestCNullUInt(t *testing.T) {
	var n json.Number = "123"
	if got := CNullUInt(n); got != 123 {
		t.Fatalf("unexpected: %d", got)
	}
}

func TestCNullStrIntAndJson(t *testing.T) {
	if v := CNullStrInt("1,234"); v == nil || *v != 1234 {
		t.Fatalf("unexpected: %#v", v)
	}
	js := map[string]interface{}{"n": " 42 ", "f": 10.0}
	if v := CNullStrIntByJson(js, "n"); v == nil || *v != 42 {
		t.Fatalf("unexpected: %#v", v)
	}
	if v := CNullFloatByJson(js, "f"); v == nil || *v != 10.0 {
		t.Fatalf("unexpected: %#v", v)
	}
	if v := CNullFloatToIntByJson(js, "f"); v == nil || *v != 10 {
		t.Fatalf("unexpected: %#v", v)
	}
}
