package gw_crypto

import "testing"

func TestArgon2GenerateAndCompare(t *testing.T) {
	saltBefore := "prefix-"
	pass := "secret"

	hash, err := GenerateHash(saltBefore, pass)
	if err != nil {
		t.Fatalf("generate hash error: %v", err)
	}
	if hash == "" {
		t.Fatalf("empty hash")
	}

	match, err := ComparePasswordAndHash(pass, saltBefore, hash)
	if err != nil {
		t.Fatalf("compare error: %v", err)
	}
	if !match {
		t.Fatalf("password should match")
	}

	match, err = ComparePasswordAndHash("wrong", saltBefore, hash)
	if err != nil {
		t.Fatalf("compare error: %v", err)
	}
	if match {
		t.Fatalf("password should not match")
	}
}
