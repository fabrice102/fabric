package mpc

import (
	"io"
)

/* Dream interface ideally non-blocking */
// Rename Channel into Conn

// We want a function blocking (timeout):
// NewConn(stub shim.ChaincodeStubInterface, targetPeer string, sessionID string) (Conn, error)

type Conn interface {
	io.Writer
	io.Reader
	Flush() error
}
