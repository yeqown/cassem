package runtime

import (
	"fmt"
	"runtime"
)

const size = 64 << 10

func Stack() []byte {
	buf := make([]byte, size)
	n := runtime.Stack(buf, false)
	return buf[:n]
}

func RecoverFrom(v any) (err error) {
	err = fmt.Errorf("panic: %v", v)
	return
}
