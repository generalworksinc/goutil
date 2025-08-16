package gw_strings

import "testing"

func TestCompressDecompressStr(t *testing.T) {
	const s = "hello gzip base64"
	c, err := CompressStr(s)
	if err != nil {
		t.Fatalf("compress error: %v", err)
	}
	d, err := DecompressStr(c)
	if err != nil {
		t.Fatalf("decompress error: %v", err)
	}
	if d != s {
		t.Fatalf("roundtrip mismatch: %q", d)
	}
}

func TestSubstring(t *testing.T) {
	s := "日本語ABC"
	if got := Substring(s, 0, 2); got != "日本" {
		t.Fatalf("unexpected substring: %q", got)
	}
	if got := Substring(s, 3, 3); got != "ABC" {
		t.Fatalf("unexpected substring: %q", got)
	}
}
