package internal

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

const stackSize = 64 << 10

func stack() []byte {
	buf := make([]byte, stackSize)
	n := runtime.Stack(buf, false)
	return buf[:n]
}

// GoFunc runs invoker in a goroutine and restarts it after panic.
func GoFunc(invokerName string, invoker func() error) {
	fn := func() {
		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				_, _ = fmt.Fprintf(os.Stderr, "goroutine panic: %v %s", v, stack())
				time.Sleep(5 * time.Second)
				GoFunc(invokerName, invoker)
			}
		}()

		if err := invoker(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "component(%s) quit: %v", invokerName, err)
		}
		panicked = false
	}

	go fn()
}
