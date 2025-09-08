package api

import (
	"math/big"
	"net/http"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type createAccountRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required,oneof=depository credit cash"`
}

type getAccountRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

type listAccountsRequest struct {
	UserID string `form:"user_id" binding:"required,uuid"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

type updateAccountRequest struct {
	ID   string `json:"id" binding:"required,uuid"`
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required,oneof=depository credit cash"`
}

type deleteAccountRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func createZeroBalance() pgtype.Numeric {
	return pgtype.Numeric{
		Int:   big.NewInt(0),
		Exp:   0,
		Valid: true,
	}
}

func (s *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	userID, err := parseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.CreateAccountParams{
		UserID:  userID,
		Name:    req.Name,
		Type:    req.Type,
		Balance: createZeroBalance(),
	}

	account, err := s.queries.CreateAccount(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, account)
}

func (s *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	accountID, err := parseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponseWithMessage("invalid account ID format"))
		return
	}

	userID := pgtype.UUID{
		Bytes: uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb"),
		Valid: true,
	}

	arg := db.GetAccountParams{
		ID:     accountID,
		UserID: userID,
	}

	account, err := s.queries.GetAccount(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

func (s *Server) listAccounts(ctx *gin.Context) {
	var req listAccountsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
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

	userID, err := parseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponseWithMessage("invalid user ID format"))
		return
	}

	offset := (req.Page - 1) * req.Limit

	accounts, err := s.queries.ListAccounts(ctx, db.ListAccountsParams{
		UserID: userID,
		Limit:  int32(req.Limit),
		Offset: int32(offset),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	total, err := s.queries.CountAccountsByUser(ctx, userID)
	if err != nil {

		ctx.JSON(http.StatusOK, gin.H{
			"data": accounts,
			"pagination": gin.H{
				"page":  req.Page,
				"limit": req.Limit,
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": accounts,
		"pagination": gin.H{
			"page":        req.Page,
			"limit":       req.Limit,
			"total":       total,
			"totalPages":  (int(total) + req.Limit - 1) / req.Limit,
			"hasNext":     req.Page*req.Limit < int(total),
			"hasPrevious": req.Page > 1,
		},
	})
}
func (s *Server) updateAccount(ctx *gin.Context) {
	var req updateAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	accountID, err := parseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponseWithMessage("invalid account ID format"))
		return
	}

	userID := pgtype.UUID{
		Bytes: uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb"),
		Valid: true,
	}

	arg := db.UpdateAccountParams{
		ID:     accountID,
		Name:   req.Name,
		Type:   req.Type,
		UserID: userID,
	}

	account, err := s.queries.UpdateAccount(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

func (s *Server) deleteAccount(ctx *gin.Context) {
	var req deleteAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	accountID, err := parseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponseWithMessage("invalid account ID format"))
		return
	}

	userID := pgtype.UUID{
		Bytes: uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb"),
		Valid: true,
	}

	arg := db.DeleteAccountParams{
		ID:     accountID,
		UserID: userID,
	}

	err = s.queries.DeleteAccount(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "account deleted successfully"})
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func errorResponseWithMessage(message string) gin.H {
	return gin.H{"error": message}
}
