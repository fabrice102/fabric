package mpc

import (
	"io"

	"github.com/hyperledger/fabric/mpc/streamio"
)

type Channel interface {
	Send(payload []byte, endpoint string) error

	Receive(timeout int) ([]byte, error)
}

/* Dream interface ideally non-blocking */
// Rename Channel into Conn

// We want a function blocking (timeout):
// NewConn(stub shim.ChaincodeStubInterface, targetPeer string, sessionID string) (Conn, error)

type MsgConn interface {
	io.Writer
	streamio.MsgReader
	Flush() error
}
