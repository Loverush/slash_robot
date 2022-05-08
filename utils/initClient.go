package utils

import (
	"fmt"
	"log"
	"reflect"
	"slash-robot/params"
	"unsafe"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	bscTestnet = params.BSCTestnet
	bsc        = params.BSC
	gethHttp   = params.GethHttp
	gethIpc    = params.GethIpc
)

func GetCurrentClient(clientEntered string) *ethclient.Client {
	var clientType string
	switch clientEntered {
	case "bsc_testnet":
		clientType = bscTestnet
	case "bsc":
		clientType = bsc
	case "geth_ipc":
		clientType = gethIpc
	default:
		clientType = gethHttp
	}

	client, err := ethclient.Dial(clientType)

	if err != nil {
		fmt.Println("Error connecting to client", clientType)
		log.Fatalln(err)
	} else {
		fmt.Println("Successfully connected to ", clientType)
	}

	return client
}

func InitRPCClient(_ClientEntered string) *rpc.Client {
	clientEntered := _ClientEntered
	var clientValue reflect.Value
	clientValue = reflect.ValueOf(GetCurrentClient(clientEntered)).Elem()
	fieldStruct := clientValue.FieldByName("c")
	clientPointer := reflect.NewAt(fieldStruct.Type(), unsafe.Pointer(fieldStruct.UnsafeAddr())).Elem()
	finalClient, _ := clientPointer.Interface().(*rpc.Client)
	return finalClient
}
