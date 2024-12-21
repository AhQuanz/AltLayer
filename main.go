package main

import (
	"assignment/api"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"math/big"
	"net/http"
	"os"
	"time"

	"log"

	_ "github.com/go-sql-driver/mysql" // Import the driver anonymously
	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
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

func checkDbConnection() (err error) {
	_, err = getDBClient()
	if err != nil {
		log.Printf("[checkDbConnection][getDbClient] err : %s", err)
		return
	}
	log.Println("[checkDbConnection] DB connected")
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

func initRouters() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/withdraw", handleWithdraw).Methods("POST")
	router.HandleFunc("/login", handleLogin).Methods("GET")
	return router
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
	port := os.Getenv("SERVER_PORT")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: initRouters(),
	}
	fmt.Printf("Server started at port %s\n", port)
	srv.ListenAndServe()
}
