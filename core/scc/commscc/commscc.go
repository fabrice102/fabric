/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package commscc

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/gossip"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("commscc")

const (
	SEND    = "send"
	RECEIVE = "receive"
)

type action func(stub shim.ChaincodeStubInterface) pb.Response

type CommSCC struct {
	*util.PubSub

	actions map[string]action

	rmc chan comm.ReceivedMessageImpl
}

func (scc *CommSCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	defer logger.Infof("Successfully initialized CommSCC.")

	scc.actions = map[string]action{}

	// Define the functions the chaincode handles
	scc.actions[SEND] = scc.send
	scc.actions[RECEIVE] = scc.receive

	// SHAI: Do we really need this? Otherwise scc.PubSub.Subscribe in commscc.go crashes when trying to lock at line 
	scc.PubSub = util.NewPubSub()

	// Start listening to MPC messages.
	// This needs to be called once and for all.
	_, rmc := service.GetGossipService().Accept(scc.mpcMessageAcceptor, true)
	go func() {
		// TODO: do we need a way to exit the loop?
		for msg := range rmc {
			// Publish the message using as topic the session ID.
			// Session ID can be chosen arbitrarily. One way to choose it
			// is by setting it to the transaction ID.
			scc.Publish(string(msg.GetGossipMessage().GetMpcData().Payload.SessionID), msg)
		}
	}()

	return shim.Success(nil)
}

func (scc *CommSCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()

	logger.Debugf("commscc invoked function [%s]", function)
	action, exists := scc.actions[function]
	if exists {
		return action(stub)
	}

	return shim.Error(fmt.Sprintf("function [%s] does not exist", function))
}

func (scc *CommSCC) send(stub shim.ChaincodeStubInterface) pb.Response {
	// Send a payload args[1] with sessionID args[2] to a given endpoint args[3]
	args := stub.GetArgs()
	// Payload
	payload := args[1]
	// SessionID
	sessionID := args[2]
	// Unmarshal the endpoint
	endpoint := string(args[3])

	// TODO: replace this with one with SendByCriteria to receive an ack
	service.GetGossipService().Send(
		&gossip.GossipMessage{
			Nonce: 0,
			// TODO: Which tag works better here?
			//Tag:     gossip.GossipMessage_CHAN_AND_ORG,
			Content: &gossip.GossipMessage_MpcData{
				MpcData: &gossip.MPCDataMessage{Payload: &gossip.MPCPayload{
					SessionID: sessionID,
					Data:      payload,
				}},
			},
		},
		&comm.RemotePeer{Endpoint: endpoint},
	)

	return shim.Success(nil)
}

func (scc *CommSCC) receive(stub shim.ChaincodeStubInterface) pb.Response {
	// Read a message from topic args[1]
	args := stub.GetArgs()
	// Topic
	topic := string(args[1])

	// Wait for the message on the given topic for a given amount of time
	// TODO: allow the invoker to specify the timeout
	sub := scc.PubSub.Subscribe(topic, time.Second*10)
	msg, err := sub.Listen()
	if err != nil {
		return shim.Error(fmt.Sprintf("failed receive [%s]", err))
	}

	// Given init, we expect to see a ReceivedMessage here.
	mpcData := msg.(gossip.ReceivedMessage).GetGossipMessage().GetMpcData()
	if mpcData == nil {
		return shim.Error("received empty mpc message.")
	}

	// Marshall MPC Data
	raw, err := proto.Marshal(mpcData)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed marshalling receive mpc message [%s]", err))
	}

	return shim.Success(raw)
}

func (scc *CommSCC) mpcMessageAcceptor(input interface{}) bool {
	// input is supposed to be of type ReceivedMessage.
	// If it is not the case, return false
	msg, ok := input.(gossip.ReceivedMessage)
	if !ok {
		// Not a ReceivedMessage
		return false
	}

	// Is this message an MPC message?
	return msg.GetGossipMessage().IsMpcData()
}
