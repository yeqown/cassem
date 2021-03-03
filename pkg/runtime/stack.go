package runtime

import (
	"errors"
	"fmt"
	"runtime"
)

const size = 64 << 10

func Stack() []byte {
	buf := make([]byte, size)
	n := runtime.Stack(buf, false)
	return buf[:n]
}

func RecoverFrom(v interface{}) (err error) {
	if v == nil {
		return errors.New("panic nil")
	}

	err = fmt.Errorf("panic: %v", err)
	return
}
