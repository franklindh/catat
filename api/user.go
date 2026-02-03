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
	AvatarUrl string             `json:"avatar_url"`
	Role      string             `json:"role"`
	CreatedAt pgtype.Timestamptz `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at" swaggertype:"string" format:"date-time"`
}

func newUserResponse(user db.GetUserRow) userResponse {
	return userResponse{
		ID:        user.ID,
		GoogleID:  user.GoogleAuthID.String,
		Email:     user.Email,
		Name:      user.Name.String,
		AvatarUrl: user.AvatarUrl.String,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// getUser godoc
// @Summary      Get current user profile
// @Description  Mendapatkan data profil user yang sedang login
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  userResponse
// @Failure      401  {object}  map[string]string  "Unauthorized"
// @Failure      404  {object}  map[string]string  "User not found"
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /user [get]
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

// updateUser godoc
// @Summary      Update user profile
// @Description  Mengupdate data profil user yang sedang login
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      updateUserRequest  true  "Update user request"
// @Success      200      {object}  map[string]string  "User updated successfully"
// @Failure      400      {object}  map[string]string  "Bad request"
// @Failure      401      {object}  map[string]string  "Unauthorized"
// @Failure      404      {object}  map[string]string  "User not found"
// @Failure      500      {object}  map[string]string  "Internal server error"
// @Router       /user [put]
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

// deleteUser godoc
// @Summary      Delete user account
// @Description  Menghapus akun user yang sedang login
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string  "User deleted successfully"
// @Failure      401  {object}  map[string]string  "Unauthorized"
// @Failure      404  {object}  map[string]string  "User not found"
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /user [delete]
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
