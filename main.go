package main

import (
	"context"
	"database/sql"
	"fmt"

	"log"

	_ "github.com/go-sql-driver/mysql" // Import the driver anonymously

	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Connect to MySQL
	dsn := "user:password@tcp(localhost:3322)/mydb"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//Connect to Ethereum Testnet (Geth)
	cl, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal(err)
	}
	chainid, err := cl.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("{TESTING}", chainid)

	// Example: Interact with MySQL
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
