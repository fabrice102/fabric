/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"

	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"time"
)


const (
	COMM_SCC        = "commscc"
	SEND            = "send"
	RECEIVE         = "receive"
	DEFAULT_TIMEOUT = time.Second * 10
)

type MPCExampleChaincode struct {
}

func (t *MPCExampleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Init")

	return shim.Success(nil)
}



func (t *MPCExampleChaincode) Invoke(s shim.ChaincodeStubInterface) pb.Response {
	stub := CommStub{s}
	fmt.Println("Invoke")
	// Run function arg[0] as master/slave decorations["master"]
	// connecting to decorations["target"] on input
	// decorations["input"]
	decorations := stub.GetDecorations()

	for key, value := range decorations {
		fmt.Printf("Decorations: [%v][%v]\n", key, value)
	}

	//function := string(args[0])
	masterStr := string(decorations["master"])
	master := true
	if masterStr == "false" {
		master = false
	}
	target := string(decorations["target"])
	input := decorations["input"]
	myInput, _ := strconv.ParseInt(string(input), 10, 64)
	fmt.Println("My Input:", myInput)
	fmt.Printf("Decorations: [%s][%v][%s][%s]\n", masterStr, master, string(target), string(input))

	fmt.Println("ex02 Invoke")
	function, args := stub.GetFunctionAndParameters()
	if function == "query" {
		// the old "Query" is now implemented in invoke
		return t.query(stub, args)
	}

	firstPhase := stub.GetTxID()

	resp := stub.Send(input, firstPhase, target)
	if resp.GetStatus() != shim.OK {
		return shim.Error(resp.Message)
	}
	resp = stub.Receive(firstPhase, target)

	otherInput, _ := strconv.ParseInt(string(resp.Payload), 10, 64)
	fmt.Println("Other Input:", otherInput)

	result := (myInput * otherInput) % 499
	fmt.Println("Computed result:", result)
	computedResult := []byte(fmt.Sprintf("%d", result))

	secondPhase := stub.GetTxID() + "||2"

	// Now send to the target what we computed,
	resp = stub.Send(computedResult, secondPhase, target)
	if resp.GetStatus() != shim.OK {
		return shim.Error(resp.Message)
	}
	// and receive what it computed, to make sure it matches.
	resp = stub.Receive(secondPhase, target)
	otherResult, _ := strconv.ParseInt(string(resp.Payload), 10, 64)
	fmt.Println("Other result:", otherResult)

	if result != otherResult {
		return shim.Error(fmt.Sprintf("I computed %d but other side computed %d", result, otherResult))
	}

	return shim.Success([]byte(fmt.Sprintf("Combined number modulo 499 is %d", result)))
}



// query callback representing the query of a chaincode
func (t *MPCExampleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	jsonResp := "{\"Name\":\"" + "A" + "\",\"Amount\":\"" + "100" + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success([]byte(strconv.Itoa(100)))
}

type CommStub struct {
	shim.ChaincodeStubInterface
}

func (stub CommStub) Send(data []byte, session, target string) pb.Response{
	fmt.Println(time.Now(), "Sending", string(data), "to", target)
	return stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(SEND), data, []byte(session), []byte(target)},
		"",
	)
}

func (stub CommStub) Receive(session, source string) pb.Response {
	resp := stub.InvokeChaincode(
		COMM_SCC,
		[][]byte{[]byte(RECEIVE), []byte(session), []byte("5000"), []byte(source)},
		"",
	)
	fmt.Println(time.Now(), "Received", string(resp.Payload), resp.Message, resp.Status)
	return resp
}


func main() {
	err := shim.Start(new(MPCExampleChaincode))
	if err != nil {
		fmt.Printf("Error starting schaincode: %s", err)
	}
}
