package core

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"
)

const size = 64 << 10

func stack() []byte {
	buf := make([]byte, size)
	n := runtime.Stack(buf, false)
	return buf[:n]
}

func startWithRecover(invokerName string, invoker func() error) {
	var (
		panicked = true
		err      error
	)

	defer func() {
		if v := recover(); v != nil || panicked {
			formatted := fmt.Sprintf("server panic: %v %s", v, stack())
			// output to stderr
			_, _ = fmt.Fprint(os.Stderr, formatted)
			err = recoverFrom(v)

			// TODO(@yeqown): backoff strategy of restart, if the invoker panics too quick.
			time.Sleep(5 * time.Second)
			go startWithRecover(invokerName, invoker)
		}
	}()

	if err = invoker(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "startWithRecover: component(%s) quit: %v", invokerName, err)
	}

	panicked = false
}

func recoverFrom(v interface{}) (err error) {
	if v == nil {
		return errors.New("nil panic")
	}

	err = errors.Wrap(err, "server panic")
	return
}
