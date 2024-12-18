package main

import (
	"context"
	"database/sql"

	"log"

	_ "github.com/go-sql-driver/mysql" // Import the driver anonymously

	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Connect to MySQL
	dsn := "root:rootpassword@tcp(mysql:3306)/mydb"
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

	//// Example: Interact with MySQL
	//rows, err := db.Query("SELECT * FROM transactions")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer rows.Close()
	//
	//// Print out data from MySQL
	//for rows.Next() {
	//	var tx string
	//	if err := rows.Scan(&tx); err != nil {
	//		log.Fatal(err)
	//	}
	//	fmt.Println(tx)
	//}
	//
	//// Additional Go code for interacting with Ethereum and MySQL...
}
