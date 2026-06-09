package runtime

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// GoFunc runs invoker in an independent goroutine and the goroutine will automatically recover from panic,
// and restart invoker under control of backoff algorithm.
func GoFunc(invokerName string, invoker func() error) {
	fn := func() {
		var (
			panicked = true
			err      error
		)

		defer func() {
			if v := recover(); v != nil || panicked {
				formatted := fmt.Sprintf("server panic: %v %s", v, Stack())
				// output to stderr
				_, _ = fmt.Fprint(os.Stderr, formatted)
				err = recoverFrom(v)

				// DONE(@yeqown): strategy: delay time duration of restart, avoid that the invoker panics too quick.
				time.Sleep(5 * time.Second)
				GoFunc(invokerName, invoker)
			}
		}()

		if err = invoker(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "startWithRecover: component(%s) quit: %v", invokerName, err)
		}

		panicked = false
	}

	go fn()
}

func recoverFrom(v any) error {
	if v == nil {
		return errors.New("nil panic")
	}

	if err, ok := v.(error); ok {
		return fmt.Errorf("server panic: %w", err)
	}

	return fmt.Errorf("server panic: %v", v)
}
