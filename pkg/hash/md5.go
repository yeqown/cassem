package hash

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5(v []byte) string {
	h := md5.New()
	_, _ = h.Write(v)
	return hex.EncodeToString(h.Sum(nil))
}
