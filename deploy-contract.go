package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Connect to the Ethereum client
	log.Println("TESTING")
	log.Println(os.Getenv("INFURA_URL"))
	log.Println("TESTING")
	client, err := ethclient.Dial(os.Getenv("INFURA_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// Load private key
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	// Read ABI and Bytecode from files
	tokenABI, err := os.ReadFile("tokenABI.json")
	abi, err := abi.JSON(strings.NewReader(string(tokenABI)))
	if err != nil {
		log.Fatalf("Failed to read token ABI: %v", err)
	}

	tokenBytecode, err := os.ReadFile("tokenBytecode.bin")
	if err != nil {
		log.Fatalf("Failed to read token bytecode: %v", err)
	}

	// Create an authorized transactor
	chainID := big.NewInt(1)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	// Get the nonce for the account
	fromAddress := auth.From
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to get account nonce: %v", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	// Set gas price and gas limit
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggested gas price: %v", err)
	}
	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(3000000) // Set your desired gas limit

	// Deploy the contract
	address, tx, _, err := bind.DeployContract(auth, abi, tokenBytecode, client)
	if err != nil {
		log.Fatalf("Failed to deploy contract: %v", err)
	}

	fmt.Printf("Contract deployed! Address: %s\nTransaction hash: %s\n", address.Hex(), tx.Hash().Hex())
}
