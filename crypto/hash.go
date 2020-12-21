package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
)

type params struct {
	memory          uint32
	iterations      uint32
	parallelism     uint8
	saltAfterLength uint32
	keyLength       uint32
}

var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

func GenerateHash(saltBefore string, str string) (string, error) {
	p := &params{
		memory:          64 * 1024,
		iterations:      3,
		parallelism:     2,
		saltAfterLength: 16,
		keyLength:       32,
	}

	saltAfterBytes, err := generateRandomBytes(p.saltAfterLength)
	if err != nil {
		return "", err
	}
	saltAfter := string(saltAfterBytes)
	saltBytes := []byte(saltBefore + saltAfter)
	encodeHash, err := generateFromStr(str, saltBytes, saltAfterBytes, p)
	return encodeHash, err
}

func generateFromStr(str string, saltAll []byte, saltAfter []byte, p *params) (encodedHash string, err error) {
	hash := argon2.IDKey([]byte(str), saltAll, p.iterations, p.memory, p.parallelism, p.keyLength)

	// Base64 encode the salt and hashed str.
	b64Salt := base64.RawStdEncoding.EncodeToString(saltAfter)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return a string using the standard encoded hash representation.
	encodedHash = fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.memory, p.iterations, p.parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func ComparePasswordAndHash(password, salteBefore string, encodedHash string) (match bool, err error) {
	// Extract the parameters, salt and derived key from the encoded password
	// hash.
	p, saltAfter, hash, err := DecodeHash(encodedHash)
	if err != nil {
		return false, err
	}
	saltBytes := []byte(salteBefore + string(saltAfter))
	// Derive the key from the other password using the same parameters.
	otherHash := argon2.IDKey([]byte(password), saltBytes, p.iterations, p.memory, p.parallelism, p.keyLength)

	// Check that the contents of the hashed passwords are identical. Note
	// that we are using the subtle.ConstantTimeCompare() function for this
	// to help prevent timing attacks.
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func DecodeHash(encodedHash string) (p *params, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	p = &params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	p.saltAfterLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}
