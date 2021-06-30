package gw_common

import (
	cryptorand "crypto/rand"
	"encoding/binary"
)

func CryptoRandSeed() int64 {
	var seed int64
	err := binary.Read(cryptorand.Reader, binary.LittleEndian, &seed)
	if err != nil {
		panic(err)
	}
	return seed
}
