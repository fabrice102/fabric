package mpc

import (
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/gossip"
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
	return &commSCCChannel{stub: stub, sessionID: sessionID}
}

func (c *commSCCChannel) Send(payload []byte, endpoint string) error {
	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(SEND), c.sessionID, payload, []byte(endpoint)},
		"",
	)

	if r.Status != shim.OK {
		return fmt.Errorf("failed sending message to [%s]: [%s]", endpoint, r.String())
	}

	return nil
}

func (c *commSCCChannel) Receive(timeout int) ([]byte, error) {
	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(RECEIVE), c.sessionID},
		"",
	)

	if r.Status != shim.OK {
		return nil, fmt.Errorf("failed receiving message [%s]", r.String())
	}

	mpcDataMsg := &gossip.MPCDataMessage{}
	err := proto.Unmarshal(r.Payload, mpcDataMsg)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling mpc data message on receive [%s]", err)
	}

	if mpcDataMsg.Payload == nil {
		return nil, errors.New("failed unmarshalling mpc data message on receive [payload is nil]")
	}

	return mpcDataMsg.Payload.Data, nil
}
