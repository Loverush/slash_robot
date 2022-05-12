package utils

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path"
	"slash-robot/abi"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	chainId           = big.NewInt(714)
	slashContractAddr = common.HexToAddress("0x0000000000000000000000000000000000001001")
)

type VotesRecordStore struct {
	VoteRecord map[types.BLSPublicKey]map[uint64]*types.VoteEnvelope
	mu         sync.RWMutex
	FileDir    string
}

type record struct {
	Height uint64
	Vote   *types.VoteEnvelope
}

type VoteData struct {
	SrcNum  *big.Int
	SrcHash common.Hash
	TarNum  *big.Int
	TarHash common.Hash
	Sig     []byte
}

type slashEvidence struct {
	VoteA    *VoteData
	VoteB    *VoteData
	VoteAddr []byte
}

func CheckVote(vote *types.VoteEnvelope, vrStore *VotesRecordStore) (bool, uint64) {
	voteAddr := vote.VoteAddress
	voteData := vote.Data
	// 1. no double vote
	if _, ok := vrStore.VoteRecord[voteAddr][voteData.TargetNumber]; ok {
		delete(vrStore.VoteRecord, voteAddr)
		return false, voteData.TargetNumber
	}
	// 2. no vote within the span of other votes
	for height := voteData.TargetNumber - 1; height > voteData.SourceNumber+1; height-- {
		if vote, ok := vrStore.VoteRecord[voteAddr][height]; ok {
			if vote.Data.SourceNumber > voteData.SourceNumber {
				delete(vrStore.VoteRecord, voteAddr)
				return false, height
			}
		}
	}
	vrStore.set(voteAddr, voteData.TargetNumber, vote)
	return true, 0
}

func ReportVote(vote1, vote2 *types.VoteEnvelope, client *ethclient.Client) {
	var evidence slashEvidence
	evidence.VoteA = &VoteData{
		SrcNum:  big.NewInt(int64(vote1.Data.SourceNumber)),
		SrcHash: vote1.Data.SourceHash,
		TarNum:  big.NewInt(int64(vote1.Data.TargetNumber)),
		TarHash: vote1.Data.TargetHash,
		Sig:     vote1.Signature[:],
	}
	evidence.VoteB = &VoteData{
		SrcNum:  big.NewInt(int64(vote2.Data.SourceNumber)),
		SrcHash: vote2.Data.SourceHash,
		TarNum:  big.NewInt(int64(vote2.Data.TargetNumber)),
		TarHash: vote2.Data.TargetHash,
		Sig:     vote2.Signature[:],
	}
	evidence.VoteAddr = vote1.VoteAddress.Bytes()

	account := SlashAccount
	account.Key, _ = crypto.HexToECDSA(account.RawKey)
	ops, _ := bind.NewKeyedTransactorWithChainID(account.Key, chainId)
	slashInstance, _ := abi.NewContractInstance(slashContractAddr, abi.SlashABI, client)
	_, err := slashInstance.Transact(ops, "submitFinalityViolationEvidence", evidence)
	if err != nil {
		log.Fatal("Report Vote:", err)
	}
}

func NewVotesRecordStore(fileDir string) *VotesRecordStore {
	vrStore := &VotesRecordStore{
		VoteRecord: make(map[types.BLSPublicKey]map[uint64]*types.VoteEnvelope),
		FileDir:    fileDir,
	}
	files, err := ioutil.ReadDir(fileDir)
	if err != nil {
		log.Fatal("VotesRecordStore:", err)
	}
	for _, file := range files {
		if len(file.Name()) != 53 {
			continue
		}
		go vrStore.load(file.Name())
	}
	return vrStore
}

func (vr *VotesRecordStore) set(voteAddr types.BLSPublicKey, height uint64, vote *types.VoteEnvelope) bool {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	if _, ok := vr.VoteRecord[voteAddr]; !ok {
		vr.VoteRecord[voteAddr] = make(map[uint64]*types.VoteEnvelope)
	}
	vr.VoteRecord[voteAddr][height] = vote
	if _, ok := vr.VoteRecord[voteAddr][height-256]; ok {
		delete(vr.VoteRecord[voteAddr], height-256)
	}
	return true
}

func (vr *VotesRecordStore) load(file string) error {
	filePath := path.Join(vr.FileDir, file)
	f, err := os.Open(filePath)
	defer f.Close()
	if err != nil {
		log.Fatal("Err saveLoop VotesRecordStore:", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	var voteAddr types.BLSPublicKey
	copy(voteAddr[:], file[:len(file)-5])
	d := json.NewDecoder(f)
	for err == nil {
		var r record
		if err = d.Decode(&r); err == nil {
			vr.set(voteAddr, r.Height, r.Vote)
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}
