
import (
	"sync"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/generalworks/scheduler_api/constant"
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
func CreateToken(id string) (string, error) {
	key, err := loadV4Key()
	if err != nil {
		return "", gw_errors.Wrap(err)
	}

	token := paseto.NewToken()
	token.Set("Id", id)
	now := time.Now()
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	// 必要に応じて有効期限を調整（ここでは30日）
	token.SetExpiration(now.Add(30 * 24 * time.Hour))

	encrypted := token.V4Encrypt(key, nil) // no implicit assertion
	return encrypted, nil
}

// VerifyData decrypts and validates a v4.local token, returning the parsed token.
func VerifyData(tokenStr string) (*paseto.Token, error) {
	key, err := loadV4Key()
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
