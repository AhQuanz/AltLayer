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
	var user User
	row := db.QueryRow("SELECT id, account_number FROM users where token = ?", token)
	err = row.Scan(&user.Id, &user.AccountNumber)
	if err != nil {
		log.Println("[HandleNewWithdraw] Query user error")
		CheckDBErr(&w, err, "User cannot be found")
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
	hasEnoughBalance, err := HasEnoughTreasuryBalance(bigNum)
	if err != nil {
		log.Printf("[HandleNewWithdraw][HasEnoughTreasuryBalance] err: %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("Could not connect to chain to check treasury balance"))
		return
	}
	if !hasEnoughBalance {
		w.WriteHeader(200)
		w.Write([]byte("Treasury does not have enough balance"))
		return
	}
	result, err := db.Exec("INSERT INTO withdraw_claim(claim_user_id, amount) VALUES (?,?)", user.Id, bigNum.String())
	if err != nil {
		CheckDBExecLastIdErr(result, &w, err, "Claim request failed to insert into DB")
		return
	}
	claimId, _ := result.LastInsertId()
	response := NewWithdrawResponse{
		ClaimId: claimId,
	}
	json.NewEncoder(w).Encode(response)
}

type User struct {
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
	var user User
	row := db.QueryRow("SELECT id , token FROM users where username = ? AND password = ?", username, password)
	err = row.Scan(&user.Id, &user.Token)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(200)
		w.Write([]byte("Could not find specified User"))
		return
	} else if err != nil {
		log.Printf("[HandleLogin][Get User] err : %v\n", err)
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
		log.Printf("[HandleLogin][Update User] err : %v\n", err)
		w.WriteHeader(200)
		w.Write([]byte("Token not updated in DB"))
		return
	}
	numRows, _ := result.RowsAffected()
	if numRows == 0 {
		log.Printf("[HandleLogin][Update User] no rows affected User id : %d\n", user.Id)
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
	Id          int64
	ClaimStatus int64
	Amount      string
	ClaimUserId int64
}

func updateWithdrawalStatus(tx *sql.Tx, claimId, status int64) (result sql.Result, err error) {
	result, err = tx.Exec(`
			UPDATE withdraw_claim
			SET claim_status = ? 
			WHERE id = ?
		`, status, claimId)
	return
}

func updateWithdrawalStatusAndTxHash(tx *sql.Tx, claimId, status int64, txHash string) (result sql.Result, err error) {
	result, err = tx.Exec(`
			UPDATE withdraw_claim
			SET claim_status = ? , transaction_hash = ?
			WHERE id = ?
		`, status, txHash, claimId)
	return
}

func RetryFailedTransactions() {
	log.Println("RUNNING RetryFailedTransactions")
	db, err := GetDBClient()
	if err != nil {
		log.Printf("[RetryFailedTransactions] GetDBClient failed err : %v\n", err)
		return
	}
	result, err := db.Query(`
		SELECT id, amount, claim_user_id FROM withdraw_claim WHERE claim_status = ?`,
		TX_CHAIN_ERROR)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return
	} else if err != nil {
		log.Printf("[RetryFailedTransactions][Get claims] err : %v\n", err)
		return
	}
	for result.Next() {
		tx, err := db.Begin()
		defer tx.Commit()
		var withdrawal withdrawalClaim
		err = result.Scan(&withdrawal.Id, &withdrawal.Amount, &withdrawal.ClaimUserId)
		if err != nil {
			log.Printf("[RetryFailedTransactions][Get claim] err : %v\n", err)
			return
		}
		var claimUser User
		userRow := db.QueryRow("SELECT account_number FROM users where id = ?", withdrawal.ClaimUserId)
		err = userRow.Scan(&claimUser.AccountNumber)
		if err != nil {
			log.Printf("[RetryFailedTransactions][Get User] err : %v\n", err)
			return
		}
		amountBig := big.NewInt(0)
		_, isSuccess := amountBig.SetString(withdrawal.Amount, 10)
		if !isSuccess {
			log.Printf("[RetryFailedTransactions][Parse Amount to big.int] err : %v\n", err)
			return
		}
		claimStatus, txHash := SubmitWithdrawalToChain(claimUser.AccountNumber, amountBig)
		updateResult, err := updateWithdrawalStatusAndTxHash(tx, withdrawal.Id, claimStatus, txHash)
		if err != nil {
			log.Printf(`
				[RetryFailedTransactions] Update withdrawal after submitting to chain failed 
				claim id : %d , transaction hash : %v\n`, withdrawal.Id, txHash)
			return
		}
		rowsAffected, _ := updateResult.RowsAffected()
		if rowsAffected == 0 {
			log.Printf(`
				[RetryFailedTransactions] Update withdrawal after submitting to chain failed 
				claim id : %d , transaction hash : %v\n`, withdrawal.Id, txHash)
			return
		}
	}
}

func HasEnoughTreasuryBalance(withdrawalAmount *big.Int) (isEnough bool, err error) {
	ctx := context.Background()
	client, err := GetEthClient()
	if err != nil {
		log.Println("[CheckTreasuryBalance] connection to node failed", err)
		return
	}
	auth, err := GetTreasuryAuth(ctx, client)
	if err != nil {
		log.Println("[setUpDefaultContract][getTreasuryAuth] err ")
		return
	}
	treasuryAddress := auth.From.String()
	diff := big.NewInt(0)
	balance, err := GetAccountBalance(treasuryAddress)
	diff.Sub(balance, withdrawalAmount)
	isEnough = diff.Int64() > 0
	return
}

func SubmitWithdrawalToChain(accountNumber string, withdrawalAmount *big.Int) (updateStatus int64, txHash string) {
	ctx := context.Background()
	client, err := GetEthClient()
	if err != nil {
		log.Println("[SubmitWithdrawalToChain] connection to node failed", err)
		return
	}
	auth, err := GetTreasuryAuth(ctx, client)
	if err != nil {
		log.Println("[setUpDefaultContract][getTreasuryAuth] err ")
		return
	}
	contract := getContract()
	toAddress := common.HexToAddress(accountNumber)
	log.Println(toAddress, withdrawalAmount)
	tx, err := contract.Withdraw(auth, toAddress, withdrawalAmount)
	if err != nil {
		log.Fatalf("Failed to wait for withdraw transction to be submitted : %v", err)
	}
	fmt.Printf("Withdrawal Transaction waiting to be mined: 0x%x\n", tx.Hash())
	receipt, err := bind.WaitMined(ctx, client, tx)
	fmt.Println("Transaction minted", receipt.Status)
	if receipt.Status == types.ReceiptStatusSuccessful {
		updateStatus = TX_SUCCESS_ON_CHAIN
		txHash = tx.Hash().String()
	} else {
		updateStatus = TX_CHAIN_ERROR
	}
	return
}

func IsUserAuthorized(db *sql.DB, w *http.ResponseWriter, token string) (user User, isAuthorized bool, err error) {
	row := db.QueryRow("SELECT id , role, account_number FROM users where token = ?", token)
	err = row.Scan(&user.Id, &user.Role, &user.AccountNumber)
	if err != nil {
		log.Printf("[IsUserAuthorized] err : %v\n", err)
		CheckDBErr(w, err, "Cannot find specific User")
		return
	}
	if user.Role != ROLE_MANAGER {
		isAuthorized = false
		(*w).WriteHeader(http.StatusUnauthorized)
		(*w).Write([]byte("User has no permission to approve withdrawal claims"))
		return
	}
	return
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
	user, isAuthorized, err := IsUserAuthorized(db, &w, token)
	if !isAuthorized || err != nil {
		return
	}

	err = r.ParseMultipartForm(10 << 20)
	claimIdStr := r.FormValue("ClaimId")
	claimId, err := strconv.ParseInt(claimIdStr, 10, 64)
	var withdrawal withdrawalClaim
	// Retrieve Withdraw Claim
	{
		withdrawRow := db.QueryRow("SELECT id, amount, claim_user_id, claim_status FROM withdraw_claim where id = ?", claimId)
		err = withdrawRow.Scan(&withdrawal.Id, &withdrawal.Amount, &withdrawal.ClaimUserId, &withdrawal.ClaimStatus)
		if err != nil {
			log.Printf("[HandleApproval][Get claim] err : %v\n", err)
			CheckDBErr(&w, err, fmt.Sprintf("Withdrawal Claim of ID : %d cannot be found", claimId))
			return
		}
	}
	if withdrawal.ClaimStatus == TX_SUCCESS_ON_CHAIN {
		w.WriteHeader(200)
		w.Write([]byte("Withdrawal Claim has already been approved by 2 managers and submitted to chain"))
		return
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
		result, err := tx.Exec(`
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
		updateWithdrawalRes, err := updateWithdrawalStatus(tx, claimId, TX_APPROVAL_IN_PROCESS)
		if err != nil {
			CheckDBExecErr(updateWithdrawalRes, &w, err, "Update withdrawal claim status failed")
			tx.Rollback()
			return
		}
	}

	approvalRes.TotalApproval += 1
	if approvalRes.TotalApproval != 2 {
		tx.Commit()
		w.WriteHeader(200)
		w.Write([]byte("Approve successful"))
		return
	}
	amountBig := big.NewInt(0)
	_, isSuccess := amountBig.SetString(withdrawal.Amount, 10)
	if !isSuccess {
		w.WriteHeader(200)
		w.Write([]byte("Claim amount parsing error"))
		return
	}
	var claimUser User
	// Retrieve User that submited claim
	{
		row := db.QueryRow("SELECT account_number FROM users where id = ?", withdrawal.ClaimUserId)
		err = row.Scan(&claimUser.AccountNumber)
		if err != nil {
			log.Printf("[HandleNewWithdraw][Get User] err : %v\n", err)
			CheckDBErr(&w, err, "Cannot find specific User")
			tx.Rollback()
			return
		}
	}

	// Update withdrawal status to be approved
	updateWithdrawalRes, err := updateWithdrawalStatus(tx, claimId, TX_APPROVED)
	if err != nil {
		CheckDBExecErr(updateWithdrawalRes, &w, err, "Update withdrawal claim status failed")
		tx.Rollback()
		return
	}
	claimStatus, txHash := SubmitWithdrawalToChain(claimUser.AccountNumber, amountBig)
	updateWithdrawalRes, err = updateWithdrawalStatusAndTxHash(tx, claimId, claimStatus, txHash)
	if err != nil {
		log.Printf(`
				"HandleWithdrawApproval] Update withdrawal after submitting to chain failed 
				claim id : %d , transaction hash : %v\n`, claimId, txHash)
		CheckDBExecErr(updateWithdrawalRes, &w, err, "Update withdrawal claim after submitting to chain failed")
		return
	}
	tx.Commit()
	w.WriteHeader(200)
	w.Write([]byte("Approve successful"))
	return
}

func HandleCheckBalance(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Token")
	db, err := GetDBClient()
	if err != nil {
		w.WriteHeader(200)
		w.Write([]byte("Could not connect to db"))
		return
	}
	defer db.Close()
	var user User
	row := db.QueryRow("SELECT id , account_number FROM users where token = ?", token)
	err = row.Scan(&user.Id, &user.AccountNumber)
	if err != nil {
		log.Printf("[HandleCheckBalance][Get User] err : %v\n", err)
		CheckDBErr(&w, err, "Cannot find specific User")
		return
	}
	log.Printf("[HandleCheckBalance] Checking Balance for %v\n", user.AccountNumber)
	balance, err := GetAccountBalance(user.AccountNumber)
	fmt.Printf("Balance is %v\n", balance)
}
