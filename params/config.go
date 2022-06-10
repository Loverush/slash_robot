package params

import "time"

var (
	BSCTestnet = "https://data-seed-prebsc-2-s1.binance.org:8545/"
	BSC        = "https://bsc-dataseed.binance.org/"
	GethWS     = "ws://127.0.0.1:8547"
	GethIpc    = "/server/validator/geth.ipc"

	Address    = "0x91D7deA99716Cbb247E81F1cfB692009164a967E"
	PrivateKey = "dcb154cd2e8e0d5deffbc43562933e02ac3415225f5147cbcd37b380191fbe49"

	RecordFilePath = "./data/"

	UpdateInterval = time.Duration(60 * 1e9)
)
