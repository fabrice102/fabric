package mpc

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

const (
	COMM_SCC        = "commscc"
	SEND            = "send"
	RECEIVE         = "receive"
	DEFAULT_TIMEOUT = time.Second * 240
)

type commSCCMsgConn struct {
	stub       shim.ChaincodeStubInterface
	sessionID  []byte
	targetPeer []byte
	timeout    time.Duration

	writeCounter uint64
	readCounter  uint64
}

// NewCommSCCConn creates a new connection conn backed by the comm scc
func NewMsgConn(stub shim.ChaincodeStubInterface, sessionID string, targetPeer string, server bool) (MsgConn, error) {
	conn := &commSCCMsgConn{
		stub:       stub,
		sessionID:  []byte(sessionID),
		targetPeer: []byte(targetPeer),
		timeout:    DEFAULT_TIMEOUT,

		writeCounter: 0,
		readCounter:  0,
	}

	return conn, nil
}

// Ths will be the Write to be used
func (c *commSCCMsgConn) Write(data []byte) (n int, err error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, c.writeCounter)
	topic := append(c.sessionID, b...)
	c.writeCounter++
	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(SEND), data, topic, c.targetPeer},
		"",
	)

	fmt.Printf("Qi Zhang, send message %s, %v\n", string(c.sessionID), c.writeCounter)
	if r.Status != shim.OK {
		errMsg := fmt.Errorf("failed sending message to [%s]: [%s]", string(c.targetPeer), r.String())
		fmt.Printf("Fabrice -- %v\n", errMsg)
		return 0, errMsg
	}

	return len(data), nil
}

// Debugging purpose: appending a counter in front of each message to check whether there is any message missing
/*
func (c *commSCCMsgConn) Write(data []byte) (n int, err error) {
	time.Sleep(time.Millisecond * 200)
	data = append([]byte{c.writeCounter}, data...)
	fmt.Printf("Write: write message %v, [%v]\n", c.writeCounter, data)
	c.writeCounter++

	return c.RawWrite(data)
}
*/

func (c *commSCCMsgConn) Read() (p []byte, err error) {
	timeout := []byte(strconv.FormatInt(c.timeout.Nanoseconds()/int64(1000000), 10))

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, c.readCounter)

	topic := append(c.sessionID, b...)
	c.readCounter++

	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(RECEIVE), topic, timeout, c.targetPeer},
		"",
	)

	fmt.Printf("Qi Zhang, read message %s, %v\n", string(c.sessionID), c.readCounter)

	if r.Status != shim.OK {
		errMsg := fmt.Errorf("failed receiving message [%s][%s]", r.String(), r.Payload)
		fmt.Printf("Fabrice -- %v\n", errMsg)
		return nil, errMsg
	}

	if r.Payload == nil {
		errMsg := fmt.Errorf("failed receiving message [%s][%s]", r.String(), r.Payload)
		fmt.Printf("Fabrice -- %v\n", errMsg)
		return nil, errMsg
	}

	return r.Payload, nil

}

/*
func (c *commSCCMsgConn) Read() (p []byte, err error) {
	p, err = c.RawRead()

	if len(p) < 1 {
		fmt.Printf("Read: missing the counter\n")
	}

	if err != nil {
		return nil, err
	}

	fmt.Printf("Read: receive msg %v; my counter is %v, [%v]\n", p[0], c.readCounter, p[1:])

	c.readCounter++

	return p[1:], nil
}
*/
func (c *commSCCMsgConn) Flush() error {
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
