package mpc

import (
	"io"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/mpc/streamio"
)

type commSCCConn struct {
	msgConn MsgConn
	r       io.Reader
}

// NewConn creates a new connection conn backed by the comm scc
func NewConn(stub shim.ChaincodeStubInterface, sessionID string, targetPeer string, server bool) (Conn, error) {
	msgConn, err := NewMsgConn(stub, sessionID, targetPeer, server)
	conn := &commSCCConn{
		msgConn: msgConn,
		r:       streamio.NewReader(msgConn),
	}

	return conn, err
}

func (c *commSCCConn) Write(data []byte) (n int, err error) {
	return c.msgConn.Write(data)
}

func (c *commSCCConn) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *commSCCConn) Flush() error {
	return c.msgConn.Flush()
}
