package ethutil

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/warrior21st/go-utils/commonutil"
)

type TxBaseParams struct {
	ChainID  *big.Int
	Nonce    uint64
	Gas      uint64
	GasPrice *big.Int
}

func GetNextNonce(client *ethclient.Client, account string) uint64 {
	nonce, err := client.NonceAt(context.Background(), common.HexToAddress(account), big.NewInt(rpc.LatestBlockNumber.Int64()))
	for err != nil {
		LogWithTime(fmt.Sprintf("get %s nonce err: %s,sleep 1s...", account, err.Error()))
		time.Sleep(time.Second)

		nonce, err = client.NonceAt(context.Background(), common.HexToAddress(account), big.NewInt(rpc.LatestBlockNumber.Int64()))
	}
	LogWithTime(fmt.Sprintf("%s next nonce: %d", account, nonce))

	return nonce
}

func WaitTxReceipt(client *ethclient.Client, txId string, txDesc string, maxQuerySeconds int64) bool {
	timeStart := time.Now().Unix()
	for true {
		receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(txId))
		if receipt == nil {
			if err == nil || err.Error() == "not found" {
				LogWithTime(fmt.Sprintf("waiting %s tx %s confirming...", txDesc, txId))
			} else {
				LogWithTime(fmt.Sprintf("get %s tx %s receipt err: %s...", txDesc, txId, err.Error()))
			}
			time.Sleep(time.Duration(3) * time.Second)
		} else {
			if receipt.Status == 1 {
				break
			} else {
				LogWithTime(txDesc + " tx exec failed")
				return false
			}
		}
		if time.Now().Unix()-timeStart >= maxQuerySeconds {
			LogWithTime(fmt.Sprintf("get receipt of tx %s time out", txId))
			return false
		}
	}

	return true
}

func GetChainID(client *ethclient.Client) *big.Int {
	chainId, err := client.ChainID(context.Background())
	for err != nil {
		LogWithTime(fmt.Sprintf("get chainId error: %s,sleep 1s...", err.Error()))
		chainId, err = client.ChainID(context.Background())
	}

	return chainId
}

func ReadPrivateKeys(filePath string) []string {
	content := commonutil.ReadFile(filePath)
	privContentArr := strings.Split(content, "\n")

	l := int64(len(privContentArr))
	results := make([]string, l)

	for i := int64(0); i < l; i++ {
		results[i] = strings.Split(privContentArr[i], ",")[0]
		results[i] = strings.Replace(results[i], "\r", "", -1)
		results[i] = strings.Replace(results[i], "\t", "", -1)
		if commonutil.IsNilOrWhiteSpace(results[i]) {
			panic(errors.New(fmt.Sprintf("index %d is error address", i)))
		}
	}

	return results
}

func GetBalance(client *ethclient.Client, account string) *big.Int {
	balance, err := client.BalanceAt(context.Background(), common.HexToAddress(account), big.NewInt(rpc.LatestBlockNumber.Int64()))
	for err != nil {
		LogWithTime(fmt.Sprintf("get balance error: %s,sleep 1s...", err.Error()))
		time.Sleep(time.Second)
		balance, err = client.BalanceAt(context.Background(), common.HexToAddress(account), big.NewInt(rpc.LatestBlockNumber.Int64()))
	}

	return balance
}

func LogWithTime(msg string) {
	fmt.Printf("%s %s\n", time.Now().UTC().Add(8*time.Hour).Format("2006-01-02 15:04:05"), msg)
}
