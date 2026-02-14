package gw_web

import (
	"encoding/hex"
	"testing"
	"time"
)

const testPasetoV4KeyHex = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

func TestCreateAccessTokenAndVerifyData(t *testing.T) {
	ttl := 2 * time.Hour
	token, expAt, err := CreateAccessToken(testPasetoV4KeyHex, "user-123", &ttl)
	if err != nil {
		t.Fatalf("CreateAccessToken error: %v", err)
	}
	if token == "" {
		t.Fatalf("token should not be empty")
	}
	if expAt == nil || expAt.Before(time.Now()) {
		t.Fatalf("expire time should be in the future: %v", expAt)
	}

	parsed, err := VerifyData(testPasetoV4KeyHex, token)
	if err != nil {
		t.Fatalf("VerifyData error: %v", err)
	}
	id, err := parsed.GetString("Id")
	if err != nil {
		t.Fatalf("token should include Id claim: %v", err)
	}
	if id != "user-123" {
		t.Fatalf("id mismatch. expected=user-123 got=%s", id)
	}
}

func TestVerifyDataInvalidToken(t *testing.T) {
	if _, err := VerifyData(testPasetoV4KeyHex, "invalid-token"); err == nil {
		t.Fatalf("invalid token should return error")
	}
}

func TestCreateRefreshToken(t *testing.T) {
	ttl := 10 * time.Minute
	token, expAt, err := CreateRefreshToken(ttl)
	if err != nil {
		t.Fatalf("CreateRefreshToken error: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("refresh token should be 64 hex chars. got len=%d", len(token))
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("refresh token should be valid hex: %v", err)
	}
	if expAt.Before(time.Now()) {
		t.Fatalf("expiry should be in the future: %v", expAt)
	}
}

func TestHashToken(t *testing.T) {
	h1 := HashToken("sample-token")
	h2 := HashToken("sample-token")
	h3 := HashToken("another-token")

	if len(h1) != 64 {
		t.Fatalf("hash should be 64 hex chars. got len=%d", len(h1))
	}
	if h1 != h2 {
		t.Fatalf("same input should produce same hash")
	}
	if h1 == h3 {
		t.Fatalf("different input should produce different hash")
	}
}
