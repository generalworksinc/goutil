package gw_common

import (
	cryptorand "crypto/rand"
)

// func CryptoRandSeed() int64 {
// 	var seed int64
// 	err := binary.Read(cryptorand.Reader, binary.LittleEndian, &seed)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return seed
// }

func CryptoRandSeed() [32]byte {
	var seed [32]byte
	_, err := cryptorand.Read(seed[:])
	if err != nil {
		panic(err)
	}
	return seed
}
