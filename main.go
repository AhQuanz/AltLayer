package main

import (
	"assignment/api"
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"os"

	"log"

	_ "github.com/go-sql-driver/mysql" // Import the driver anonymously
	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func getEthClient() (cl *ethclient.Client, err error) {
	//Connect to Ethereum Testnet (Geth)
	//link := os.Getenv("INFURA_URL")
	link := "http://localhost:8545"
	cl, err = ethclient.Dial(link)
	if err != nil {
		return
	}
	return
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

func checkExistDefaultContract() bool {
	contractAddress := os.Getenv("CONTRACT_ADDRESS")
	if len(contractAddress) == 0 {
		return false
	}
	address := common.HexToAddress(contractAddress)
	client, err := getEthClient()
	if err != nil {
		log.Println("[checkExistDefaultContract] getEthClient err", err)
	}
	// Get the code at the contract address
	code, err := client.CodeAt(context.Background(), address, nil)
	if err != nil {
		log.Fatalf("Failed to get code at address: %v", err)
	}

	// Check if the contract is deployed
	if len(code) == 0 {
		fmt.Println("No contract found at this address.", address, code)
	} else {
		fmt.Println("Contract found at this address!")
	}
	return len(code) != 0
}

func setUpDefaultContract() {
	if checkExistDefaultContract() {
		return
	}
	client, err := getEthClient()
	if err != nil {
		log.Println("[setUpDefaultContract] connection to node failed", err)
		return
	}
	// Load private key
	//privateKeyHex := os.Getenv("PRIVATE_KEY")
	privateKeyHex := "c9732d3436f13a29caf048977070cf0f58cc7ce88a71cf448071bdfa27176bbb"
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		panic(err)
	}
	// Get the nonce for the account
	fromAddress := auth.From
	//fetch the last use nonce of account
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		panic(err)
	}
	balance, err := client.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Account balance: %s wei\n", balance.String())

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}
	//abi, err := store.StoreMetaData.GetAbi()
	//if err != nil {
	//	log.Panicln("[GetABi]err", err)
	//}
	auth.GasPrice = gasPrice
	address, tranasction, _, err := store.DeployStore(auth, client) //api is redirected from api directory from our contract go file
	// address, tranasction, _, err := bind.DeployContract(auth, *abi, []byte(store.StoreMetaData.Bin), client)
	if err != nil {
		panic(err)
	}
	if err != nil {
		log.Fatalf("Failed to deploy new storage contract: %v", err)
		panic(err)
	}
	fmt.Printf("Contract pending deploy: 0x%x\n", address)
	fmt.Printf("Transaction waiting to be mined: 0x%x\n\n", tranasction.Hash())
}

func getPrivateKey() {
	inPath := "keystore/UTC--2024-12-18T10-12-11.593180336Z--c81786c23e512324ed9b544ad8b7cdfc629e1651"
	outPath := "key.hex"
	//password := os.Getenv("WALLET_PASSWORD")
	password := "123456"
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

func main() {
	getPrivateKey()
	// Load environment variables from .env file (only in local development)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, proceeding with Docker environment variables")
	}
	setUpDefaultContract()

	//// Connect to MySQL
	//db, err := getDBClient()
	//defer db.Close()
	////Example: Interact with MySQL
	//rows, err := db.Query("SELECT username FROM users")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer rows.Close()
	//
	//// Print out data from MySQL
	//for rows.Next() {
	//	var tx struct {
	//		Id       int64
	//		Username string
	//		Password string
	//		Token    string
	//		Role     int64
	//	}
	//	if err := rows.Scan(&tx.Username); err != nil {
	//		log.Fatal(err)
	//	}
	//	fmt.Println(tx.Username)
	//}
}
