package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
				VoteAddress: newBLSPubKey("b32d4d46a7127dcc865f0d30f2ee3dcd5983b686f4e3a9202afc8b608652001c9938906ae1ff1417486096e32511f1bc"),
				Signature:   newBLSSig("942858780b29fcb8ccea7f59edc8849ef804c267086e0718e21a9218621eeb2172141383909ed6b6b751b39d5c07a99e0223f00ea6d53e78d93e2a49423a8d403c55d0e868df08edd6be956e71c0581bc892b298fbe169c700a6fa6d582ef210"),
				Data: &types.VoteData{
					SourceNumber: uint64(1),
					SourceHash:   common.BytesToHash(common.Hex2Bytes(string(rune(1)))),
					TargetNumber: uint64(10),
					TargetHash:   common.BytesToHash(common.Hex2Bytes(string(rune(10)))),
				},
			},
			Vote2: &types.VoteEnvelope{
				VoteAddress: newBLSPubKey("b32d4d46a7127dcc865f0d30f2ee3dcd5983b686f4e3a9202afc8b608652001c9938906ae1ff1417486096e32511f1bc"),
				Signature:   newBLSSig("8684e33ee8cc6dca24858a2c4deb88698755c629cfb6583ff4c75450bd4c4608405f6c7547085407e27c05fdc01275e40ac4c145f3c71dbdb3053b8cedbe91244d6db3e76b94b6ee5c9350d67196545cbac94a0d37e5ccb1ee3ac645a9c00476"),
				Data: &types.VoteData{
					SourceNumber: uint64(2),
					SourceHash:   common.BytesToHash(common.Hex2Bytes(string(rune(2)))),
					TargetNumber: uint64(9),
					TargetHash:   common.BytesToHash(common.Hex2Bytes(string(rune(9)))),
				},
			},
		},
	}
)

func newBLSPubKey(voteAddr string) types.BLSPublicKey {
	var BLSPubKey types.BLSPublicKey
	copy(BLSPubKey[:], voteAddr)
	return BLSPubKey
}

func newBLSSig(sig string) types.BLSSignature {
	var BLSSig types.BLSSignature
	copy(BLSSig[:], sig)
	return BLSSig
}

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
	client := GetCurrentClient("geth_http")
	for _, item := range submitEvidenceList {
		ReportVote(item.Vote1, item.Vote2, client)
	}
}
