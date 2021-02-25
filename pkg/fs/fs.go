package fs

import (
	"os"
)

func Exists(p string) bool {
	_, err := os.Stat(p)
	if err != nil {
		return os.IsExist(err)
	}

	return true
}
