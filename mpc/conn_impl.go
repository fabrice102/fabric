package mpc

import (
	"errors"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

const (
	COMM_SCC = "commscc"
	SEND     = "send"
	RECEIVE  = "receive"
)

type commSCCConn struct {
	stub       shim.ChaincodeStubInterface
	sessionID  []byte
	targetPeer []byte
}

// NewCommSCCConn creates a new connection conn backed by the comm scc
func NewConn(stub shim.ChaincodeStubInterface, sessionID string, targetPeer string) (Conn, error) {
	return &commSCCConn{
		stub:       stub,
		sessionID:  []byte(sessionID),
		targetPeer: []byte(targetPeer),
	}, nil
}

func (c *commSCCConn) Write(data []byte) (n int, err error) {
	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(SEND), data, c.sessionID, c.targetPeer},
		"",
	)

	if r.Status != shim.OK {
		return 0, fmt.Errorf("failed sending message to [%s]: [%s]", string(c.targetPeer), r.String())
	}

	return len(data), nil
}

func (c *commSCCConn) Read(p []byte) (n int, err error) {
	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(RECEIVE), c.sessionID},
		"",
	)

	if r.Status != shim.OK {
		return 0, fmt.Errorf("failed receiving message [%s][%s]", r.String(), r.Payload)
	}

	if r.Payload == nil {
		return 0, errors.New("failed receiving message [payload is nil]")
	}

	copy(p, r.Payload)

	return len(r.Payload), nil

}

func (c *commSCCConn) Flush() error {
	return nil
}

/*
func (c *commSCCChannel) Send(payload []byte, endpoint string) error {
	fmt.Printf("[%v] Send [%v] to [%v]\n",  string(c.sessionID), string(payload), endpoint)

	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(SEND), payload, c.sessionID, []byte(endpoint)},
		"",
	)

	if r.Status != shim.OK {
		return fmt.Errorf("failed sending message to [%s]: [%s]", endpoint, r.String())
	}

	return nil
}

func (c *commSCCChannel) Receive(timeout int) ([]byte, error) {
	fmt.Printf("[%v] Receive with timeout [%v]\n",  string(c.sessionID), timeout)

	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(RECEIVE), c.sessionID},
		"",
	)

	if r.Status != shim.OK {
		return nil, fmt.Errorf("failed receiving message [%s][%s]", r.String(), r.Payload)
	}

	if r.Payload == nil {
		return nil, errors.New("failed receiving message [payload is nil]")
	}

	return r.Payload, nil
}
*/
