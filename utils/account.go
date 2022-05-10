package utils

import (
	"crypto/ecdsa"
	"slash-robot/params"

	"github.com/ethereum/go-ethereum/common"
)

var SlashAccount = ExtAcc{
	RawKey: params.PrivateKey,
	Addr:   common.HexToAddress(params.Address),
}

type ExtAcc struct {
	RawKey string
	Key    *ecdsa.PrivateKey
	Addr   common.Address
}
