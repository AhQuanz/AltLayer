package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"strings"

	"log"

	_ "github.com/go-sql-driver/mysql" // Import the driver anonymously
	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func getEthClient() (cl *ethclient.Client, err error) {
	//Connect to Ethereum Testnet (Geth)
	cl, err = ethclient.Dial(os.Getenv("INFURA_URL"))
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
		return true
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
		fmt.Println("No contract found at this address.", address , code)
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
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("{TESTING}", chainID)
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
	auth.GasLimit = uint64(8000000)
	initialSupply := big.NewInt(1000000) // 1 million tokens, for example

	// Deploy the contract
	address, tx, _, err := bind.DeployContract(auth, abi, tokenBytecode, client, initialSupply)
	if err != nil {
		log.Fatalf("Failed to deploy contract: %v", err)
	}

	fmt.Printf("Contract deployed! Address: %s\nTransaction hash: %s\n", address.Hex(), tx.Hash().Hex())
}

func getPrivateKey () {	
	inPath:="keystore/UTC--2024-12-18T10-12-11.593180336Z--c81786c23e512324ed9b544ad8b7cdfc629e1651"
	outPath:="key.hex"
	password:=os.Getenv("WALLET_PASSWORD")
	keyjson,e:=os.ReadFile(inPath); if e != nil { panic(e) }
	key,e:=keystore.DecryptKey(keyjson,password); if e != nil { panic(e) }
	e=crypto.SaveECDSA(outPath,key.PrivateKey); if e!=nil { panic(e) }
	fmt.Println("Key saved to:",outPath)
}

func main() {
	// Load environment variables from .env file (only in local development)
	if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, proceeding with Docker environment variables")
	}
	setUpDefaultContract()

	// Connect to MySQL
	db, err := getDBClient()
	defer db.Close()
	//Example: Interact with MySQL
	rows, err := db.Query("SELECT username FROM users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Print out data from MySQL
	for rows.Next() {
		var tx struct {
			Id       int64
			Username string
			Password string
			Token    string
			Role     int64
		}
		if err := rows.Scan(&tx.Username); err != nil {
			log.Fatal(err)
		}
		fmt.Println(tx.Username)
	}
}
