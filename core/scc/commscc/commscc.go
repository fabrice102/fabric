/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package commscc

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/gossip/discovery"
	gossip_svc "github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("commscc")

const (
	SEND    = "send"
	RECEIVE = "receive"

	// maxMessageSaveTime denotes the maximum period of time to save a message
	// in memory until it is purged
	maxMessageSaveTime = time.Second * 10

	ackWaitTime = time.Second * 3
)

type action func(stub shim.ChaincodeStubInterface) pb.Response

type CommSCC struct {
	pubSub  *util.PubSub
	actions map[string]action
	gossip_svc.Gossip
	msgStore *MessageStore
}

func NewCommSCC() *CommSCC {
	c := &CommSCC{
		pubSub:   util.NewPubSub(),
		Gossip:   service.GetGossipService(),
		msgStore: NewMessageStore(),
	}
	c.actions = map[string]action{
		SEND:    c.send,
		RECEIVE: c.receive,
	}
	go c.listen()
	return c
}

func (scc *CommSCC) listen() {
	_, rmc := scc.Accept(mpcMessageAcceptor, true)
	for msg := range rmc {
		mpcMsg := msg.GetGossipMessage().GetMpcData()
		session := mpcMsg.Payload.SessionID
		addr := msg.GetConnectionInfo().Endpoint
		logger.Debugf("Message from %s in session %s, contains payload of %d bytes", addr, string(session), len(mpcMsg.Payload.Data))
		// Notify the other side we've received the message
		msg.Ack(nil)
		// Probe to see if some receive operation is interested in this session
		err := scc.pubSub.Publish(hex.EncodeToString(session), msg)
		if err == nil {
			continue
		}
		// If the publish failed we need to save the message
		scc.msgStore.Put(hex.EncodeToString(session), msg)

		// Cleanup in any case
		time.AfterFunc(maxMessageSaveTime, func() {
			scc.msgStore.Remove(hex.EncodeToString(session))
		})
	}
}

func (scc *CommSCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Infof("Successfully initialized CommSCC.")
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

	logger.Infof("[%v] Send [%v] to [%v]", string(sessionID), string(payload), endpoint)

	msg := &gossip.GossipMessage{
		Nonce: 0,
		Content: &gossip.GossipMessage_MpcData{
			MpcData: &gossip.MPCDataMessage{Payload: &gossip.MPCPayload{
				SessionID: sessionID,
				Data:      payload,
			}},
		},
	}
	sMsg, err := msg.NoopSign()
	if err != nil {
		logger.Panicf("Failed marshaling gossip message: %v", err)
	}

	err = scc.SendByCriteria(sMsg, gossip_svc.SendCriteria{
		MaxPeers: 1,
		MinAck:   1,
		Timeout:  ackWaitTime,
		IsEligible: func(member discovery.NetworkMember) bool {
			return member.PreferredEndpoint() == endpoint
		},
	})
	if err != nil {
		logger.Warningf("Failed sending message to %s: %v", endpoint, err)
		return shim.Error(fmt.Sprintf("failed sending message to %s: %v", endpoint, err))
	}
	return shim.Success(nil)
}

func (scc *CommSCC) receive(stub shim.ChaincodeStubInterface) pb.Response {
	// Read a message from topic args[1]
	args := stub.GetArgs()
	// Topic
	topic := string(args[1])
	// timeout in millisecond
	timeout, err := strconv.Atoi(string(args[2]))
	// source peer
	endpoint := string(args[3])

	if err != nil {
		logger.Errorf("The second argument 'timeout' needs to be an integer")
		return shim.Error(fmt.Sprintf("The second argument 'timeout' needs to be an integer"))
	}

	logger.Infof("Receive on [%v]", topic)

	var totalMessages []gossip.ReceivedMessage
	// First, check if we have messages waiting for us
	waitingMessages := scc.msgStore.MsgsByID(topic)
	totalMessages = append(totalMessages, waitingMessages...)
	scc.msgStore.Remove(topic)

	// Wait for the message on the given topic for a given amount of time
	sub := scc.pubSub.Subscribe(topic, time.Millisecond*time.Duration(timeout))
	for {
		msgRaw, err := sub.Listen()
		if err != nil {
			break
		}
		msg := msgRaw.(gossip.ReceivedMessage)
		totalMessages = append(totalMessages, msg)
	}

	msg := findMessage(totalMessages, endpoint)
	if msg == nil {
		return shim.Error(fmt.Sprintf("Didn't receive any message from %s on session %s, but did receive %d messages", endpoint, topic, len(totalMessages)))
	}
	// Given init, we expect to see a ReceivedMessage here.
	mpcData := msg.GetGossipMessage().GetMpcData()
	logger.Infof("Received on [%v], [%v]", topic, mpcData.Payload.Data)

	// TODO: check that payload is different from nil
	return shim.Success(mpcData.Payload.Data)
}

func findMessage(msgs []gossip.ReceivedMessage, endpoint string) gossip.ReceivedMessage {
	for _, msg := range msgs {
		sID := &msp.SerializedIdentity{}
		proto.Unmarshal(msg.GetConnectionInfo().Identity, sID)
		bl, _ := pem.Decode(sID.IdBytes)
		cert, _ := x509.ParseCertificate(bl.Bytes)
		actualEndPoint := cert.Subject.CommonName

		// Check the certificate to tell who is the sender
		if strings.Compare(strings.Split(endpoint, ":")[0], actualEndPoint) != 0 {
			logger.Warningf("Expected to receive msg from [%s], not [%s]", endpoint, actualEndPoint)
			continue
		}
		return msg
	}
	return nil
}

func mpcMessageAcceptor(input interface{}) bool {
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
