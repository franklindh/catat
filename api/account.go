package api

import (
	"errors"
	"math/big"
	"net/http"
	"strings"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.CreateAccountParams{
		UserID:  userID,
		Name:    req.Name,
		Type:    req.Type,
		Balance: createZeroBalance(),
	}

	account, err := s.Store.CreateAccount(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, account)
}

func (s *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	accountID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid account ID format"))
		return
	}

	userID := pgtype.UUID{
		Bytes: uuid.MustParse("b25d7919-6071-422a-85f9-c88afb3f63ad"),
		Valid: true,
	}

	arg := db.GetAccountParams{
		ID:     accountID,
		UserID: userID,
	}

	account, err := s.Store.GetAccount(ctx, arg)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			ctx.JSON(http.StatusNotFound, util.ErrorResponseWithMessage("account not found"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

func (s *Server) listAccounts(ctx *gin.Context) {
	var req listAccountsRequest
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

	accounts, err := s.Store.ListAccounts(ctx, db.ListAccountsParams{
		UserID: userID,
		Limit:  int32(req.Limit),
		Offset: int32(offset),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	total, err := s.Store.CountAccountsByUser(ctx, userID)
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
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	accountID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid account ID format"))
		return
	}

	userID := pgtype.UUID{
		Bytes: uuid.MustParse("b25d7919-6071-422a-85f9-c88afb3f63ad"),
		Valid: true,
	}

	arg := db.UpdateAccountParams{
		ID:     accountID,
		Name:   req.Name,
		Type:   req.Type,
		UserID: userID,
	}

	account, err := s.Store.UpdateAccount(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponseWithMessage("account not found"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

func (s *Server) deleteAccount(ctx *gin.Context) {
	var req deleteAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	accountID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid account ID format"))
		return
	}

	userID := pgtype.UUID{
		Bytes: uuid.MustParse("b25d7919-6071-422a-85f9-c88afb3f63ad"),
		Valid: true,
	}

	arg := db.DeleteAccountParams{
		ID:     accountID,
		UserID: userID,
	}

	err = s.Store.DeleteAccount(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponseWithMessage("account not found"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "account deleted successfully"})
}
