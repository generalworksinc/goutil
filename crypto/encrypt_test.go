package gw_crypto

import "testing"

func TestEncryptCFBAndDecrypt(t *testing.T) {
	key := make([]byte, 32)
	copy(key, []byte("0123456789abcdef0123456789abcdef"))
	plain := []byte("hello world")

	enc, err := Encrypt(key, plain)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	got, err := Decrypt(key, enc)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}
	if string(got) != string(plain) {
		t.Fatalf("unexpected decrypt: %q", got)
	}
}

func TestEncryptAESGCM(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey error: %v", err)
	}
	plain := []byte("hello gcm")
	ciphertext, err := EncryptAESGCM(key, plain)
	if err != nil {
		t.Fatalf("EncryptAESGCM error: %v", err)
	}
	plaintxt, err := DecryptAESGCM(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptAESGCM error: %v", err)
	}
	if string(plaintxt) != string(plain) {
		t.Fatalf("unexpected plaintext: %q", plaintxt)
	}
}
