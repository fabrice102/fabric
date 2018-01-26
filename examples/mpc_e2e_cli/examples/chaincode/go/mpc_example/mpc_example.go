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

	fmt.Printf("Decorations: [%s][%v][%s][%s]\n", masterStr, master, target, string(input))

	fmt.Println("ex02 Invoke")
	function, args := stub.GetFunctionAndParameters()
	if function == "query" {
		// the old "Query" is now implemtned in invoke
		return t.query(stub, args)
	}

	//// Open channel
	channel, err := mpc.NewConn(stub, "session1", target, master)

	if err != nil {
		return shim.Error(err.Error())
	}

	if master {
		fmt.Println("master 1")

		fmt.Println("wait a bit")
		//time.Sleep(time.Second * 10)
		fmt.Println("send")
		// First send, then receive
		_, err := channel.Write(input)
		if err != nil {
			return shim.Error(err.Error())
		}

		fmt.Printf("master 2, err [%s]\n", err)
		if err != nil {
			fmt.Printf("master 3, err [%s]", err)
			return shim.Error(err.Error())
		}
		fmt.Printf("master 4, err [%s]\n", err)

		p := make([]byte, len(input))
		_, err = channel.Read(p)
		if err != nil {
			return shim.Error(err.Error())
		}

		if err != nil {
			fmt.Printf("master 5, err [%s]", err)
			return shim.Error(err.Error())
		}
		fmt.Println("master 6")
		fmt.Printf("got [%v] from [%s]", p, target)
	} else {
		// First receive, then send
		fmt.Println("slave 1")
		p := make([]byte, len(input))
		_, err = channel.Read(p)
		if err != nil {
			return shim.Error(err.Error())
		}

		fmt.Printf("slave 2, err [%v]\n", err)
		if err != nil {
			fmt.Printf("slave 3, err [%s]", err)
			return shim.Error(err.Error())
		}
		fmt.Println("slave 4")
		fmt.Printf("got [%v] from [%s]", p, target)

		_, err = channel.Write(input)
		if err != nil {
			return shim.Error(err.Error())
		}

		if err != nil {
			fmt.Printf("slave 5, err [%s]", err)
			return shim.Error(err.Error())
		}
		fmt.Println("slave 6")
	}
	/*
		msg := make([]byte, 1024000)
		var msg2 []byte
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		log.Printf("start 10*1024000 - ping pong benchmark")
		for i := 0; i < 10; i++ {
			if master {
				_, err = channel.Write(msg)
				_, _ = channel.Read()
			} else {
				msg2, _ = channel.Read()
				_ = channel.Write(msg2)
			}
		}
		log.Printf("stop 10*1024000")
	*/
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
