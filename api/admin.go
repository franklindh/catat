package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type adminUserResponse struct {
	ID        int64              `json:"id"`
	GoogleID  string             `json:"google_id"`
	Email     string             `json:"email"`
	Name      string             `json:"name"`
	AvatarUrl string             `json:"avatar_url"`
	Role      string             `json:"role"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

func newAdminUserResponse(user db.ListUsersRow) adminUserResponse {
	return adminUserResponse{
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

func newAdminUserResponseFromGetUser(user db.GetUserRow) adminUserResponse {
	return adminUserResponse{
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

func newAdminUserResponseFromUpdateRole(user db.UpdateUserRoleRow) adminUserResponse {
	return adminUserResponse{
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

type listUsersRequest struct {
	Page   int32  `form:"page" binding:"min=0"`
	Limit  int32  `form:"limit" binding:"min=0,max=100"`
	Search string `form:"search"`
}

type listUsersResponse struct {
	Users      []adminUserResponse `json:"users"`
	TotalCount int64               `json:"total_count"`
	Page       int32               `json:"page"`
	Limit      int32               `json:"limit"`
	TotalPages int32               `json:"total_pages"`
}

// listUsers godoc
// @Summary      List all users (Admin only)
// @Description  Mendapatkan daftar semua user dengan pagination dan pencarian
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page number"  default(1)
// @Param        limit   query     int     false  "Number of items per page (max: 100)"  default(20)
// @Param        search  query     string  false  "Search by name or email"
// @Success      200     {object}  listUsersResponse
// @Failure      400     {object}  map[string]string  "Bad request"
// @Failure      401     {object}  map[string]string  "Unauthorized"
// @Failure      403     {object}  map[string]string  "Forbidden - Admin only"
// @Failure      500     {object}  map[string]string  "Internal server error"
// @Router       /admin/users [get]
func (s *Server) listUsers(ctx *gin.Context) {
	var req listUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	offset := (req.Page - 1) * req.Limit

	var users []db.ListUsersRow
	var err error

	if req.Search != "" {

		searchResults, searchErr := s.store.SearchUsers(ctx, db.SearchUsersParams{
			Column1: pgtype.Text{String: req.Search, Valid: true},
			Limit:   req.Limit,
			Offset:  offset,
		})
		if searchErr != nil {
			ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(searchErr))
			return
		}

		for _, sr := range searchResults {
			users = append(users, db.ListUsersRow{
				ID:           sr.ID,
				Email:        sr.Email,
				Name:         sr.Name,
				AvatarUrl:    sr.AvatarUrl,
				GoogleAuthID: sr.GoogleAuthID,
				Role:         sr.Role,
				CreatedAt:    sr.CreatedAt,
				UpdatedAt:    sr.UpdatedAt,
			})
		}
	} else {
		users, err = s.store.ListUsers(ctx, db.ListUsersParams{
			Limit:  req.Limit,
			Offset: offset,
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
			return
		}
	}

	totalCount, err := s.store.CountUsers(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	userResponses := make([]adminUserResponse, len(users))
	for i, user := range users {
		userResponses[i] = newAdminUserResponse(user)
	}

	totalPages := int32(totalCount) / req.Limit
	if int32(totalCount)%req.Limit > 0 {
		totalPages++
	}

	rsp := listUsersResponse{
		Users:      userResponses,
		TotalCount: totalCount,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}

	ctx.JSON(http.StatusOK, rsp)
}

// getUserByID godoc
// @Summary      Get user by ID (Admin only)
// @Description  Mendapatkan detail user berdasarkan ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  adminUserResponse
// @Failure      400  {object}  map[string]string  "Bad request"
// @Failure      401  {object}  map[string]string  "Unauthorized"
// @Failure      403  {object}  map[string]string  "Forbidden - Admin only"
// @Failure      404  {object}  map[string]string  "User not found"
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /admin/users/{id} [get]
func (s *Server) getUserByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid user id")))
		return
	}

	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := newAdminUserResponseFromGetUser(user)
	ctx.JSON(http.StatusOK, rsp)
}

type updateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=ADMIN USER"`
}

// updateUserRole godoc
// @Summary      Update user role (Admin only)
// @Description  Mengubah role user (ADMIN atau USER)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                    true  "User ID"
// @Param        request  body      updateUserRoleRequest  true  "Update role request"
// @Success      200      {object}  adminUserResponse
// @Failure      400      {object}  map[string]string  "Bad request"
// @Failure      401      {object}  map[string]string  "Unauthorized"
// @Failure      403      {object}  map[string]string  "Forbidden - Admin only"
// @Failure      404      {object}  map[string]string  "User not found"
// @Failure      500      {object}  map[string]string  "Internal server error"
// @Router       /admin/users/{id}/role [put]
func (s *Server) updateUserRole(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid user id")))
		return
	}

	var req updateUserRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	user, err := s.store.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:   id,
		Role: req.Role,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := newAdminUserResponseFromUpdateRole(user)
	ctx.JSON(http.StatusOK, rsp)
}

// deleteUserByAdmin godoc
// @Summary      Delete user (Admin only)
// @Description  Menghapus user berdasarkan ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  map[string]string  "User deleted successfully"
// @Failure      400  {object}  map[string]string  "Bad request"
// @Failure      401  {object}  map[string]string  "Unauthorized"
// @Failure      403  {object}  map[string]string  "Forbidden - Admin only"
// @Failure      404  {object}  map[string]string  "User not found"
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /admin/users/{id} [delete]
func (s *Server) deleteUserByAdmin(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid user id")))
		return
	}

	_, err = s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	err = s.store.DeleteUser(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

type adminStatsResponse struct {
	TotalUsers        int64 `json:"total_users"`
	TotalTransactions int64 `json:"total_transactions"`
	TotalCategories   int64 `json:"total_categories"`
}

// getAdminStats godoc
// @Summary      Get admin statistics (Admin only)
// @Description  Mendapatkan statistik admin (total user, transaksi, kategori)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  adminStatsResponse
// @Failure      401  {object}  map[string]string  "Unauthorized"
// @Failure      403  {object}  map[string]string  "Forbidden - Admin only"
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /admin/stats [get]
func (s *Server) getAdminStats(ctx *gin.Context) {
	totalUsers, err := s.store.CountUsers(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := adminStatsResponse{
		TotalUsers: totalUsers,
	}

	ctx.JSON(http.StatusOK, rsp)
}
