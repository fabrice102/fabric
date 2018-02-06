package mpc

import (
	"errors"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

const (
	COMM_SCC        = "commscc"
	SEND            = "send"
	RECEIVE         = "receive"
	DEFAULT_TIMEOUT = time.Second * 240
)

type commSCCConn struct {
	stub       shim.ChaincodeStubInterface
	sessionID  []byte
	targetPeer []byte
	timeout    time.Duration
}

// NewCommSCCConn creates a new connection conn backed by the comm scc
func NewConn(stub shim.ChaincodeStubInterface, sessionID string, targetPeer string, server bool) (Conn, error) {
	conn := &commSCCConn{
		stub:       stub,
		sessionID:  []byte(sessionID),
		targetPeer: []byte(targetPeer),
		timeout:    DEFAULT_TIMEOUT,
	}

	if server {
		/* the server waits for 50 ms for the ack from the client("1").
		 * If ack is not received, server keeps probing (sending "1")
		 * Otherwise, server sends "2" to finish the connection setup
		 */
		conn.timeout = time.Millisecond * 50
		for {
			conn.Write([]byte("0"))

			p := make([]byte, 1)
			_, err := conn.Read(p)
			/*Needs to check whether the err is due to a timeout*/
			if err != nil {
				continue
			} else {
				if string(p) == "1" {
					conn.Write([]byte("2"))
					break
				}
			}
		}

		conn.timeout = DEFAULT_TIMEOUT
	} else {
		p := make([]byte, 1)

		for {
			_, err := conn.Read(p)
			/*Todo: should check  'len == 1'*/
			if err != nil {
				continue
			} else {
				conn.Write([]byte("1"))
				break
			}
		}

		for {
			_, err := conn.Read(p)

			/*Todo: should check  'len == 1'*/
			if err != nil {
				return nil, err
			}

			if string(p) == "0" {
				continue
			} else if string(p) == "2" {
				break
			} else {
				return nil, fmt.Errorf("Unexpected message received by client: not equals to 0 or 2: %v", p)
			}

		}

	}

	return conn, nil
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
