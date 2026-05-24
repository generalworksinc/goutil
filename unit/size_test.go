package gw_unit

import "testing"

func TestFormatFileSize(t *testing.T) {
	if got := FormatFileSize(999); got != "999 B" {
		t.Fatalf("unexpected: %s", got)
	}
	if got := FormatFileSize(1024); got != "1.00 KB" {
		t.Fatalf("unexpected: %s", got)
	}
}
