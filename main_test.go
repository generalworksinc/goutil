package main

import (
	"log"
	"testing"

	gw_unit "github.com/generalworksinc/goutil/unit"
)

// formatFileSize 関数のテストケース
func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{123, "123 B"},
		{1234, "1.21 KB"},
		{1234567, "1.18 MB"},
		{1234567890, "1.15 GB"},
		{1234567890123, "1.12 TB"},
	}

	for _, test := range tests {
		result := gw_unit.FormatFileSize(test.input)
		log.Println(test.input, ":", result)
		if result != test.expected {
			t.Errorf("formatFileSize(%d) = %s; expected %s", test.input, result, test.expected)
		}
	}
}
