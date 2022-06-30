package collectutil

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/warrior21st/blockchain-utils/ethutil"
	"github.com/warrior21st/blockchain-utils/tokenutil"
	"github.com/warrior21st/go-utils/commonutil"
)

type CollectTokenParams struct {
	Endpoint     string
	GasPriceGwei float64
	Token        string
	IncomeTo     string
}

func CollectTokensByFile(collectParams *CollectTokenParams, privsFile string, detailSaveFile string) {
	privs := ethutil.ReadPrivateKeys(privsFile)
	CollectTokens(collectParams, privs, detailSaveFile)
}

func CollectTokens(collectParams *CollectTokenParams, privs []string, detailSaveFile string) {

	client, err := ethclient.Dial(collectParams.Endpoint)
	if err != nil {
		panic(err)
	}

	decimals, err := tokenutil.Decimals(client, collectParams.Token)
	if err != nil {
		panic(err)
	}

	tokenSymbol, err := tokenutil.Symbol(client, collectParams.Token)
	if err != nil {
		panic(err)
	}

	total := len(privs)
	for i := 0; i < total; i++ {
		priv := ethutil.HexToECDSAPrivateKey(privs[i])
		addr := ethutil.PubkeyToAddress(&priv.PublicKey)
		balance, _ := tokenutil.BalanceOf(client, collectParams.Token, addr)
		if balance.Cmp(big.NewInt(0)) == 1 {
			ethutil.LogWithTime(fmt.Sprintf("%s checked %s %s", addr, tokenutil.ConvertAmount(balance, decimals), tokenSymbol))
			commonutil.AppendToFile(detailSaveFile, privs[i]+"\n")

			nonce := ethutil.GetNextNonce(client, addr)
			ethutil.LogWithTime(fmt.Sprintf("%s current nonce: %d", addr, nonce))

			txId, _ := tokenutil.Transfer(client, priv, collectParams.Token, collectParams.IncomeTo, balance, nonce, tokenutil.TransferERC20DefaultGas, big.NewInt(int64(collectParams.GasPriceGwei*params.GWei)))
			ethutil.WaitTxReceiptSuccess(client, txId, fmt.Sprintf("income %d %s from %s", tokenutil.ConvertAmount(balance, decimals), tokenSymbol, addr), 0)
		}
		ethutil.LogWithTime(fmt.Sprintf("scan progress %d/%d...", i, total-1))
	}
}

func CollectETHs(collectParams *CollectTokenParams, privs []string, detailSaveFile string) {
	client, err := ethclient.Dial(collectParams.Endpoint)
	if err != nil {
		panic(err)
	}
	tokenSymbol, err := tokenutil.Symbol(client, collectParams.Token)
	if err != nil {
		panic(err)
	}

	chainId := ethutil.GetChainID(client)
	decimals := 18

	total := len(privs)
	for i := 0; i < total; i++ {
		priv := ethutil.HexToECDSAPrivateKey(privs[i])
		addr := ethutil.PubkeyToAddress(&priv.PublicKey)
		balance, _ := tokenutil.BalanceOf(client, collectParams.Token, addr)
		if balance.Cmp(big.NewInt(0)) == 1 {
			ethutil.LogWithTime(fmt.Sprintf("%s checked %s %s", addr, tokenutil.ConvertAmount(balance, int32(decimals)), tokenSymbol))
			commonutil.AppendToFile(detailSaveFile, privs[i]+"\n")

			nonce := ethutil.GetNextNonce(client, addr)
			ethutil.LogWithTime(fmt.Sprintf("%s current nonce: %d", addr, nonce))

			gasPrice := big.NewInt(int64(collectParams.GasPriceGwei * params.GWei))
			gasFee := big.NewInt(0).Mul(big.NewInt(21000), gasPrice)
			if balance.Cmp(gasFee) <= 0 {
				ethutil.LogWithTime(fmt.Sprintf("%s balance less than or equals to gas fee,continue...", addr))
			}
			incomeAmount := balance.Sub(balance, gasFee)
			incomeTx := ethutil.NewTx(nonce, collectParams.IncomeTo, incomeAmount, uint64(21000), gasPrice, nil)
			signedIncomeTx := ethutil.SignTx(priv, incomeTx, chainId)
			txId := ethutil.GetRawTxHash(signedIncomeTx)
			ethutil.SendRawTx(client, signedIncomeTx)

			ethutil.WaitTxReceiptSuccess(client, txId, fmt.Sprintf("income %d %s from %s", tokenutil.ConvertAmount(balance, int32(decimals)), tokenSymbol, addr), 0)
		}
		ethutil.LogWithTime(fmt.Sprintf("income progress %d/%d...", i, total-1))
	}
}
