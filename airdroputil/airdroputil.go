package airdroputil

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
	"github.com/warrior21st/blockchain-utils/ethutil"
	"github.com/warrior21st/blockchain-utils/tokenutil"
	"github.com/warrior21st/go-utils/commonutil"
)

type AirdropParams struct {
	Endpoint        string
	SenderPrv       string
	GasLimit        int64
	GasPriceGwei    float64
	AirdropContract string
	Token           string
	TokenDecimals   int64
	AccountsPerTx   int
}

const AirdropAbi = `[{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"sender","type":"address"},{"indexed":false,"internalType":"address","name":"token","type":"address"},{"indexed":false,"internalType":"uint256","name":"totalAccount","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"totalAmount","type":"uint256"}],"name":"Aidroped","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"caller","type":"address"},{"indexed":false,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"CfoTakedETH","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"caller","type":"address"},{"indexed":false,"internalType":"address","name":"token","type":"address"},{"indexed":false,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"CfoTakedToken","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"inputs":[{"internalType":"address","name":"_admin","type":"address"}],"name":"addAdmin","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address[]","name":"accounts","type":"address[]"},{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"name":"airdropETH","outputs":[],"stateMutability":"payable","type":"function"},{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address[]","name":"accounts","type":"address[]"},{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"name":"airdropToken","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"cfo","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"isAdmin","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"_admin","type":"address"}],"name":"removeAdmin","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"renounceOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"_cfo","type":"address"}],"name":"setCfo","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"to","type":"address"}],"name":"takeAllETH","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"takeAllETHToSelf","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"to","type":"address"}],"name":"takeAllToken","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"takeAllTokenToSelf","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"takeETH","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"takeToken","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

func AirdropTokensByFile(paras *AirdropParams, airdropListFile string) {
	accounts, amounts := ReadAirdropList(airdropListFile, paras.TokenDecimals)
	AirdropTokens(paras, accounts, amounts)
}

func AirdropTokens(paras *AirdropParams, accounts []common.Address, amounts []*big.Int) {
	prv := ethutil.HexToECDSAPrivateKey(paras.SenderPrv)
	sender := ethutil.PubkeyToAddress(&prv.PublicKey)
	gas := uint(paras.GasLimit)
	gasPrice := big.NewInt(int64(math.Floor(paras.GasPriceGwei * params.GWei)))
	// tokenDecimals := paras.TokenDecimals
	airdropContract := ethutil.GetContractAbi(AirdropAbi)

	client, err := ethclient.Dial(paras.Endpoint)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	chainId := ethutil.GetChainID(client)

	ethutil.LogWithTime(fmt.Sprintf("airdrop chainId: %s", chainId.String()))

	totalAccount := len(accounts)
	if totalAccount != len(amounts) {
		panic(errors.New("account length not equals to amounts length"))
	}

	totalAmount := big.NewInt(0)
	for i := 0; i < totalAccount; i++ {
		totalAmount = totalAmount.Add(totalAmount, amounts[i])
	}
	ethutil.LogWithTime(fmt.Sprintf("start airdrop accounts count: %d, totalAmount: %s", totalAccount, tokenutil.ConvertAmount(totalAmount, int32(paras.TokenDecimals))))

	nonce := ethutil.GetNextNonce(client, sender)
	allowanceAmount, err := tokenutil.Allowance(client, paras.Token, sender, paras.AirdropContract)
	if err != nil {
		panic(err)
	}
	if allowanceAmount.Cmp(totalAmount) == -1 {
		txId, err := tokenutil.Approve(client, chainId, prv, paras.Token, paras.AirdropContract, nonce, tokenutil.ApproveERC20DefaultGas, gasPrice)
		if err != nil {
			panic(err)
		}

		ethutil.LogWithTime(fmt.Sprintf("sended approve tx: %s...", txId))
		ethutil.WaitTxReceiptSuccess(client, txId, "approve token for airdrop contract", 0)

		nonce++
	}

	balance, err := tokenutil.BalanceOf(client, paras.Token, sender)
	if err != nil {
		panic(err)
	}
	if balance.Cmp(totalAmount) == -1 {
		panic(errors.New("insufficient sender balance"))
	}

	each := paras.AccountsPerTx
	for i := 0; i < totalAccount; i += each {

		endIndex := i + each
		if endIndex > totalAccount {
			endIndex = totalAccount
		}
		ethutil.LogWithTime("starting airdrop for accounts index: " + strconv.Itoa(i) + " - " + strconv.Itoa(endIndex-1) + "...")
		airdropInputData, err := airdropContract.Pack("airdropToken", common.HexToAddress(paras.Token), accounts[i:endIndex], amounts[i:endIndex])
		if err != nil {
			panic(err)
		}
		airdropTx := ethutil.NewTx(nonce, paras.AirdropContract, big.NewInt(0), uint64(gas), gasPrice, airdropInputData)
		airdropSignedTx := ethutil.SignTx(prv, airdropTx, chainId)
		airdropTxId := ethutil.GetRawTxHash(airdropSignedTx)
		err = ethutil.SendRawTx(client, airdropSignedTx)
		if err != nil {
			panic(err)
		}
		ethutil.LogWithTime(fmt.Sprintf("sended airdrop Tokens tx: %s...", airdropTxId))

		ethutil.WaitTxReceiptSuccess(client, airdropTxId, fmt.Sprintf("airdrop for accounts index: %d - %d / %d", i, endIndex-1, totalAccount-1), 0)

		nonce++
	}
}

func AirdropETHsByFile(paras *AirdropParams, airdropListFile string) {
	accounts, amounts := ReadAirdropList(airdropListFile, paras.TokenDecimals)
	AirdropETHs(paras, accounts, amounts)
}

func AirdropETHs(paras *AirdropParams, accounts []common.Address, amounts []*big.Int) {
	prv := ethutil.HexToECDSAPrivateKey(paras.SenderPrv)
	sender := ethutil.PubkeyToAddress(&prv.PublicKey)
	gas := uint(paras.GasLimit)
	gasPrice := big.NewInt(int64(math.Floor(paras.GasPriceGwei * params.GWei)))
	airdropContract := ethutil.GetContractAbi(AirdropAbi)

	client, err := ethclient.Dial(paras.Endpoint)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	chainId := ethutil.GetChainID(client)
	ethutil.LogWithTime(fmt.Sprintf("airdrop chainId: %s", chainId.String()))

	totalAccount := len(accounts)
	if totalAccount != len(amounts) {
		panic(errors.New("account length not equals to amounts length"))
	}

	totalAmount := big.NewInt(0)
	for i := 0; i < totalAccount; i++ {
		totalAmount = totalAmount.Add(totalAmount, amounts[i])
	}
	ethutil.LogWithTime(fmt.Sprintf("start airdrop accounts count: %d, totalAmount: %s", totalAccount, tokenutil.ConvertAmount(totalAmount, int32(paras.TokenDecimals))))

	balance, err := client.BalanceAt(context.Background(), common.HexToAddress(sender), big.NewInt(rpc.LatestBlockNumber.Int64()))
	if err != nil {
		panic(err)
	}
	if balance.Cmp(totalAmount) == -1 {
		panic(errors.New("insufficient sender balance"))
	}

	nonce := ethutil.GetNextNonce(client, sender)
	each := paras.AccountsPerTx
	for i := 0; i < totalAccount; i += each {

		endIndex := i + each
		if endIndex > totalAccount {
			endIndex = totalAccount
		}
		ethutil.LogWithTime("starting airdrop for accounts index: " + strconv.Itoa(i) + " - " + strconv.Itoa(endIndex-1) + "...")

		periodAmounts := amounts[i:endIndex]
		periodTotalAmount := big.NewInt(0)
		for j := 0; j < len(periodAmounts); j++ {
			periodTotalAmount = periodTotalAmount.Add(periodTotalAmount, periodAmounts[j])
		}

		airdropInputData, err := airdropContract.Pack("airdropETH", accounts[i:endIndex], amounts[i:endIndex])
		if err != nil {
			panic(err)
		}
		airdropTx := ethutil.NewTx(nonce, paras.AirdropContract, periodTotalAmount, uint64(gas), gasPrice, airdropInputData)
		airdropSignedTx := ethutil.SignTx(prv, airdropTx, chainId)
		airdropTxId := ethutil.GetRawTxHash(airdropSignedTx)
		err = ethutil.SendRawTx(client, airdropSignedTx)
		if err != nil {
			panic(err)
		}
		ethutil.LogWithTime(fmt.Sprintf("sended airdrop ETHs tx: %s...", airdropTxId))

		ethutil.WaitTxReceiptSuccess(client, airdropTxId, fmt.Sprintf("airdrop for accounts index: %d - %d / %d", i, endIndex-1, totalAccount-1), 0)

		nonce++
	}
}

func ReadAirdropList(filePath string, tokenDecimals int64) ([]common.Address, []*big.Int) {
	precision := decimal.NewFromFloat(math.Pow(10, float64(tokenDecimals)))
	list := strings.Split(commonutil.ReadFile(filePath), "\n")
	total := len(list)
	accounts := make([]common.Address, total)
	amounts := make([]*big.Int, total)
	totalAmount := decimal.Zero
	for i := 0; i < total; i++ {
		detail := strings.Replace(strings.TrimSpace(list[i]), "\r", "", -1)
		addrStr := strings.TrimSpace(strings.Split(detail, ",")[0])
		amountStr := strings.TrimSpace(strings.Split(detail, ",")[1])
		amountDecimal, err := decimal.NewFromString(amountStr)
		if err != nil {
			panic(err)
		}

		if !common.IsHexAddress(addrStr) {
			panic(fmt.Sprintf("invalid address: index %d", i))
		}
		if amountDecimal.Cmp(decimal.Zero) <= 0 {
			panic(fmt.Sprintf("invalid amount: index %d", i))
		}

		accounts[i] = common.HexToAddress(addrStr)
		amounts[i] = amountDecimal.Mul(precision).BigInt()

		totalAmount = totalAmount.Add(amountDecimal)
	}
	ethutil.LogWithTime("readed address count: " + strconv.Itoa(len(accounts)) + ", total amount: " + totalAmount.String())

	return accounts, amounts
}

func ReadAirdropAddressesOnly(filePath string) []common.Address {
	list := strings.Split(commonutil.ReadFile(filePath), "\n")
	accounts := make([]common.Address, 0)

	for i := 0; i < len(list); i++ {
		detail := strings.TrimSpace(list[i])
		addrStr := strings.Replace(detail, "\r", "", -1)
		addrStr = strings.ToLower(addrStr)
		if !common.IsHexAddress(addrStr) {
			ethutil.LogWithTime(fmt.Sprintf("address index %d invalid", i))
			continue
		}

		accounts = append(accounts, common.HexToAddress(addrStr))
	}

	total := len(accounts)
	ethutil.LogWithTime(fmt.Sprintf("valid addr of total: %d / %d", total, len(list)))

	return accounts
}

func TrimContractAccount(client *ethclient.Client, allAccountsTemp []common.Address) []common.Address {
	allAccounts := make([]common.Address, 0)
	for i := range allAccountsTemp {
		ethutil.LogWithTime(fmt.Sprintf("check address's code length progress %d / %d", i+1, len(allAccountsTemp)))
		b, err := client.CodeAt(context.Background(), allAccountsTemp[i], big.NewInt(rpc.LatestBlockNumber.Int64()))
		if err != nil {
			panic(err)
		}
		if len(b) > 0 {
			ethutil.LogWithTime(fmt.Sprintf("%s is contract address,skip...", allAccountsTemp[i].Hex()))
			continue
		}

		allAccounts = append(allAccounts, allAccountsTemp[i])
	}

	return allAccounts
}

func ReadNFTAirdropAddresssWithAmount(filePath string) (addrs []common.Address, amount []int64) {
	list := strings.Split(commonutil.ReadFile(filePath), "\n")
	accounts := make([]common.Address, 0)
	amounts := make([]int64, 0)
	totalAmount := int64(0)
	for i := 0; i < len(list); i++ {
		detail := strings.Replace(strings.TrimSpace(list[i]), "\r", "", -1)
		arr := strings.Split(detail, ",")
		if commonutil.IsNilOrWhiteSpace(arr[0]) || commonutil.IsNilOrWhiteSpace(arr[1]) {
			panic(fmt.Sprintf("invalid address or amount: index %d", i))
		}

		addrStr := strings.TrimSpace(arr[0])
		amountStr := strings.TrimSpace(arr[1])
		amount := commonutil.ParseInt64(amountStr)

		if !common.IsHexAddress(addrStr) {
			panic(fmt.Sprintf("invalid address: index %d", i))
		}
		if amount <= 0 {
			panic(fmt.Sprintf("invalid amount: index %d", i))
		}

		accounts = append(accounts, common.HexToAddress(addrStr))
		amounts = append(amounts, amount)

		totalAmount = totalAmount + amount
	}
	ethutil.LogWithTime("readed address count: " + strconv.Itoa(len(accounts)) + ", total amount: " + commonutil.Int64ToString(totalAmount))

	return accounts, amounts
}
