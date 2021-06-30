package gw_uuid

import (
	gw_common "github.com/generalworksinc/goutil/common"
	"github.com/oklog/ulid/v2"
	"math/rand"
	"time"
)

func GetUlid() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(gw_common.CryptoRandSeed())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

func GetUlidFromTimestamp(t *time.Time) string {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(gw_common.CryptoRandSeed())), 0)
	return ulid.MustNew(ulid.Timestamp(*t), entropy).String()
}
