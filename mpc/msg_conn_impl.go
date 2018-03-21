package mpc

import (
	"errors"
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
}

// NewCommSCCConn creates a new connection conn backed by the comm scc
func NewMsgConn(stub shim.ChaincodeStubInterface, sessionID string, targetPeer string, server bool) (MsgConn, error) {
	conn := &commSCCMsgConn{
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
		// to do dirty fix to avoid pub-sub issues when it is 50
		conn.timeout = time.Millisecond * 200
		//fmt.Printf("******MASTER starts \n")
		for {
			//fmt.Printf("******MASTER sends '0' \n")
			conn.Write([]byte("0"))

			fmt.Printf("******MASTER tries to read '1' \n")
			p, err := conn.Read()
			/*Needs to check whether the err is due to a timeout*/
			if err != nil {
				fmt.Printf("******MASTER continue tries to read '1'")
				continue
			} else {
				fmt.Printf("******MASTER gets something from read")
				if string(p) == "1" {
					fmt.Printf("******MASTER gets '1' from read, and sends '2' \n")
					conn.Write([]byte("2"))
					break
				}
			}
		}
		fmt.Printf("******MASTER resets timeout to DEFAULT_TIMEOUT \n")
		conn.timeout = DEFAULT_TIMEOUT
	} else {

		fmt.Printf("~~~~~~SLAVE starts \n")
		for {
			fmt.Printf("~~~~~~SLAVE tries to read '0' \n")
			_, err := conn.Read()
			/*Todo: should check  'len == 1'*/
			if err != nil {
				fmt.Printf("~~~~~~SLAVE continue tries to read '0' \n")
				continue
			} else {
				fmt.Printf("~~~~~~SLAVE get something from read, and sends '1' \n")
				conn.Write([]byte("1"))
				break
			}
		}

		for {
			//fmt.Printf("~~~~~~SLAVE tries to read '2' \n")
			p, err := conn.Read()

			/*Todo: should check  'len == 1'*/
			if err != nil {
				//fmt.Printf("~~~~~~SLAVE gets some error from read, exists \n")
				return nil, err
			}

			if string(p) == "0" {
				//fmt.Printf("~~~~~~SLAVE gets '0' from read, continue reading \n")
				continue
			} else if string(p) == "2" {
				//fmt.Printf("~~~~~~SLAVE gets '2' from read, connection established \n")
				break
			} else {
				return nil, fmt.Errorf("Unexpected message received by client: not equals to 0 or 2: %v", p)
			}

		}

	}

	return conn, nil
}

func (c *commSCCMsgConn) Write(data []byte) (n int, err error) {
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

func (c *commSCCMsgConn) Read() (p []byte, err error) {

	timeout := []byte(strconv.FormatInt(c.timeout.Nanoseconds()/int64(1000000), 10))
	r := c.stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(RECEIVE), c.sessionID, timeout, c.targetPeer},
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
