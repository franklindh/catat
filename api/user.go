package api

import (
	"database/sql"
	"errors"
	"net/http"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type userResponse struct {
	ID        int64              `json:"id"`
	GoogleID  string             `json:"google_id"`
	Email     string             `json:"email"`
	Name      string             `json:"name"`
	Balance   string             `json:"balance"`
	AvatarUrl string             `json:"avatar_url"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		ID:        user.ID,
		GoogleID:  user.GoogleAuthID.String,
		Email:     user.Email,
		Name:      user.Name.String,
		AvatarUrl: user.AvatarUrl.String,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (s *Server) getUser(ctx *gin.Context) {
	payload, ok := ctx.Get(authorizationPayloadKey)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization payload not found")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(errors.New("invalid payload type")))
		return
	}

	userID := authPayload.UserID

	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := newUserResponse(user)
	ctx.JSON(http.StatusOK, rsp)
}

type updateUserRequest struct {
	Name      string `json:"name"`
	AvatarUrl string `json:"avatar_url" binding:"omitempty,url"`
}

func (s *Server) updateUser(ctx *gin.Context) {
	var req updateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	payload, ok := ctx.Get(authorizationPayloadKey)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization payload not found")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(errors.New("invalid payload type")))
		return
	}

	userID := authPayload.UserID

	arg := db.UpdateUserParams{
		ID:        userID,
		Name:      pgtype.Text{String: req.Name, Valid: req.Name != ""},
		AvatarUrl: pgtype.Text{String: req.AvatarUrl, Valid: req.AvatarUrl != ""},
	}

	_, err := s.store.UpdateUser(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("user not found to update")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user updated successfully"})
}

func (s *Server) deleteUser(ctx *gin.Context) {
	payload, ok := ctx.Get(authorizationPayloadKey)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization payload not found")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(errors.New("invalid payload type")))
		return
	}

	userID := authPayload.UserID

	err := s.store.DeleteUser(ctx, userID)
	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}
