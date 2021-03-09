package hash

import (
	"encoding/hex"

	"golang.org/x/crypto/scrypt"
)

func WithSalt(password, salt string) string {
	hashedKey, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 32)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(hashedKey)
}
