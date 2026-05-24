package gw_date

import (
	"testing"
	"time"
)

func TestJapaneseEraFormats(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	reiwaStart := time.Date(2019, time.May, 1, 0, 0, 0, 0, jst)

	name, year, err := FormatJapaneseEraYear(&reiwaStart)
	if err != nil || name != "令和" || year != 1 {
		t.Fatalf("unexpected era: %s %d (%v)", name, year, err)
	}

	str, err := FormatJapaneseEraYYYYMD(&reiwaStart)
	if err != nil || str == "" {
		t.Fatalf("unexpected era ymd: %q (%v)", str, err)
	}
}

func TestMonthBoundaries(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	ref := time.Date(2024, 2, 15, 12, 0, 0, 0, loc)
	last := GetLastDayOfMonth(ref, loc)
	if last.Day() == 1 || last.Month() != 2 {
		t.Fatalf("unexpected last day: %v", last)
	}
	first := GetFirstDayOfMonth(ref, loc)
	if first.Day() != 1 || first.Month() != 2 {
		t.Fatalf("unexpected first day: %v", first)
	}
}
