package gw_uuid

import (
	"math/rand/v2"
	"time"

	gw_common "github.com/generalworksinc/goutil/common"
	"github.com/oklog/ulid/v2"
)

func GetUlid() string {
	t := time.Now()
	// entropy := ulid.Monotonic(rand.New(rand.NewSource(gw_common.CryptoRandSeed())), 0)
	entropy := ulid.Monotonic(rand.NewChaCha8(gw_common.CryptoRandSeed()), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

func GetUlidFromTimestamp(t *time.Time) string {
	// entropy := ulid.Monotonic(rand.New(rand.NewSource(gw_common.CryptoRandSeed())), 0)
	entropy := ulid.Monotonic(rand.NewChaCha8(gw_common.CryptoRandSeed()), 0)
	return ulid.MustNew(ulid.Timestamp(*t), entropy).String()
}
