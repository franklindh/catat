package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type transactionResponse struct {
	ID              int64              `json:"id"`
	UserID          int64              `json:"user_id"`
	CategoryID      *int64             `json:"category_id"`
	Amount          pgtype.Numeric     `json:"amount"`
	Description     string             `json:"description"`
	TransactionDate pgtype.Timestamptz `json:"transaction_date"`
	Type            string             `json:"type"`
	CreatedAt       pgtype.Timestamptz `json:"created_at"`
}

type createTransactionRequest struct {
	CategoryID      *int64 `json:"category_id"`
	Amount          string `json:"amount" binding:"required"`
	Description     string `json:"description"`
	TransactionDate string `json:"transaction_date" binding:"required"`
	Type            string `json:"type" binding:"required"`
}

type updateTransactionRequest struct {
	CategoryID      *int64 `json:"category_id"`
	Amount          string `json:"amount" binding:"omitempty"`
	Description     string `json:"description" binding:"omitempty"`
	TransactionDate string `json:"transaction_date" binding:"omitempty"`
	Type            string `json:"type" binding:"omitempty"`
}

type listTransactionsRequest struct {
	PageID   int32 `form:"page_id" binding:"min=1"`
	PageSize int32 `form:"page_size" binding:"min=5,max=100"`
}

func transactionToTransactionResponse(row interface{}) transactionResponse {
	switch t := row.(type) {
	case db.CreateTransactionRow:
		var categoryID *int64
		if t.CategoryID.Valid {
			categoryID = &t.CategoryID.Int64
		}
		return transactionResponse{
			ID:              t.ID,
			UserID:          t.UserID,
			CategoryID:      categoryID,
			Amount:          t.Amount,
			Description:     t.Description.String,
			TransactionDate: t.TransactionDate,
			Type:            t.Type,
			CreatedAt:       t.CreatedAt,
		}
	case db.GetTransactionRow:
		var categoryID *int64
		if t.CategoryID.Valid {
			categoryID = &t.CategoryID.Int64
		}
		return transactionResponse{
			ID:              t.ID,
			UserID:          t.UserID,
			CategoryID:      categoryID,
			Amount:          t.Amount,
			Description:     t.Description.String,
			TransactionDate: t.TransactionDate,
			Type:            t.Type,
			CreatedAt:       pgtype.Timestamptz{},
		}
	case db.UpdateTransactionRow:
		var categoryID *int64
		if t.CategoryID.Valid {
			categoryID = &t.CategoryID.Int64
		}
		return transactionResponse{
			ID:              t.ID,
			UserID:          t.UserID,
			CategoryID:      categoryID,
			Amount:          t.Amount,
			Description:     t.Description.String,
			TransactionDate: t.TransactionDate,
			Type:            t.Type,
			CreatedAt:       pgtype.Timestamptz{},
		}
	case db.ListTransactionsRow:
		var categoryID *int64
		if t.CategoryID.Valid {
			categoryID = &t.CategoryID.Int64
		}
		return transactionResponse{
			ID:              t.ID,
			UserID:          t.UserID,
			CategoryID:      categoryID,
			Amount:          t.Amount,
			Description:     t.Description.String,
			TransactionDate: t.TransactionDate,
			Type:            t.Type,
			CreatedAt:       pgtype.Timestamptz{},
		}
	default:
		return transactionResponse{}
	}
}

func transactionsToTransactionResponses(transactions []db.ListTransactionsRow) []transactionResponse {
	var responses []transactionResponse
	for _, transaction := range transactions {
		responses = append(responses, transactionToTransactionResponse(transaction))
	}
	return responses
}

func (s *Server) createTransaction(ctx *gin.Context) {
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

	var req createTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	var categoryID pgtype.Int8
	if req.CategoryID != nil {
		categoryID = pgtype.Int8{Int64: *req.CategoryID, Valid: true}
	}

	var amount pgtype.Numeric
	if err := amount.Scan(req.Amount); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid amount format")))
		return
	}

	parsedTime, err := time.Parse(time.RFC3339, req.TransactionDate)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format, use ISO 8601")))
		return
	}
	transactionDate := pgtype.Timestamptz{Time: parsedTime, Valid: true}

	_, err = s.store.GetUser(ctx, authPayload.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("user not found, please login again")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(errors.New("failed to validate user")))
		return
	}

	arg := db.CreateTransactionParams{
		UserID:          authPayload.UserID,
		CategoryID:      categoryID,
		Amount:          amount,
		Description:     pgtype.Text{String: req.Description, Valid: req.Description != ""},
		TransactionDate: transactionDate,
		Type:            req.Type,
	}

	transaction, err := s.store.CreateTransaction(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("user not found")))
				return
			case "23505":
				ctx.JSON(http.StatusConflict, util.ErrorResponse(errors.New("transaction already exists")))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

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
	transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid transaction ID format")))
		return
	}

	transaction, err := s.store.GetTransaction(ctx, db.GetTransactionParams{
		ID:     transactionID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("transaction not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
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

	arg := db.ListTransactionsParams{
		UserID: authPayload.UserID,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	transactions, err := s.store.ListTransactions(ctx, arg)
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
	transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid transaction ID format")))
		return
	}

	var req updateTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	existingTransaction, err := s.store.GetTransaction(ctx, db.GetTransactionParams{
		ID:     transactionID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("transaction not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	var categoryID pgtype.Int8
	if req.CategoryID != nil {
		categoryID = pgtype.Int8{Int64: *req.CategoryID, Valid: true}
	} else {
		categoryID = existingTransaction.CategoryID
	}

	var amount pgtype.Numeric
	if req.Amount != "" {
		if err := amount.Scan(req.Amount); err != nil {
			ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid amount format")))
			return
		}
	} else {
		amount = existingTransaction.Amount
	}

	var transactionDate pgtype.Timestamptz
	if req.TransactionDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, req.TransactionDate)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format, use ISO 8601")))
			return
		}
		transactionDate = pgtype.Timestamptz{Time: parsedTime, Valid: true}
	} else {
		transactionDate = existingTransaction.TransactionDate
	}

	description := existingTransaction.Description
	if req.Description != "" {
		description = pgtype.Text{String: req.Description, Valid: true}
	}

	transactionType := existingTransaction.Type
	if req.Type != "" {
		transactionType = req.Type
	}

	arg := db.UpdateTransactionParams{
		ID:              transactionID,
		UserID:          authPayload.UserID,
		CategoryID:      categoryID,
		Amount:          amount,
		Description:     description,
		TransactionDate: transactionDate,
		Type:            transactionType,
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
	transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid transaction ID format")))
		return
	}

	_, err = s.store.GetTransaction(ctx, db.GetTransactionParams{
		ID:     transactionID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("transaction not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	err = s.store.DeleteTransaction(ctx, db.DeleteTransactionParams{
		ID:     transactionID,
		UserID: authPayload.UserID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "transaction deleted successfully"})
}
