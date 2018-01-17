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

type commSCCChannel struct {
	stub      shim.ChaincodeStubInterface
	sessionID []byte
}

// NewCommSCCChannel creates a new Channel backed by the comm scc
// and using txID as sessionID
func NewCommSCCChannel(stub shim.ChaincodeStubInterface) Channel {
	sessionID := []byte(stub.GetTxID())
	fmt.Printf("Session ID: [%v]\n", string(sessionID))
	return &commSCCChannel{stub: stub, sessionID: sessionID}
}

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
