package gw_encode

import "testing"

func TestSjisUtf8RoundTrip(t *testing.T) {
	const s = "あいうABC"
	sjis, err := Utf8ToSjis(s)
	if err != nil {
		t.Fatalf("Utf8ToSjis error: %v", err)
	}
	utf8, err := SjisToUtf8(sjis)
	if err != nil {
		t.Fatalf("SjisToUtf8 error: %v", err)
	}
	if utf8 != s {
		t.Fatalf("roundtrip mismatch: %q", utf8)
	}
}
