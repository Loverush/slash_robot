package utils

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path"
	"slash-robot/abi"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
)

var (
	ChainId            = big.NewInt(714)
	SlashIndicatorAddr = common.HexToAddress("0x0000000000000000000000000000000000001001")
	TokenHubAddr       = common.HexToAddress("0x0000000000000000000000000000000000001004")
	RelayerHubAddr     = common.HexToAddress("0x0000000000000000000000000000000000001006")
	ValidatorSetAddr   = common.HexToAddress("0x0000000000000000000000000000000000001000")
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
		return false, voteData.TargetNumber
	}
	// 2. no vote within the span of other votes
	for height := voteData.TargetNumber - 1; height > voteData.SourceNumber+1; height-- {
		if vote, ok := vrStore.VoteRecord[voteAddr][height]; ok {
			if vote.Data.SourceNumber > voteData.SourceNumber {
				return false, height
			}
		}
	}
	vrStore.set(voteAddr, voteData.TargetNumber, vote)
	return true, 0
}

func ReportVote(vote1, vote2 *types.VoteEnvelope, client *ethclient.Client) {
	var evidence abi.SlashIndicatorFinalityEvidence
	evidence.VoteA = abi.SlashIndicatorVoteData{
		SrcNum:  big.NewInt(int64(vote1.Data.SourceNumber)),
		SrcHash: vote1.Data.SourceHash,
		TarNum:  big.NewInt(int64(vote1.Data.TargetNumber)),
		TarHash: vote1.Data.TargetHash,
		Sig:     vote1.Signature[:],
	}
	evidence.VoteB = abi.SlashIndicatorVoteData{
		SrcNum:  big.NewInt(int64(vote2.Data.SourceNumber)),
		SrcHash: vote2.Data.SourceHash,
		TarNum:  big.NewInt(int64(vote2.Data.TargetNumber)),
		TarHash: vote2.Data.TargetHash,
		Sig:     vote2.Signature[:],
	}
	evidence.VoteAddr = vote1.VoteAddress.Bytes()

	account := SlashAccount
	account.Key, _ = crypto.HexToECDSA(account.RawKey)
	ops, _ := bind.NewKeyedTransactorWithChainID(account.Key, ChainId)
	//ops.GasLimit = 800000
	slashIndicator, _ := abi.NewSlash(SlashIndicatorAddr, client)
	tx, err := slashIndicator.SubmitFinalityViolationEvidence(ops, evidence)
	if err != nil {
		log.Fatal("Report Vote:", err)
	}
	var rc *types.Receipt
	for i := 0; i < 180; i++ {
		rc, err = client.TransactionReceipt(context.Background(), tx.Hash())
		if err == nil && rc.Status != 0 {
			fmt.Println("Report Vote: submit evidence success")
			break
		}
		if rc != nil && rc.Status == 0 {
			log.Fatal("Report Vote: tx failed: ", err, rc)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if rc == nil {
		log.Fatal("Report Vote: submit evidence failed")
	}
}

func TestSlash(vote *types.VoteEnvelope, client *ethclient.Client) {
	keyfile := "./bls/keystore/keystore-wholly-valid-oryx.json"
	keyJSON, err := ioutil.ReadFile(keyfile)
	if err != nil {
		log.Fatal("error read keystore file: ", err)
	}
	keystore := &keymanager.Keystore{}
	if err := json.Unmarshal(keyJSON, keystore); err != nil {
		log.Fatalf("Could not decode keystore file: %v.", err)
	}
	if keystore.Pubkey == "" {
		log.Fatalf(" Missing public key, wrong keystore file.")
	}

	walletDir := "./bls/wallet"
	dirExists, err := wallet.Exists(walletDir)
	if err != nil || !dirExists {
		log.Fatalf("BLS wallet not exists.")
	}

	walletPassword := "password"
	w, err := wallet.OpenWallet(context.Background(), &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: walletPassword,
	})
	if err != nil {
		log.Fatalf("Open BLS wallet failed: %v.", err)
	}
	km, err := w.InitializeKeymanager(context.Background(), iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		log.Fatalf("Initialize key manager failed: %v.", err)
	}
	ikm, ok := km.(*imported.Keymanager)
	if !ok {
		log.Fatalf("Could not assert keymanager interface to concrete type.")
	}

	var fakeVote = &types.VoteEnvelope{
		Data: &types.VoteData{
			SourceNumber: vote.Data.SourceNumber,
			SourceHash:   vote.Data.SourceHash,
			TargetNumber: vote.Data.TargetNumber,
			TargetHash:   common.BytesToHash(common.Hex2Bytes(string(rune(vote.Data.TargetNumber)))),
		},
	}
	voteHash := fakeVote.Data.Hash()
	pubKeys, err := km.FetchValidatingPublicKeys(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()
	signature, err := ikm.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:   pubKeys[0][:],
		SigningRoot: voteHash[:],
	})
	copy(fakeVote.VoteAddress[:], pubKeys[0][:])
	copy(fakeVote.Signature[:], signature.Marshal()[:])

	ReportVote(vote, fakeVote, client)
}

func newBLSPubKey(voteAddr string) types.BLSPublicKey {
	var BLSPubKey types.BLSPublicKey
	voteAddrBytes, _ := hex.DecodeString(voteAddr)
	copy(BLSPubKey[:], voteAddrBytes)
	return BLSPubKey
}

func newBLSSig(sig string) types.BLSSignature {
	var BLSSig types.BLSSignature
	sigBytes, _ := hex.DecodeString(sig)
	copy(BLSSig[:], sigBytes)
	return BLSSig
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
