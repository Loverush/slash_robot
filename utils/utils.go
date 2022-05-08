package utils

import (
	"encoding/json"
	"io"
	"log"
	"math/big"
	"os"
	"slash-robot/abi"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	BLSPublicKeyLength = 48
	BLSSignatureLength = 96
)

type BLSPublicKey [BLSPublicKeyLength]byte
type BLSSignature [BLSSignatureLength]byte
type ValidatorsBitSet uint64

type VoteData struct {
	SourceNumber uint64
	SourceHash   common.Hash
	TargetNumber uint64
	TargetHash   common.Hash
}

type VoteEnvelope struct {
	VoteAddress BLSPublicKey
	Signature   BLSSignature
	Data        *VoteData

	// caches
	hash atomic.Value
}

var (
	chainId           = big.NewInt(714)
	slashContractAddr = common.HexToAddress("0x0000000000000000000000000000000000001001")
)

type VotesRecordStore struct {
	VoteRecord map[BLSPublicKey]map[uint64]*VoteEnvelope
	mu         sync.RWMutex
	file       *os.File
}

type record struct {
	voteAddr BLSPublicKey
	height   uint64
	vote     *VoteEnvelope
}

type slashEvidence struct {
	voteA    *VoteData
	voteB    *VoteData
	voteAddr BLSPublicKey
}

func CheckVote(vote *VoteEnvelope, vrStore *VotesRecordStore) (bool, uint64) {
	voteAddr := vote.VoteAddress
	voteData := vote.Data
	// 1. no double vote
	if _, ok := vrStore.VoteRecord[voteAddr][voteData.TargetNumber]; ok {
		// TODO delete local data
		delete(vrStore.VoteRecord, voteAddr)
		return false, voteData.TargetNumber
	}
	// 2. no vote within the span of other votes
	for height := voteData.TargetNumber - 1; height > voteData.SourceNumber+1; height-- {
		if vote, ok := vrStore.VoteRecord[voteAddr][height]; ok {
			if vote.Data.SourceNumber > voteData.SourceNumber {
				// TODO delete local data
				delete(vrStore.VoteRecord, voteAddr)
				return false, height
			}
		}
	}
	vrStore.Set(voteAddr, voteData.TargetNumber, vote)
	vrStore.save(voteAddr, voteData.TargetNumber, vote)
	return true, 0
}

func ReportVote(vote1, vote2 *VoteEnvelope, client *ethclient.Client) {
	var evidence slashEvidence
	evidence.voteA = vote1.Data
	evidence.voteB = vote2.Data
	evidence.voteAddr = vote1.VoteAddress

	account := SlashAccount
	account.Key, _ = crypto.HexToECDSA(account.RawKey)
	ops, _ := bind.NewKeyedTransactorWithChainID(account.Key, chainId)
	slashInstance, _ := abi.NewContractInstance(slashContractAddr, abi.SlashABI, client)
	_, err := slashInstance.Transact(ops, "submitFinalityViolationEvidence", evidence)
	if err != nil {
		return
	}
}

func NewVotesRecordStore(filename string) *VotesRecordStore {
	s := &VotesRecordStore{VoteRecord: make(map[BLSPublicKey]map[uint64]*VoteEnvelope)}
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("VotesRecordStore:", err)
	}
	s.file = f
	return s
}

func (vr *VotesRecordStore) save(voteAddr BLSPublicKey, height uint64, vote *VoteEnvelope) error {
	e := json.NewEncoder(vr.file)
	return e.Encode(record{voteAddr, height, vote})
}

func (vr *VotesRecordStore) Set(voteAddr BLSPublicKey, height uint64, vote *VoteEnvelope) bool {
	vr.VoteRecord[voteAddr][height] = vote
	return true
}

func (vr *VotesRecordStore) Load() error {
	if _, err := vr.file.Seek(0, 0); err != nil {
		return err
	}
	d := json.NewDecoder(vr.file)
	var err error
	for err == nil {
		var r record
		if err = d.Decode(&r); err == nil {
			vr.Set(r.voteAddr, r.height, r.vote)
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}
