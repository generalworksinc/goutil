package gw_uuid

import (
	"testing"
	"time"
)

func TestGetUlid(t *testing.T) {
	id := GetUlid()
	if id == "" {
		t.Fatalf("empty ulid")
	}
}

func TestGetUlidFromTimestamp(t *testing.T) {
	now := time.Now()
	id := GetUlidFromTimestamp(&now)
	if id == "" {
		t.Fatalf("empty ulid from ts")
	}
}
