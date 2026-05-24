package gw_japanese_address

import "testing"

func TestNormalizeAddress(t *testing.T) {
	in := "東京都渋谷区神南一丁目 二番 三号 ５階"
	out := NormalizeAddress(in)
	if out == "" {
		t.Fatalf("unexpected empty")
	}
}
