package gw_web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"aidanwoods.dev/go-paseto"
	gw_errors "github.com/generalworksinc/goutil/errors"
)

var (
	v4SymmetricKey paseto.V4SymmetricKey
	keyOnce        sync.Once
	keyInitErr     error
)

func loadV4Key(hexString string) (paseto.V4SymmetricKey, error) {
	keyOnce.Do(func() {
		v4SymmetricKey, keyInitErr = paseto.V4SymmetricKeyFromHex(hexString)
	})
	return v4SymmetricKey, gw_errors.Wrap(keyInitErr)
}

// CreateToken creates a v4.local (symmetric) token with the user Id claim.
func CreateAccessToken(hexString string, id string, exp *time.Duration) (string, *time.Time, error) {
	key, err := loadV4Key(hexString)
	if err != nil {
		return "", nil, gw_errors.Wrap(err)
	}

	token := paseto.NewToken()
	token.Set("Id", id)
	now := time.Now()
	token.SetIssuedAt(now)
	// token.SetNotBefore(now)
	// 必要に応じて有効期限を調整（ここでは30日）
	if exp != nil {
		token.SetExpiration(now.Add(*exp))
	} else {
		token.SetExpiration(now.Add(30 * 24 * time.Hour))
	}

	encrypted := token.V4Encrypt(key, nil) // no implicit assertion
	return encrypted, exp, nil
}

// VerifyData decrypts and validates a v4.local token, returning the parsed token.
func VerifyData(hexString string, tokenStr string) (*paseto.Token, error) {
	key, err := loadV4Key(hexString)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}

	parser := paseto.NewParser() // validates exp/nbf/iat ifセット
	parsed, err := parser.ParseV4Local(key, tokenStr, nil)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}
	return parsed, nil
}

// CreateRefreshToken returns a random token string and expiry.
func CreateRefreshToken(ttl time.Duration) (string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, gw_errors.Wrap(err)
	}
	token := hex.EncodeToString(b)
	exp := time.Now().Add(ttl)
	return token, exp, nil
}

// HashToken creates a hex string hash (SHA-256) for storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
