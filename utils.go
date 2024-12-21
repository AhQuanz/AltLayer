package main

import (
	storage "assignment/api"
	"database/sql"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"os"
)

func getPrivateKey() {
	inPath := "keystore/UTC--2024-12-18T10-12-11.593180336Z--c81786c23e512324ed9b544ad8b7cdfc629e1651"
	outPath := "key.hex"
	password := os.Getenv("WALLET_PASSWORD")
	keyjson, e := os.ReadFile(inPath)
	if e != nil {
		panic(e)
	}
	key, e := keystore.DecryptKey(keyjson, password)
	if e != nil {
		panic(e)
	}
	e = crypto.SaveECDSA(outPath, key.PrivateKey)
	if e != nil {
		panic(e)
	}
	fmt.Println("Key saved to:", outPath)
}

func getDBClient() (db *sql.DB, err error) {
	dsn := fmt.Sprintf("user:password@tcp(%s)/mydb", os.Getenv("DB_URL"))
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return
	}
	err = db.Ping()
	return
}

func getContract() (contract *storage.Storage) {
	client, err := getEthClient()
	contractAddress := os.Getenv("CONTRACT_ADDRESS")
	if len(contractAddress) == 0 {
		return
	}
	address := common.HexToAddress(contractAddress)
	contract, err = storage.NewStorage(address, client)
	if err != nil {
		return
	}
	return contract
}

func getEthClient() (cl *ethclient.Client, err error) {
	link := os.Getenv("INFURA_URL")
	cl, err = ethclient.Dial(link)
	if err != nil {
		return
	}
	return
}

func getTreasuryAuth(client *ethclient.Client) (auth *bind.TransactOpts, err error) {
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	auth, err = bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		panic(err)
	}
	return
}
