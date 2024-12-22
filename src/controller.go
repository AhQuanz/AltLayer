package src

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"log"
	"math/big"
	"net/http"
	"strconv"
)

type NewWithdrawResponse struct {
	ClaimId int64 `json:"ClaimId"`
}

func HandleNewWithdraw(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Token")
	db, err := GetDBClient()
	if err != nil {
		w.WriteHeader(200)
		w.Write([]byte("Could not connect to db"))
		return
	}
	var user user
	row := db.QueryRow("SELECT id, account_number FROM users where token = ?", token)
	err = row.Scan(&user.Id, &user.AccountNumber)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(200)
		w.Write([]byte("User cannot be found"))
		return
	} else if err != nil {
		log.Printf("[HandleNewWithdraw][Get user] err : %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("DB Error"))
		return
	}
	err = r.ParseMultipartForm(10 << 20)
	amountStr := r.FormValue("Amount")
	bigNum := big.NewInt(0)
	_, isSuccess := bigNum.SetString(amountStr, 10)
	if !isSuccess {
		log.Printf("[HandleNewWithdraw]Amount parse err: %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("Invalid Amount"))
		return
	}
	result, err := db.Exec("INSERT INTO withdraw_claim(claim_user_id, amount) VALUES (?,?)", user.Id, bigNum.String())
	if err != nil {
		log.Printf("[HandleNewWithdraw] DB insert err: %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("Claim request failed to insert into DB"))
		return
	}
	claimId, _ := result.LastInsertId()
	if claimId == 0 {
		log.Printf("[HandleNewWithdraw] DB insert failed")
		w.WriteHeader(200)
		w.Write([]byte("Claim request failed to insert into DB"))
		return
	}
	response := NewWithdrawResponse{
		ClaimId: claimId,
	}
	json.NewEncoder(w).Encode(response)
}

type user struct {
	Id            int
	Token         string
	AccountNumber string
	Role          int64
}

type LoginResponse struct {
	Token string `json:"token"`
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("Username")
	password := r.URL.Query().Get("Password")
	db, err := GetDBClient()
	if err != nil {
		w.WriteHeader(200)
		w.Write([]byte("Could not connect to db"))
		return
	}
	var user user
	row := db.QueryRow("SELECT id , token FROM users where username = ? AND password = ?", username, password)
	err = row.Scan(&user.Id, &user.Token)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(200)
		w.Write([]byte("Could not find specified user"))
		return
	} else if err != nil {
		log.Printf("[HandleLogin][Get user] err : %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("DB Error"))
		return
	}
	isExpired, isValid := CheckTokenExpiry(user.Token)
	if !isExpired && isValid {
		w.WriteHeader(200)
		response := LoginResponse{
			Token: user.Token,
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	tokenString, err := GenerateToken(username)
	if err != nil {
		log.Printf("[HandleLogin][JWT SignedString] err : %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("Could not generate token"))
		return
	}
	result, err := db.Exec("UPDATE users SET token = ? WHERE id = ?", tokenString, user.Id)
	if err != nil {
		log.Printf("[HandleLogin][Update user] err : %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("Token not updated in DB"))
		return
	}
	numRows, _ := result.RowsAffected()
	if numRows == 0 {
		log.Printf("[HandleLogin][Update user] no rows affected user id : %d\n", user.Id)
		w.WriteHeader(200)
		w.Write([]byte("Token not updated in DB"))
		return
	}
	w.WriteHeader(200)
	response := LoginResponse{
		Token: tokenString,
	}
	json.NewEncoder(w).Encode(response)
	return
}

type withdrawalClaim struct {
	Id          int
	ClaimStatus int
	Amount      string
}

func updateWithdrawalStatus(db *sql.DB, claimId, status int64) (result sql.Result, err error) {
	result, err = db.Exec(`
			UPDATE withdraw_claim
			SET status = ? 
			WHERE id = ?
		`, claimId, status)
	return
}

func SubmitWithdrawalToChain(accountNumber string, withdrawalAmount *big.Int) (updateStatus int64) {
	ctx := context.Background()
	client, err := GetEthClient()
	if err != nil {
		log.Println("[SubmitWithdrawalToChain] connection to node failed", err)
		return
	}
	auth, err := GetTreasuryAuth(ctx, client)
	if err != nil {
		log.Println("[setUpDefaultContract][getTreasuryAuth] err ")
	}
	contract := getContract()
	toAddress := common.HexToAddress(accountNumber)
	tx, err := contract.Withdraw(auth, toAddress, withdrawalAmount)
	if err != nil {
		log.Fatalf("Failed to wait for contract to be deployed : %v", err)
	}
	fmt.Printf("Transaction waiting to be mined: 0x%x\n", tx.Hash())
	receipt, err := bind.WaitMined(ctx, client, tx)
	fmt.Println("Transaction minted")
	updateStatus = TX_CHAIN_ERROR
	if receipt.Status != types.ReceiptStatusSuccessful {
		updateStatus = TX_SUCCESS_ON_CHAIN
	}
	return updateStatus
}

func HandleWithdrawApproval(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Token")
	db, err := GetDBClient()
	if err != nil {
		w.WriteHeader(200)
		w.Write([]byte("Could not connect to db"))
		return
	}
	tx, err := db.Begin()
	defer db.Close()
	var user user
	// Retrieve User
	{
		row := db.QueryRow("SELECT id , role FROM users where token = ?", token)
		err = row.Scan(&user.Id, &user.Role)
		if err != nil {
			log.Printf("[HandleNewWithdraw][Get user] err : %v\n", err)
			CheckDBErr(&w, err, "Cannot find specific user")
			return
		}
	}
	if user.Role != ROLE_MANAGER {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("User has no permission to approve withdrawal claims"))
		return
	}
	err = r.ParseMultipartForm(10 << 20)
	claimIdStr := r.FormValue("ClaimId")
	claimId, err := strconv.ParseInt(claimIdStr, 10, 64)
	var withdrawal withdrawalClaim
	// Retrieve Withdraw Claim
	{
		withdrawRow := db.QueryRow("SELECT id, amount FROM withdraw_claim where id = ?", claimId)
		err = withdrawRow.Scan(&withdrawal.Id, &withdrawal.Amount)
		if err != nil {
			log.Printf("[HandleApproval][Get claim] err : %v\n", err)
			CheckDBErr(&w, err, fmt.Sprintf("Withdrawal Claim of ID : %d cannot be found", claimId))
			return
		}
	}
	// Retrieve withdraw claim approval records
	var approvalRes struct {
		HasAlreadyApproved bool
		TotalApproval      int64
	}
	{
		approvalRow := db.QueryRow(
			`	
		SELECT 
    	MAX(approve_manager_id = ?) AS has_already_approved,
		COUNT(*) AS total_approval
		FROM withdraw_claims_approval WHERE claim_id = ?
		GROUP BY claim_id`, user.Id, claimId)
		err = approvalRow.Scan(&approvalRes.HasAlreadyApproved, &approvalRes.TotalApproval)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("[HandleApproval][Get claim approvals] err : %v\n", err)
			w.WriteHeader(200)
			w.Write([]byte("DB ERROR"))
			return
		}
	}
	if approvalRes.HasAlreadyApproved {
		w.WriteHeader(200)
		w.Write([]byte("Withdrawal Claim has been approved by you already"))
		return
	}
	// Insert approval
	{
		result, err := db.Exec(`
		INSERT INTO withdraw_claims_approval(claim_id, approve_manager_id)
		VALUES(? , ?)
	`, claimId, user.Id)
		if err != nil {
			log.Printf("[HandleWithdrawApproval] DB insert err: %v\n", err)
			w.WriteHeader(200)
			w.Write([]byte("Claim request failed to insert into DB"))
			return
		}
		approvalId, _ := result.LastInsertId()
		if approvalId == 0 {
			log.Printf("[HandleWithdrawApproval] DB insert failed")
			w.WriteHeader(200)
			w.Write([]byte("Claim request failed to insert into DB"))
			return
		}
	}
	if approvalRes.TotalApproval == 0 {
		updateWithdrawalRes, updateWithdrawlErr := updateWithdrawalStatus(db, claimId, TX_APPROVAL_IN_PROCESS)
		if updateWithdrawlErr != nil {
			CheckDBErr(&w, err, "Update withdrawal claim status failed")
		}
		numRows, _ := updateWithdrawalRes.RowsAffected()
		if numRows == 0 {
			log.Printf("[HandleLogin][Update user] no rows affected user id : %d\n", user.Id)
			w.WriteHeader(200)
			w.Write([]byte("No withdrawal claim is updated"))
			tx.Rollback()
			return
		}
	}
	approvalRes.TotalApproval += 1
	if approvalRes.TotalApproval != 2 {
		w.WriteHeader(200)
		w.Write([]byte("Withdrawal claim approved"))
		return
	}
	amountBig := big.NewInt(0)
	_, isSuccess := amountBig.SetString(withdrawal.Amount, 10)
	if !isSuccess {
		w.WriteHeader(200)
		w.Write([]byte("Claim amount parsing error"))
		return
	}
	_ = SubmitWithdrawalToChain(user.AccountNumber, amountBig)
	tx.Commit()
}
