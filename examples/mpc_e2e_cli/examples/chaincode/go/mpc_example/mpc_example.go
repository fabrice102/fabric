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
	"github.com/hyperledger/fabric/mpc"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type MPCExampleChaincode struct {
}

func (t *MPCExampleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Init")

	return shim.Success(nil)
}

func (t *MPCExampleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Invoke")
	// Run function arg[0] as master/slave decorations["master"]
	// connecting to decorations["target"] on input
	// decorations["input"]
	decorations := stub.GetDecorations()

	//function := string(args[0])
	masterStr := string(decorations["master"])
	master := true
	if masterStr == "0" {
		master = false
	}
	target := string(decorations["target"])
	input := decorations["input"]

	fmt.Printf("Decorations: [%s][%s][%s]\n", masterStr, target, string(input))

	fmt.Println("ex02 Invoke")
	function, args := stub.GetFunctionAndParameters()
	if function == "query" {
		// the old "Query" is now implemtned in invoke
		return t.query(stub, args)
	}

	//// Open channel
	channel := mpc.NewCommSCCChannel(stub)
	if master {
		fmt.Println("master 1")
		// First send, then receive
		err := channel.Send(input, target)
		fmt.Println("master 2")
		if err != nil {
			fmt.Println("master 3")
			return shim.Error(err.Error())
		}

		fmt.Println("master 4")
		res, err := channel.Receive(10)
		if err != nil {
			fmt.Println("master 5")
			return shim.Error(err.Error())
		}
		fmt.Println("master 6")
		fmt.Printf("got [%v] from [%s]", res, target)
	} else {
		// First receive, then send
		fmt.Println("slave 1")
		res, err := channel.Receive(10)
		fmt.Println("slave 2")
		if err != nil {
			fmt.Println("slave 3")
			return shim.Error(err.Error())
		}
		fmt.Println("slave 4")
		fmt.Printf("got [%v] from [%s]", res, target)

		err = channel.Send(input, target)
		if err != nil {
			fmt.Println("slave 5")
			return shim.Error(err.Error())
		}
		fmt.Println("slave 6")
	}

	return shim.Success(nil)
}

// query callback representing the query of a chaincode
func (t *MPCExampleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	jsonResp := "{\"Name\":\"" + "A" + "\",\"Amount\":\"" + "100" + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success([]byte(strconv.Itoa(100)))
}

func main() {
	err := shim.Start(new(MPCExampleChaincode))
	if err != nil {
		fmt.Printf("Error starting schaincode: %s", err)
	}
}
