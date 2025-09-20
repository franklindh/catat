package api

import (
	"net/http"
	"strings"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type getUserRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

type getUserByEmailRequest struct {
	Email string `form:"email" binding:"required,email"`
}

type listUsersRequest struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

type updateUserRequest struct {
	ID    string `json:"id" binding:"required,uuid"`
	Email string `json:"email,omitempty" binding:"omitempty,email"`
	Name  string `json:"name,omitempty" binding:"omitempty"`
}

type deleteUserRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func (s *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	hashedPassword, _ := util.HashPassword(req.Password)

	arg := db.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: hashedPassword,
	}

	user, err := s.Store.CreateUser(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

func (s *Server) getUserByID(ctx *gin.Context) {
	var req getUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	user, err := s.Store.GetUserByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			ctx.JSON(http.StatusNotFound, util.ErrorResponseWithMessage("user not found"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (s *Server) getUserByEmail(ctx *gin.Context) {
	var req getUserByEmailRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	user, err := s.Store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (s *Server) listUsers(ctx *gin.Context) {
	var req listUsersRequest
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

	offset := (req.Page - 1) * req.Limit

	users, err := s.Store.ListUsers(ctx, db.ListUsersParams{
		Limit:  int32(req.Limit),
		Offset: int32(offset),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	total := int64(len(users))

	ctx.JSON(http.StatusOK, gin.H{
		"data": users,
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

func (s *Server) updateUser(ctx *gin.Context) {
	var req updateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.UpdateUserParams{
		ID: userID,
		Email: pgtype.Text{
			String: req.Email,
			Valid:  req.Email != "",
		},
		Name: pgtype.Text{
			String: req.Name,
			Valid:  req.Name != "",
		},
	}

	user, err := s.Store.UpdateUser(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (s *Server) deleteUser(ctx *gin.Context) {
	var req deleteUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	err = s.Store.DeleteUser(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}
