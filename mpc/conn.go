package mpc

import (
	"io"
)

type Channel interface {
	Send(payload []byte, endpoint string) error

	Receive(timeout int) ([]byte, error)
}

/* Dream interface ideally non-blocking */
// Rename Channel into Conn

// We want a function:
// newConn(targetPeer string, sessionID string)

type Conn interface {
	io.Writer
	io.Reader
	Flush() error
}
