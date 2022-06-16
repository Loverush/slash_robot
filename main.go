package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"path"
	"slash-robot/abi"
	"slash-robot/params"
	"slash-robot/utils"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func voteMonitorLoop(client *ethclient.Client, vrStore *utils.VotesRecordStore) {
	newVoteChannel := make(chan *types.VoteEnvelope)
	sub, err := client.SubscribeNewVotes(context.Background(), newVoteChannel)
	defer sub.Unsubscribe()

	if err != nil {
		log.Fatal("error while subscribing new vote: ", err)
	} else {
		fmt.Println("Subscribed to new vote")
	}

	c := make(chan os.Signal, 0)
	signal.Notify(c)
	//var startNum uint64 = 0
	for {
		select {
		case vote := <-newVoteChannel:
			//if startNum == 0 {
			//	startNum = vote.Data.SourceNumber
			//}
			//if vote.Data.SourceNumber == startNum+5 {
			//	fmt.Println("test slash")
			//	utils.TestSlash(vote, client)
			//}
			fmt.Println(vote.Data.TargetNumber, vote.VoteAddress[0])
			ok, height := utils.CheckVote(vote, vrStore)
			if !ok {
				vote2 := vrStore.VoteRecord[vote.VoteAddress][height]
				fmt.Println("--------------bad vote detected!--------------")
				fmt.Println("vote address:", hex.EncodeToString(vote.VoteAddress.Bytes()))
				utils.ReportVote(vote, vote2, client)
				delete(vrStore.VoteRecord, vote.VoteAddress)
			}
		case s := <-c:
			if s == os.Interrupt || s == os.Kill {
				if _, err := os.Stat(vrStore.FileDir); os.IsNotExist(err) {
					err := os.MkdirAll(vrStore.FileDir, os.ModePerm)
					if err != nil {
						log.Fatal("Error create data dir:", err)
					}
				}
				for val, record := range vrStore.VoteRecord {
					filePath := path.Join(vrStore.FileDir, hex.EncodeToString(val.Bytes()))
					f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
					if err != nil {
						log.Fatal("Error saveLoop VotesRecordStore:", err)
					}
					e := json.NewEncoder(f)
					if err := e.Encode(record); err != nil {
						log.Fatal("Error saving vrStore:", err)
					}
					_ = f.Close()
				}
				os.Exit(0)
			}
		}

	}
}

func finalizedHeaderMonitorLoop(client *ethclient.Client) {
	newFinalizedHeaderChannel := make(chan *types.Header)
	sub, err := client.SubscribeNewFinalizedHeader(context.Background(), newFinalizedHeaderChannel)
	defer sub.Unsubscribe()

	if err != nil {
		log.Fatal("error while subscribing finalized header: ", err)
	} else {
		fmt.Println("Subscribed to finalized header")
	}

	var preFinalizedHeight uint64
	var finalizedHeights []uint64
	for {
		header := <-newFinalizedHeaderChannel
		if height := header.Number.Uint64(); height >= preFinalizedHeight {
			preFinalizedHeight = height
			finalizedHeights = append(finalizedHeights, height)
		} else {
			log.Fatal("Finalized height decline: ", finalizedHeights[len(finalizedHeights)-10:])
		}
	}
}

func registerRelayer(client *ethclient.Client) {
	account := utils.SlashAccount
	account.Key, _ = crypto.HexToECDSA(account.RawKey)
	relayerHub, _ := abi.NewRelayerhub(utils.RelayerHubAddr, client)

	out, err := relayerHub.IsRelayer(&bind.CallOpts{}, account.Addr)
	if err != nil {
		log.Fatal("Error register relayer:", err)
	}

	if !out {
		ops, _ := bind.NewKeyedTransactorWithChainID(account.Key, utils.ChainId)
		ops.Value = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100))
		tx, err := relayerHub.Register(ops)
		if err != nil {
			log.Fatal("Error register relayer:", err)
		}
		var rc *types.Receipt
		for i := 0; i < 180; i++ {
			rc, err = client.TransactionReceipt(context.Background(), tx.Hash())
			if err == nil && rc.Status != 0 {
				break
			} else if rc != nil && rc.Status == 0 {
				log.Fatal("Register relayer failed")
			}
			time.Sleep(100 * time.Millisecond)
		}
		if rc == nil {
			log.Fatal("Register relayer failed")
		}
	}
}

func monitorHeader(client *ethclient.Client) {
	newHeadChannel := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), newHeadChannel)
	defer sub.Unsubscribe()

	if err != nil {
		log.Fatal("Error while subscribing new head: ", err)
	} else {
		fmt.Println("Subscribed to new head")
	}

	c := make(chan os.Signal, 0)
	signal.Notify(c)

	ticker := time.NewTicker(params.UpdateInterval)
	defer ticker.Stop()

	type CountReocord struct {
		Record  []int
		Average int
	}
	cr := &CountReocord{Record: make([]int, 0), Average: 0}
	count := 0
	for {
		select {
		case head := <-newHeadChannel:
			fmt.Println("New head at height:", head.Number.Uint64())
			count += 1
		case <-ticker.C:
			fmt.Printf("\n--------%d Heads Received Last Minute--------\n\n", count)
			cr.Record = append(cr.Record, count)
			cr.Average += count
			count = 0
		case s := <-c:
			if s == os.Interrupt || s == os.Kill {
				cr.Average = cr.Average / len(cr.Record)
				filePath := path.Join("./data/", time.Now().String()+".json")
				f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
				if err != nil {
					log.Fatal("Error saving countRecord:", err)
				}
				e := json.NewEncoder(f)
				if err := e.Encode(cr); err != nil {
					log.Fatal("Error saving countRecord:", err)
				}
				_ = f.Close()
				os.Exit(0)
			}
		}

	}
}

func callContract(client *ethclient.Client) {
	account := utils.SlashAccount
	account.Key, _ = crypto.HexToECDSA(account.RawKey)
	tokenHub, _ := abi.NewTokenhub(utils.TokenHubAddr, client)

	ops, _ := bind.NewKeyedTransactorWithChainID(account.Key, utils.ChainId)
	ops.Value = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2))
	tx, err := tokenHub.TransferOut(
		ops,
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0xC0853E5C6b9Fa64F05D1eAc460a8beA48cDA9d63"),
		big.NewInt(1e18),
		150000+uint64(time.Now().Unix()))
	if err != nil {
		log.Fatal("Error call contract:", err)
	}
	var rc *types.Receipt
	for i := 0; i < 180; i++ {
		rc, err = client.TransactionReceipt(context.Background(), tx.Hash())
		if err == nil && rc.Status != 0 {
			break
		} else if rc != nil && rc.Status == 0 {
			log.Fatal("Error call contract", rc.Logs)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if rc == nil {
		log.Fatal("Transaction failed")
	} else {
		fmt.Println("Transaction success")
	}

}

func main() {
	clientEntered := flag.String("client", "geth_ws", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_ws\n\t-geth_ipc")
	flag.Parse()

	client := utils.GetCurrentClient(*clientEntered)
	defer client.Close()

	registerRelayer(client)

	var vrStore = utils.NewVotesRecordStore(params.RecordFilePath)
	voteMonitorLoop(client, vrStore)

	//finalizedHeaderMonitorLoop(client)

	//callContract(client)
}
