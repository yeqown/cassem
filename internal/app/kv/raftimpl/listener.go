package raftimpl

import (
	"errors"
	"net"
	"time"
)

// stoppableListener sets TCP keep-alive timeouts on accepted
// connections and waits on stopc message
type stoppableListener struct {
	*net.TCPListener
	stopc <-chan struct{}
}

func newStoppableListener(addr string, stopc <-chan struct{}) (*stoppableListener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	tcpListener, ok := ln.(*net.TCPListener)
	if !ok {
		_ = ln.Close()
		return nil, errors.New("listener is not tcp")
	}
	return &stoppableListener{tcpListener, stopc}, nil
}

func (ln stoppableListener) Accept() (c net.Conn, err error) {
	connc := make(chan *net.TCPConn, 1)
	errc := make(chan error, 1)
	go func() {
		tc, err := ln.AcceptTCP()
		if err != nil {
			errc <- err
			return
		}
		connc <- tc
	}()
	select {
	case <-ln.stopc:
		return nil, errors.New("server stopped")
	case err := <-errc:
		return nil, err
	case tc := <-connc:
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(3 * time.Minute)
		return tc, nil
	}
}
