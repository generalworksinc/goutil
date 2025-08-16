package gw_models

import "testing"

func TestBeforeInsertUUID(t *testing.T) {
	b := &BaseModel{}
	b.BeforeInsert()
	if b.Id == "" {
		t.Fatalf("id not set")
	}
}

func TestBeforeInsertUlid(t *testing.T) {
	u := &BaseModelUlid{}
	u.BeforeInsert()
	if u.Id == "" {
		t.Fatalf("ulid not set")
	}
}
