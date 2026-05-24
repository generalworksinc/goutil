package gw_map

import (
	"reflect"
	"testing"
)

func TestGetKeysValuesFromMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	keys := GetKeysFromMap(m)
	vals := GetValuesFromMap(m)
	if len(keys) != 2 || len(vals) != 2 {
		t.Fatalf("unexpected lengths: %d %d", len(keys), len(vals))
	}
}

func TestDoubleKeyMapJSON(t *testing.T) {
	var dkm DoubleKeyMap[string, string, int]
	dkm = make(DoubleKeyMap[string, string, int])
	dkm.Set("A", "X", 1)
	dkm.Set("A", "Y", 2)
	dkm.Set("B", "Z", 3)

	b, err := dkm.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back DoubleKeyMap[string, string, int]
	if err := back.UnmarshalJSON(b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(dkm, back) {
		t.Fatalf("roundtrip mismatch")
	}
}
