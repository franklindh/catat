package api

import (
	"math/big"
	"net/http"
	"time"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type createTransactionRequest struct {
	UserID          string    `json:"user_id" binding:"required,uuid"`
	AccountID       string    `json:"account_id" binding:"required,uuid"`
	CategoryID      string    `json:"category_id" binding:"required,uuid"`
	Amount          float64   `json:"amount" binding:"required"`
	Description     string    `json:"description" binding:"required"`
	TransactionDate time.Time `json:"transaction_date" binding:"required"`
}

type listTransactionsRequest struct {
	UserID string `form:"user_id" binding:"required,uuid"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

type listTransactionsByAccountRequest struct {
	UserID    string `form:"user_id" binding:"required,uuid"`
	AccountID string `form:"account_id" binding:"required,uuid"`
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
}

type listTransactionsByDateRangeRequest struct {
	UserID    string    `form:"user_id" binding:"required,uuid"`
	StartDate time.Time `form:"start_date" binding:"required"`
	EndDate   time.Time `form:"end_date" binding:"required"`
}

type updateTransactionRequest struct {
	ID              string    `json:"id" binding:"required,uuid"`
	UserID          string    `json:"user_id" binding:"required,uuid"`
	AccountID       string    `json:"account_id" binding:"required,uuid"`
	CategoryID      string    `json:"category_id" binding:"required,uuid"`
	Amount          float64   `json:"amount" binding:"required"`
	Description     string    `json:"description" binding:"required"`
	TransactionDate time.Time `json:"transaction_date" binding:"required"`
}

func (s *Server) createTransaction(ctx *gin.Context) {
	var req createTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	accountID, err := util.ParseUUID(req.AccountID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid account ID format"))
		return
	}

	categoryID, err := util.ParseUUID(req.CategoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid category ID format"))
		return
	}

	amount := pgtype.Numeric{
		Int:   big.NewInt(int64(req.Amount * 10000)),
		Exp:   -4,
		Valid: true,
	}

	transactionDate := pgtype.Timestamptz{
		Time:  req.TransactionDate,
		Valid: true,
	}

	arg := db.CreateTransactionParams{
		UserID:          userID,
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		Description:     req.Description,
		TransactionDate: transactionDate,
	}

	transaction, err := s.store.CreateTransaction(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, transaction)
}

func (s *Server) getTransaction(ctx *gin.Context) {
	var uriReq struct {
		ID string `uri:"id" binding:"required,uuid"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	var queryReq struct {
		UserID string `form:"user_id" binding:"required,uuid"`
	}
	if err := ctx.ShouldBindQuery(&queryReq); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	transactionID, err := util.ParseUUID(uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid transaction ID format"))
		return
	}

	userID, err := util.ParseUUID(queryReq.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.GetTransactionParams{
		ID:     transactionID,
		UserID: userID,
	}

	transaction, err := s.store.GetTransaction(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transaction)
}

func (s *Server) listTransactions(ctx *gin.Context) {
	var req listTransactionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	offset := (req.Page - 1) * req.Limit

	transactions, err := s.store.ListTransactions(ctx, db.ListTransactionsParams{
		UserID: userID,
		Limit:  int32(req.Limit),
		Offset: int32(offset),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transactions)
}

func (s *Server) listTransactionsByAccount(ctx *gin.Context) {
	var req listTransactionsByAccountRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	accountID, err := util.ParseUUID(req.AccountID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid account ID format"))
		return
	}

	offset := (req.Page - 1) * req.Limit

	transactions, err := s.store.ListTransactionsByAccount(ctx, db.ListTransactionsByAccountParams{
		UserID:    userID,
		AccountID: accountID,
		Limit:     int32(req.Limit),
		Offset:    int32(offset),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transactions)
}

func (s *Server) listTransactionsByDateRange(ctx *gin.Context) {
	var req listTransactionsByDateRangeRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	startDate := pgtype.Timestamptz{
		Time:  req.StartDate,
		Valid: true,
	}

	endDate := pgtype.Timestamptz{
		Time:  req.EndDate,
		Valid: true,
	}

	transactions, err := s.store.ListTransactionsByDateRange(ctx, db.ListTransactionsByDateRangeParams{
		UserID:            userID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transactions)
}

func (s *Server) updateTransaction(ctx *gin.Context) {
	var req updateTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	transactionID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid transaction ID format"))
		return
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	accountID, err := util.ParseUUID(req.AccountID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid account ID format"))
		return
	}

	categoryID, err := util.ParseUUID(req.CategoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid category ID format"))
		return
	}

	amount := pgtype.Numeric{
		Int:   big.NewInt(int64(req.Amount * 10000)),
		Exp:   -4,
		Valid: true,
	}

	transactionDate := pgtype.Timestamptz{
		Time:  req.TransactionDate,
		Valid: true,
	}

	arg := db.UpdateTransactionParams{
		ID:              transactionID,
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		Description:     req.Description,
		TransactionDate: transactionDate,
		UserID:          userID,
	}

	transaction, err := s.store.UpdateTransaction(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transaction)
}

func (s *Server) deleteTransaction(ctx *gin.Context) {
	var uriReq struct {
		ID string `uri:"id" binding:"required,uuid"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	var queryReq struct {
		UserID string `form:"user_id" binding:"required,uuid"`
	}
	if err := ctx.ShouldBindQuery(&queryReq); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	transactionID, err := util.ParseUUID(uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid transaction ID format"))
		return
	}

	userID, err := util.ParseUUID(queryReq.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.DeleteTransactionParams{
		ID:     transactionID,
		UserID: userID,
	}

	err = s.store.DeleteTransaction(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "transaction deleted successfully"})
}
