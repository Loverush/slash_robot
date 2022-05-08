package utils

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
)

var address = "0x81F37cc0EcAE1dD1c89D79A98f857563873cFA76"
var privateKey = "de8c0753508570d6bc3aea027a5896401c82fe997d3717d19c785Fbbee128695"

var SlashAccount = ExtAcc{
	RawKey: privateKey,
	Addr:   common.HexToAddress(address),
}

type ExtAcc struct {
	RawKey string
	Key    *ecdsa.PrivateKey
	Addr   common.Address
}
