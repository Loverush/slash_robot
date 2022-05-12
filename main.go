package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"slash-robot/params"
	"slash-robot/utils"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func mainLoop(client *ethclient.Client, rpcClient *rpc.Client, vrStore *utils.VotesRecordStore) {
	// Go channel to pipe data from client subscription
	newVoteChannel := make(chan *types.VoteEnvelope)

	// Subscribe to receive one time events for new vote
	_, err := rpcClient.EthSubscribe(
		context.Background(), newVoteChannel, "newVotes",
	)

	if err != nil {
		fmt.Println("error while subscribing: ", err)
	} else {
		fmt.Println("Subscribed to vote pool")
	}

	c := make(chan os.Signal, 0)
	signal.Notify(c)
	for {
		select {
		case vote := <-newVoteChannel:
			//fmt.Println("vote message received:", vote.Data)
			ok, height := checkVote(vote, vrStore)
			if !ok {
				vote2 := vrStore.VoteRecord[vote.VoteAddress][height]
				fmt.Println("--------------bad vote detected!--------------")
				fmt.Println("vote address:", vote.VoteAddress)
				fmt.Println("vote message:", vote2.Data)
				utils.ReportVote(vote, vote2, client)
			}
		case s := <-c:
			if s == os.Interrupt || s == os.Kill {
				for val, record := range vrStore.VoteRecord {
					filePath := path.Join(vrStore.FileDir, hex.EncodeToString(val.Bytes()))
					f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
					if err != nil {
						log.Fatal("Err saveLoop VotesRecordStore:", err)
					}
					e := json.NewEncoder(f)
					if err := e.Encode(record); err != nil {
						log.Fatal("Error saving vrStore: ", err)
					}
					f.Close()
				}
				os.Exit(0)
			}
		}

	}
}

func checkVote(vote *types.VoteEnvelope, vrStore *utils.VotesRecordStore) (bool, uint64) {
	return utils.CheckVote(vote, vrStore)
}

func main() {
	clientEntered := flag.String("client", "geth_ws", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_ws\n\t-geth_ipc")
	flag.Parse()

	rpcClient := utils.InitRPCClient(*clientEntered)
	client := utils.GetCurrentClient(*clientEntered)

	var vrStore = utils.NewVotesRecordStore(params.RecordFile)
	mainLoop(client, rpcClient, vrStore)
}
