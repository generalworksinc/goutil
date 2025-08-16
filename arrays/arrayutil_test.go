package gw_arrays

import (
	"reflect"
	"testing"
)

func TestContainsAndIndexString(t *testing.T) {
	arr := []string{"a", "b", "c"}
	if !ContainsString(arr, "b") {
		t.Errorf("expected to contain 'b'")
	}
	if ContainsString(arr, "z") {
		t.Errorf("did not expect to contain 'z'")
	}
	if idx := IndexString(arr, "c"); idx != 2 {
		t.Errorf("unexpected index: %d", idx)
	}
	if idx := IndexString(arr, "x"); idx != -1 {
		t.Errorf("unexpected index for not found: %d", idx)
	}
}

func TestRemoveDuplicate(t *testing.T) {
	arr := []int{1, 2, 2, 3, 1}
	got := RemoveDuplicate(arr)
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected unique slice: %#v", got)
	}
}

func TestFilter(t *testing.T) {
	arr := []int{1, 2, 3, 4, 5}
	got := Filter(arr, func(v int) bool { return v%2 == 0 })
	expected := []int{2, 4}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected filtered slice: %#v", got)
	}
}
