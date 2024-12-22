package src

import (
	storage "assignment/api"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/golang-jwt/jwt/v4"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"
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

func GetDBClient() (db *sql.DB, err error) {
	dsn := fmt.Sprintf("user:password@tcp(%s)/mydb", os.Getenv("DB_URL"))
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return
	}
	err = db.Ping()
	return
}

func getContract() (contract *storage.Storage) {
	client, err := GetEthClient()
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

func GetEthClient() (cl *ethclient.Client, err error) {
	link := os.Getenv("INFURA_URL")
	cl, err = ethclient.Dial(link)
	if err != nil {
		return
	}
	return
}

func GetTreasuryAuth(ctx context.Context, client *ethclient.Client) (auth *bind.TransactOpts, err error) {
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
	nonce, err := client.PendingNonceAt(ctx, auth.From)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	auth.GasPrice = gasPrice
	return
}

func CheckTokenExpiry(tokenStr string) (isExpired bool, isValid bool) {
	if len(tokenStr) == 0 {
		return false, false
	}
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	isExpired = errors.Is(err, jwt.ErrTokenExpired)
	isValid = token.Valid
	return
}

func GenerateToken(username string) (tokenStr string, err error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	tokenStr, err = token.SignedString([]byte(secret))
	return
}

func CheckDBExecErr() {
	
}

func CheckDBErr(w *http.ResponseWriter, err error, errMsg string) {
	if errors.Is(err, sql.ErrNoRows) {
		(*w).WriteHeader(200)
		(*w).Write([]byte(errMsg))
		return
	} else if err != nil {
		(*w).WriteHeader(200)
		(*w).Write([]byte("DB Error"))
		return
	}
	return
}
