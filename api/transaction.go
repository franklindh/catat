package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type transactionResponse struct {
	ID              uuid.UUID          `json:"id"`
	UserID          string             `json:"user_id"`
	CategoryID      string             `json:"category_id"`
	Amount          pgtype.Numeric     `json:"amount"`
	Description     string             `json:"description"`
	TransactionDate pgtype.Timestamptz `json:"transaction_date"`
	CreatedAt       pgtype.Timestamptz `json:"created_at"`
}

type createTransactionRequest struct {
	CategoryID      uuid.UUID          `json:"category_id" binding:"required,uuid"`
	Amount          pgtype.Numeric     `json:"amount" binding:"required"`
	Description     string             `json:"description"`
	TransactionDate pgtype.Timestamptz `json:"transaction_date"`
}

type updateTransactionRequest struct {
	CategoryID      string             `json:"category_id" binding:"omitempty,uuid"`
	Amount          pgtype.Numeric     `json:"amount" binding:"omitempty"`
	Description     string             `json:"description" binding:"omitempty"`
	TransactionDate pgtype.Timestamptz `json:"transaction_date" binding:"omitempty"`
}

type listTransactionsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=100"`
}

type listTransactionsByUserAndCategoryRequest struct {
	CategoryID string `uri:"category_id" binding:"required,uuid"`
	PageID     int32  `form:"page_id" binding:"required,min=1"`
	PageSize   int32  `form:"page_size" binding:"required,min=5,max=100"`
}

type listTransactionsByUserInDateRangeRequest struct {
	StartDate string `form:"start_date" binding:"required"`
	EndDate   string `form:"end_date" binding:"required"`
	PageID    int32  `form:"page_id" binding:"required,min=1"`
	PageSize  int32  `form:"page_size" binding:"required,min=5,max=100"`
}

func transactionToTransactionResponse(transaction db.Transaction) transactionResponse {
	return transactionResponse{
		ID:              util.PgxUUIDToGoogleUUID(transaction.ID),
		UserID:          util.PgxUUIDToGoogleUUID(transaction.UserID).String(),
		CategoryID:      util.PgxUUIDToGoogleUUID(transaction.CategoryID).String(),
		Amount:          transaction.Amount,
		Description:     transaction.Description.String,
		TransactionDate: transaction.TransactionDate,
		CreatedAt:       transaction.CreatedAt,
	}
}

func transactionsToTransactionResponses(transactions []db.Transaction) []transactionResponse {
	var responses []transactionResponse
	for _, transaction := range transactions {
		responses = append(responses, transactionToTransactionResponse(transaction))
	}
	return responses
}

func (s *Server) createTransaction(ctx *gin.Context) {
	fmt.Printf("DEBUG: createTransaction called\n")
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}
	fmt.Printf("DEBUG: authPayload.UserID: %s\n", authPayload.UserID.String())

	var req createTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fmt.Printf("DEBUG: Binding error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}
	fmt.Printf("DEBUG: Request bound: %+v\n", req)

	categoryID := req.CategoryID

	arg := db.CreateTransactionParams{
		UserID:          util.GoogleUUIDToPgxUUID(authPayload.UserID),
		CategoryID:      util.GoogleUUIDToPgxUUID(categoryID),
		Amount:          req.Amount,
		Description:     pgtype.Text{String: req.Description, Valid: req.Description != ""},
		TransactionDate: req.TransactionDate,
	}

	transaction, err := s.store.CreateTransaction(ctx, arg)
	if err != nil {
		fmt.Printf("DEBUG: Store CreateTransaction error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}
	fmt.Printf("DEBUG: Transaction created: %+v\n", transaction)

	rsp := transactionToTransactionResponse(transaction)
	ctx.JSON(http.StatusCreated, rsp)
}

func (s *Server) getTransactions(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	transactionIDStr := ctx.Param("id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid transaction ID format")))
		return
	}

	transaction, err := s.store.GetTransaction(ctx, util.GoogleUUIDToPgxUUID(transactionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("transaction not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if transaction.UserID != util.GoogleUUIDToPgxUUID(authPayload.UserID) {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("forbidden: cannot access other user's transaction")))
		return
	}

	rsp := transactionToTransactionResponse(transaction)
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) getTransaction(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	var req listTransactionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	arg := db.GetTransactionsParams{
		UserID: util.GoogleUUIDToPgxUUID(authPayload.UserID),
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	transactions, err := s.store.GetTransactions(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := transactionsToTransactionResponses(transactions)
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) listTransaction(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	var req listTransactionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	offset := (req.PageID - 1) * req.PageSize

	arg := db.GetTransactionsParams{
		UserID: util.GoogleUUIDToPgxUUID(authPayload.UserID), // Gunakan UserID dari token
		Limit:  req.PageSize,
		Offset: offset,
	}

	transactions, err := s.store.GetTransactions(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := transactionsToTransactionResponses(transactions)
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) listTrnsactionsByDateRange(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	var req listTransactionsByUserInDateRangeRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid start date format, expected YYYY-MM-DD")))
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid end date format, expected YYYY-MM-DD")))
		return
	}

	// 4. Hitung offset untuk pagination
	offset := (req.PageID - 1) * req.PageSize

	// 5. Siapkan argumen untuk SQLC
	arg := db.GetTransactionsByDateRangeParams{
		UserID:            util.GoogleUUIDToPgxUUID(authPayload.UserID), // Gunakan UserID dari token
		TransactionDate:   pgtype.Timestamptz{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Timestamptz{Time: endDate.Add(24*time.Hour - time.Nanosecond), Valid: true}, // Sertakan akhir hari
		Limit:             req.PageSize,
		Offset:            offset,
	}

	transactions, err := s.store.GetTransactionsByDateRange(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := transactionsToTransactionResponses(transactions)
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) updateTransaction(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	transactionIDStr := ctx.Param("id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid transaction ID format")))
		return
	}

	var req updateTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	existingTransaction, err := s.store.GetTransaction(ctx, util.GoogleUUIDToPgxUUID(transactionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("transaction not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if existingTransaction.UserID != util.GoogleUUIDToPgxUUID(authPayload.UserID) {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("forbidden: cannot access other user's transaction")))
		return
	}

	arg := db.UpdateTransactionParams{
		ID:              util.GoogleUUIDToPgxUUID(transactionID),
		UserID:          util.GoogleUUIDToPgxUUID(authPayload.UserID),
		CategoryID:      existingTransaction.CategoryID,
		Amount:          existingTransaction.Amount,
		Description:     existingTransaction.Description,
		TransactionDate: existingTransaction.TransactionDate,
	}

	if req.CategoryID != "" {
		categoryID, parseErr := uuid.Parse(req.CategoryID)
		if parseErr != nil {
			ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid category ID format")))
			return
		}
		arg.CategoryID = util.GoogleUUIDToPgxUUID(categoryID)
	}
	if req.Amount != (pgtype.Numeric{}) {
		arg.Amount = req.Amount
	}
	if req.Description != "" {
		arg.Description = pgtype.Text{String: req.Description, Valid: true}
	}
	if req.TransactionDate.Valid {
		arg.TransactionDate = req.TransactionDate
	}

	updatedTransaction, err := s.store.UpdateTransaction(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := transactionToTransactionResponse(updatedTransaction)
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) deleteTransaction(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	transactionIDStr := ctx.Param("id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid transaction ID format")))
		return
	}

	existingTransaction, err := s.store.GetTransaction(ctx, util.GoogleUUIDToPgxUUID(transactionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("transaction not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if existingTransaction.UserID != util.GoogleUUIDToPgxUUID(authPayload.UserID) {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("forbidden: cannot access other user's transaction")))
		return
	}

	arg := db.DeleteTransactionParams{
		ID:     util.GoogleUUIDToPgxUUID(transactionID),
		UserID: util.GoogleUUIDToPgxUUID(authPayload.UserID),
	}

	err = s.store.DeleteTransaction(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "transaction deleted successfully"})
}
