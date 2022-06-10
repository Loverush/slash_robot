package utils

import (
	"fmt"
	"log"
	"slash-robot/abi"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type checkVote struct {
	vote    *types.VoteEnvelope
	vrStore *VotesRecordStore
}

type evidence struct {
	Vote1 *types.VoteEnvelope
	Vote2 *types.VoteEnvelope
}

var (
	checkVoteList = []checkVote{
		{
			vote: &types.VoteEnvelope{
				VoteAddress: newBLSPubKey("b32d4d46a7127dcc865f0d30f2ee3dcd5983b686f4e3a9202afc8b608652001c9938906ae1ff1417486096e32511f1bc"),
				Data: &types.VoteData{
					SourceNumber: uint64(1),
					SourceHash:   common.BytesToHash(common.Hex2Bytes(string(rune(1)))),
					TargetNumber: uint64(10),
					TargetHash:   common.BytesToHash(common.Hex2Bytes(string(rune(10)))),
				},
			},
			vrStore: &VotesRecordStore{
				VoteRecord: newVoteRecord("b32d4d46a7127dcc865f0d30f2ee3dcd5983b686f4e3a9202afc8b608652001c9938906ae1ff1417486096e32511f1bc", 2, 10),
			},
		},
		{
			vote: &types.VoteEnvelope{
				VoteAddress: newBLSPubKey("b32d4d46a7127dcc865f0d30f2ee3dcd5983b686f4e3a9202afc8b608652001c9938906ae1ff1417486096e32511f1bc"),
				Data: &types.VoteData{
					SourceNumber: uint64(1),
					SourceHash:   common.BytesToHash(common.Hex2Bytes(string(rune(1)))),
					TargetNumber: uint64(10),
					TargetHash:   common.BytesToHash(common.Hex2Bytes(string(rune(10)))),
				},
			},
			vrStore: &VotesRecordStore{
				VoteRecord: newVoteRecord("b32d4d46a7127dcc865f0d30f2ee3dcd5983b686f4e3a9202afc8b608652001c9938906ae1ff1417486096e32511f1bc", 2, 9),
			},
		},
	}
	submitEvidenceList = []evidence{
		{
			Vote1: &types.VoteEnvelope{
				VoteAddress: newBLSPubKey("85e6972fc98cd3c81d64d40e325acfed44365b97a7567a27939c14dbc7512ddcf54cb1284eb637cfa308ae4e00cb5588"),
				Signature:   newBLSSig("942858780b29fcb8ccea7f59edc8849ef804c267086e0718e21a9218621eeb2172141383909ed6b6b751b39d5c07a99e0223f00ea6d53e78d93e2a49423a8d403c55d0e868df08edd6be956e71c0581bc892b298fbe169c700a6fa6d582ef210"),
				Data: &types.VoteData{
					SourceNumber: uint64(750),
					SourceHash:   common.BytesToHash(common.Hex2Bytes("555d45d77921e0f26487706179f73c5f8539744b55147c73a3621366bf809c06")),
					TargetNumber: uint64(760),
					TargetHash:   common.BytesToHash(common.Hex2Bytes("6c8781959ce3621f67c9345da9b5e01ce5113d7b5ae3c6dcd3ca88ad4ed9023f")),
				},
			},
			Vote2: &types.VoteEnvelope{
				VoteAddress: newBLSPubKey("85e6972fc98cd3c81d64d40e325acfed44365b97a7567a27939c14dbc7512ddcf54cb1284eb637cfa308ae4e00cb5588"),
				Signature:   newBLSSig("8684e33ee8cc6dca24858a2c4deb88698755c629cfb6583ff4c75450bd4c4608405f6c7547085407e27c05fdc01275e40ac4c145f3c71dbdb3053b8cedbe91244d6db3e76b94b6ee5c9350d67196545cbac94a0d37e5ccb1ee3ac645a9c00476"),
				Data: &types.VoteData{
					SourceNumber: uint64(751),
					SourceHash:   common.BytesToHash(common.Hex2Bytes("0284d0d1efd5baad5cec3da093e963eeba7827f3d9aacf332c0081393d5cc107")),
					TargetNumber: uint64(759),
					TargetHash:   common.BytesToHash(common.Hex2Bytes("cd4dfcdc99332fd9b96b275acaf3fd8dc79645ccebddb510788b179c260bde66")),
				},
			},
		},
	}
)

func newVoteRecord(voteAddr string, srcNum, tarNum uint64) map[types.BLSPublicKey]map[uint64]*types.VoteEnvelope {
	voteRecord := make(map[types.BLSPublicKey]map[uint64]*types.VoteEnvelope)
	bLSKey := newBLSPubKey(voteAddr)
	voteRecord[bLSKey] = make(map[uint64]*types.VoteEnvelope)
	voteRecord[bLSKey][tarNum] = &types.VoteEnvelope{
		VoteAddress: bLSKey,
		Data: &types.VoteData{
			SourceNumber: srcNum,
			SourceHash:   common.BytesToHash(common.Hex2Bytes(string(rune(srcNum)))),
			TargetNumber: tarNum,
			TargetHash:   common.BytesToHash(common.Hex2Bytes(string(rune(tarNum)))),
		},
	}
	return voteRecord
}

func TestCheckVote(t *testing.T) {
	for _, item := range checkVoteList {
		if flag, _ := CheckVote(item.vote, item.vrStore); flag {
			t.Error("check vote wrong", item)
		}
	}
}

func TestReportVote(t *testing.T) {
	client := GetCurrentClient("geth_ws")
	for _, item := range submitEvidenceList {
		ReportVote(item.Vote1, item.Vote2, client)
	}
}

func TestContractCall(t *testing.T) {
	client := GetCurrentClient("geth_ws")
	account := SlashAccount
	account.Key, _ = crypto.HexToECDSA(account.RawKey)
	validatorSet, _ := abi.NewValidatorset(ValidatorSetAddr, client)

	out1, out2, err := validatorSet.GetLivingValidators(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Error register relayer:", err)
	}
	fmt.Println(out1, out2)
}
