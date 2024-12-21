package main

import (
	"assignment/api"
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"time"

	"log"

	_ "github.com/go-sql-driver/mysql" // Import the driver anonymously
	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func checkEthConnection() (err error) {
	client, err := getEthClient()
	if err != nil {
		log.Printf("[checkEthConnection][getEthCLient] err: %s", err)
		return
	}
	networkId, err := client.NetworkID(context.Background())
	if err != nil {
		log.Printf("[checkEthConnection] Getting network id failed, err:%s  \n", err)
		return
	}
	log.Printf("[checkEthConnection] Network connected , network id : %d \n", networkId)
	return
}

func getEthClient() (cl *ethclient.Client, err error) {
	link := os.Getenv("INFURA_URL")
	cl, err = ethclient.Dial(link)
	if err != nil {
		return
	}
	return
}

func checkDbConnection() (err error) {
	_, err = getDBClient()
	if err != nil {
		log.Printf("[checkDbConnection][getDbClient] err : %s", err)
		return
	}
	log.Println("[checkDbConnection] DB connected")
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

func setUpDefaultContract() {
	if checkExistDefaultContract() {
		return
	}
	log.Println("Default contract is not found")
	log.Println("Setting up network with default contract")
	client, err := getEthClient()
	if err != nil {
		log.Println("[setUpDefaultContract] connection to node failed", err)
		return
	}
	auth, err := getTreasuryAuth(client)
	if err != nil {
		log.Println("[setUpDefaultContract][getTreasuryAuth] err ")
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
	auth.GasPrice = gasPrice
	address, tranasction, _, err := storage.DeployStorage(auth, client, fromAddress) //api is redirected from api directory from our contract go file
	if err != nil {
		panic(err)
	}
	if err != nil {
		log.Fatalf("Failed to deploy new storage contract: %v", err)
		panic(err)
	}

	fmt.Printf("Contract pending deploy: 0x%x\n", address)
	fmt.Printf("Transaction waiting to be mined: 0x%x\n", tranasction.Hash())
	_, err = bind.WaitDeployed(context.Background(), client, tranasction)
	if err != nil {
		log.Fatalf("Failed to wait for contract to be deployed : %v", err)
	}
	fmt.Println("Contract deployed")
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
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, proceeding with Docker environment variables")
	}
	for {
		err := checkEthConnection()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	for {
		err := checkDbConnection()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	setUpDefaultContract()
	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
	contract := getContract()
	if contract == nil {
		log.Fatalln("NO CONTRACT FOUND")
	}
	client, err := getEthClient()
	auth, err := getTreasuryAuth(client)
	if err != nil {
		log.Println("[setUpDefaultContract][getTreasuryAuth] err ")
	}
	balance, err := contract.BalanceOf(callOpts, auth.From)
	fmt.Printf("Balance is %v\n", balance)
}
