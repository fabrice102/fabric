/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package commscc

import (
	"crypto/x509"
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
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"sync"
	"github.com/hyperledger/fabric/gossip/service"
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
	sync.Mutex
	pubSub  *util.PubSub
	actions map[string]action
	gossip_svc.Gossip
	msgStore *MessageStore
}

func NewCommSCC() *CommSCC {
	c := &CommSCC{
		pubSub:   util.NewPubSub(),
		msgStore: NewMessageStore(),
	}
	c.actions = map[string]action{
		SEND:    c.send,
		RECEIVE: c.receive,
	}
	go c.listen()
	return c
}

func waitForGossip() {
	for {
		gossipSvc := service.GetGossipService()
		if gossipSvc != nil {
			return
		}
		time.Sleep(time.Second)
	}
}

func (scc *CommSCC) listen() {
	waitForGossip()
	scc.Gossip = service.GetGossipService()
	logger.Info(">>> Gossip instance established")
	_, rmc := scc.Accept(mpcMessageAcceptor, true)
	for msg := range rmc {
		logger.Info(">>>> Got message", string(msg.GetGossipMessage().GetMpcData().Payload.Data), "from", msg.GetConnectionInfo().Endpoint)
		scc.handleMessage(msg)
	}
}

func (scc *CommSCC) handleMessage(msg gossip.ReceivedMessage) {
	mpcMsg := msg.GetGossipMessage().GetMpcData()
	session := mpcMsg.Payload.SessionID
	addr := msg.GetConnectionInfo().Endpoint
	logger.Infof(">>> Message from %s in session %s, contains payload of %d bytes", addr, string(session), len(mpcMsg.Payload.Data))
	// Notify the other side we've received the message
	msg.Ack(nil)
	scc.relayMessageToReceivers(msg, string(session))
}

func (scc *CommSCC) relayMessageToReceivers(msg gossip.ReceivedMessage, session string) {
	// Probe to see if some receive operation is interested in this session
	err := scc.pubSub.Publish(session, msg)
	if err == nil {
		logger.Info(">>> publish succeeded")
		return
	}
	logger.Info(">>>> publish failed")

	// If the publish failed we need to save the message
	scc.msgStore.Put(session, msg)

	// Cleanup in any case
	time.AfterFunc(maxMessageSaveTime, func() {
		logger.Info("Purging messages related to", session)
		scc.msgStore.Remove(session)
	})

	// Publish again, because the receive operation may have missed the message store if it
	// came too early to the comm scc
	scc.pubSub.Publish(session, msg)
}

func (scc *CommSCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Infof("Successfully initialized CommSCC.")
	return shim.Success(nil)
}

func (scc *CommSCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()

	logger.Infof("commscc invoked function [%s]", function)
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
	rStub := ReceivedStub{stub}
	if err := rStub.Validate(); err != nil {
		return shim.Error(err.Error())
	}
	topic := rStub.Topic()
	endpoint := rStub.Endpoint()

	logger.Infof("Receive on [%v]", topic)

	// Subscribe to messages for the topic
	sub := scc.pubSub.Subscribe(topic, time.Second * 5)

	// First, check if we have messages waiting for us.
	// This may happen if we invoked Subscribe() too late
	waitingMessages := scc.msgStore.Search(topic, messageFrom(endpoint))
	logger.Info(">>>> collected", len(waitingMessages), "messages waiting for this session")
	scc.msgStore.Remove(topic)
	if len(waitingMessages) > 0 {
		res := waitingMessages[0].GetGossipMessage().GetMpcData().Payload.Data
		logger.Info(">>>>> Returning", string(res))
		return shim.Success(res)
	}

	var totalMessages []gossip.ReceivedMessage
	// Next, wait for messages that we may have received after subscribing.
	totalMessages = append(totalMessages, drainSubscription(sub) ...)
	logger.Info("Received", len(totalMessages), "via subscription")
	if len(totalMessages) == 0 {
		logger.Warningf("Didn't receive any message from %s - received %d messages", endpoint, len(totalMessages))
		return shim.Error(fmt.Sprintf("Didn't receive any message from %s", endpoint))
	}
	// Grab the first message from endpoint
	msg := totalMessages[0]
	// Given init, we expect to see a ReceivedMessage here.
	mpcData := msg.GetGossipMessage().GetMpcData()
	logger.Infof("Received on [%v], [%v]", topic, mpcData.Payload.Data)
	// TODO: check that payload is different from nil
	return shim.Success(mpcData.Payload.Data)
}

func drainSubscription(sub util.Subscription) []gossip.ReceivedMessage {
	var res []gossip.ReceivedMessage
	for {
		msgRaw, err := sub.Listen()
		if err != nil {
			break
		}
		msg := msgRaw.(gossip.ReceivedMessage)
		res = append(res, msg)
		logger.Info(">>>> got", string(msg.GetGossipMessage().GetMpcData().Payload.Data))
		break
		// Yacov: Should we wait for more messages from other peers? Ideally we have an expected number of messages we should collect
		// so that we won't stop by timing out... Ask me if that's not clear
	}
	if len(res) == 0 {
		logger.Info("Didn't find any message from subscriptions")
	} else {
		logger.Info("Found", len(res), "messages from subscriptions")
	}
	return res
}

func messageFrom(endpoint string) func(gossip.ReceivedMessage) bool {
	return func(msg gossip.ReceivedMessage) bool {
		sID := &msp.SerializedIdentity{}
		proto.Unmarshal(msg.GetConnectionInfo().Identity, sID)
		bl, _ := pem.Decode(sID.IdBytes)
		cert, _ := x509.ParseCertificate(bl.Bytes)
		actualEndPoint := cert.Subject.CommonName

		// Check the certificate to tell who is the sender
		if strings.Compare(strings.Split(endpoint, ":")[0], actualEndPoint) != 0 {
			logger.Warningf("Expected to receive msg from [%s], not [%s]", endpoint, actualEndPoint)
			return false
		}
		return true
	}
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

type ReceivedStub struct {
	shim.ChaincodeStubInterface
}

func (stub ReceivedStub) Validate() error {
	if len(stub.GetArgs()) == 4 {
		return nil
	}
	return fmt.Errorf("there should be exactly 4 arguments")
}

func (stub ReceivedStub) Topic() string {
	return string(stub.GetArgs()[1])
}

func (stub ReceivedStub) Timeout() (time.Duration, error) {
	timeout, err := strconv.Atoi(string(stub.GetArgs()[2]))
	if err != nil {
		return 0, err
	}
	return time.Millisecond * time.Duration(timeout), nil
}

func (stub ReceivedStub) Endpoint() string {
	return string(stub.GetArgs()[3])
}