package hash

import (
	"math/rand"
	"time"
)

var src = rand.NewSource(time.Now().UnixNano())

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	letterBytes = "abcdefghijklmnopqrstuvwxyz123456790ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// RandKey get one random string with specified length.
// reference from https://colobu.com/2018/09/02/generate-random-string-in-Go/#Source
func RandKey(n int) string {
	if n < 6 {
		n = 6
	}

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}
