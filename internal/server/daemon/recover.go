package daemon

import (
	"fmt"
	"os"
	"runtime"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
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
		}
	}()

	if err = invoker(); err != nil {
		log.Errorf("Daemon: component(%s) quit: %v", invoker, err)
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
