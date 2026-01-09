package gw_web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"aidanwoods.dev/go-paseto"
	gw_errors "github.com/generalworksinc/goutil/errors"
	"github.com/gofiber/fiber/v3"
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
	var expireTime time.Time
	// token.SetNotBefore(now)
	// 必要に応じて有効期限を調整（ここでは30日）
	if exp != nil {
		expireTime = now.Add(*exp)
	} else {
		expireTime = now.Add(30 * 24 * time.Hour)
	}
	token.SetExpiration(expireTime)

	encrypted := token.V4Encrypt(key, nil) // no implicit assertion
	return encrypted, &expireTime, nil
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

type RefreshTokenCookieOptions struct {
	// Name defaults to "refresh_token"
	Name string
	// Path defaults to "/api"
	Path string
	// Domain defaults to empty (host-only cookie)
	Domain string
	// SameSite defaults to "Lax"
	SameSite string
	// HTTPOnly defaults to true
	HTTPOnly *bool
	// Secure defaults to false if nil. (prod などで true にしたい場合は明示的に渡す)
	Secure *bool
	// MaxAgeSeconds overrides computed MaxAge. If nil, MaxAge is computed from expiresAt.
	MaxAgeSeconds *int
}

func SetRefreshTokenCookie(c *WebCtx, token string, expiresAt time.Time, opt *RefreshTokenCookieOptions) {
	ck := &WebCookie{Cookie: &fiber.Cookie{}}
	fc := ck.Cookie.(*fiber.Cookie)

	name := "refresh_token"
	path := "/api"
	domain := ""
	sameSite := "Lax"
	httpOnly := true
	if opt != nil {
		if opt.Name != "" {
			name = opt.Name
		}
		if opt.Path != "" {
			path = opt.Path
		}
		if opt.Domain != "" {
			domain = opt.Domain
		}
		if opt.SameSite != "" {
			sameSite = opt.SameSite
		}
		if opt.HTTPOnly != nil {
			httpOnly = *opt.HTTPOnly
		}
	}

	fc.Name = name
	fc.Value = token
	fc.Path = path
	fc.Domain = domain
	fc.HTTPOnly = httpOnly
	fc.SameSite = sameSite
	fc.Expires = expiresAt

	// MaxAge は秒指定。期限が過去なら削除扱い。
	maxAge := int(time.Until(expiresAt).Seconds())
	if opt != nil && opt.MaxAgeSeconds != nil {
		maxAge = *opt.MaxAgeSeconds
	}
	if token == "" {
		maxAge = -1
		fc.Expires = time.Unix(0, 0)
	}
	fc.MaxAge = maxAge

	if opt != nil && opt.Secure != nil {
		fc.Secure = *opt.Secure
	}
	c.Cookie(ck)
}
