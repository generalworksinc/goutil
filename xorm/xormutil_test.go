package gw_xorm

import "testing"

type sub struct {
	A int `xorm:"extends"`
}

type sample struct {
	sub `xorm:"extends"`
	B   string
}

func TestGetStructFields(t *testing.T) {
	s := sample{sub: sub{A: 1}, B: "x"}
	fs := GetStructFields(&s, true, "")
	if len(fs) == 0 {
		t.Fatalf("no fields")
	}
}
