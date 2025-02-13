package main

import (
	"testing"

	gw_strings "github.com/generalworksinc/goutil/strings"
)

func TestErrorsWrap(t *testing.T) {
	strList := gw_strings.RandString6(100)
	t.Errorf("strList: %v", strList)
}
