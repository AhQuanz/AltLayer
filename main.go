package main

import (
	"assignment/api"
	"assignment/src"
	"context"
	"fmt"
	"github.com/gorilla/mux"
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
	client, err := src.GetEthClient()
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
	_, err = src.GetDBClient()
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
	client, err := src.GetEthClient()
	if err != nil {
		log.Println("[checkExistDefaultContract] GetEthClient err", err)
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
	ctx := context.Background()
	log.Println("Default contract is not found")
	log.Println("Setting up network with default contract")
	client, err := src.GetEthClient()
	if err != nil {
		log.Println("[setUpDefaultContract] connection to node failed", err)
		return
	}
	auth, err := src.GetTreasuryAuth(ctx, client)
	if err != nil {
		log.Println("[setUpDefaultContract][getTreasuryAuth] err ")
	}
	address, tranasction, _, err := storage.DeployStorage(auth, client, auth.From)
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
	contractAddress := os.Getenv("CONTRACT_ADDRESS")
	if address.String() != contractAddress {
		fmt.Printf("Please update the docker-compose file`s contract address to be : %s\n", contractAddress)
	}
}

func initRouters() *mux.Router {
	router := mux.NewRouter()
	router.Use(src.AuthMiddleware)
	router.HandleFunc("/submitWithdraw", src.HandleNewWithdraw).Methods("POST")
	router.HandleFunc("/approveWithdraw", src.HandleWithdrawApproval).Methods("POST")
	router.HandleFunc("/login", src.HandleLogin).Methods("GET")
	router.HandleFunc("/checkBalance", src.HandleCheckBalance).Methods("GET")
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
