package main

import (
	"context"
	"flag"
	"fmt"
	"slash-robot/utils"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const recordFile = "./data/voteRecord.json"

func mainLoop(client *ethclient.Client, rpcClient *rpc.Client, vrStore *utils.VotesRecordStore) {
	// Go channel to pipe data from client subscription
	newVoteChannel := make(chan *utils.VoteEnvelope)

	// Subscribe to receive one time events for new vote
	_, err := rpcClient.EthSubscribe(
		context.Background(), newVoteChannel, "newVotes",
	)

	if err != nil {
		fmt.Println("error while subscribing: ", err)
	}
	fmt.Println("Subscribed to vote pool")

	for {
		vote := <-newVoteChannel
		ok, height := checkVote(vote, vrStore)
		if !ok {
			vote2 := vrStore.VoteRecord[vote.VoteAddress][height]
			fmt.Println("bad vote detected")
			fmt.Println("vote address:")
			fmt.Println(vote.VoteAddress)
			fmt.Println("vote message:")
			fmt.Println(vote.Data)
			fmt.Println(vote2)
			utils.ReportVote(vote, vote2, client)
		}
	}
}

func checkVote(vote *utils.VoteEnvelope, vrStore *utils.VotesRecordStore) (bool, uint64) {
	return utils.CheckVote(vote, vrStore)
}

func main() {
	clientEntered := flag.String("client", "geth_http", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_http\n\t-geth_ipc")
	flag.Parse()

	rpcClient := utils.InitRPCClient(*clientEntered)
	client := utils.GetCurrentClient(*clientEntered)

	var vrStore = utils.NewVotesRecordStore(recordFile)
	err := vrStore.Load()
	if err != nil {
		fmt.Println(err)
		return
	}
	mainLoop(client, rpcClient, vrStore)
}
