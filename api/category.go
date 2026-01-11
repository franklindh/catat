package api

import (
	"database/sql"
	"errors"
	"net/http"

	"strconv"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type categoryResponse struct {
	ID        int64              `json:"id"`
	UserID    int64              `json:"user_id"`
	Name      string             `json:"name"`
	Type      string             `json:"type"`
	IconURL   string             `json:"icon_url"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

type createCategoryRequest struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"omitempty,oneof=INCOME EXPENSE"`
	IconURL string `json:"icon_url"`
}

type updateCategoryRequest struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"omitempty,oneof=INCOME EXPENSE"`
	IconURL string `json:"icon_url"`
}

func categoryResponseFromParts(id, userID int64, name, catType string, icon pgtype.Text, created, updated pgtype.Timestamptz) categoryResponse {
	return categoryResponse{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Type:      catType,
		IconURL:   icon.String,
		CreatedAt: created,
		UpdatedAt: updated,
	}
}

func (s *Server) createCategory(ctx *gin.Context) {

	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization payload not found")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	var req createCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {

		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	catType := req.Type
	if catType == "" {
		catType = "EXPENSE"
	}

	arg := db.CreateCategoryParams{
		UserID:  authPayload.UserID,
		Name:    req.Name,
		Type:    catType,
		IconUrl: pgtype.Text{String: req.IconURL, Valid: req.IconURL != ""},
	}

	category, err := s.store.CreateCategory(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				ctx.JSON(http.StatusConflict, util.ErrorResponse(errors.New("kategori dengan nama dan tipe yang sama sudah ada")))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := categoryResponseFromParts(category.ID, category.UserID, category.Name, category.Type, category.IconUrl, category.CreatedAt, category.UpdatedAt)
	ctx.JSON(http.StatusCreated, rsp)
}

func (s *Server) getCategory(ctx *gin.Context) {
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

	catType := ctx.DefaultQuery("type", "EXPENSE")
	arg := db.GetCategoriesByUserParams{
		UserID: authPayload.UserID,
		Type:   catType,
	}

	categories, err := s.store.GetCategoriesByUser(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusOK, []categoryResponse{})
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	var rsp []categoryResponse
	for _, c := range categories {
		rsp = append(rsp, categoryResponseFromParts(c.ID, c.UserID, c.Name, c.Type, c.IconUrl, c.CreatedAt, c.UpdatedAt))
	}
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) getCategoryByID(ctx *gin.Context) {
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

	categoryIDStr := ctx.Param("id")
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	category, err := s.store.GetCategory(ctx, categoryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if category.UserID != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("forbidden: cannot access other user's category")))
		return
	}

	rsp := categoryResponseFromParts(category.ID, category.UserID, category.Name, category.Type, category.IconUrl, category.CreatedAt, category.UpdatedAt)
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) updateCategory(ctx *gin.Context) {
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

	categoryIDStr := ctx.Param("id")
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid category ID format")))
		return
	}

	var req updateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	existingCategory, err := s.store.GetCategory(ctx, categoryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("category not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if existingCategory.UserID != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("forbidden: cannot access other user's category")))
		return
	}

	catType := req.Type
	if catType == "" {
		catType = existingCategory.Type
	}

	arg := db.UpdateCategoryParams{
		ID:      categoryID,
		Name:    req.Name,
		IconUrl: pgtype.Text{String: req.IconURL, Valid: req.IconURL != ""},
		Type:    catType,
		UserID:  authPayload.UserID,
	}

	updatedCategory, err := s.store.UpdateCategory(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				ctx.JSON(http.StatusConflict, util.ErrorResponse(errors.New("kategori dengan nama dan tipe yang sama sudah ada")))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := categoryResponseFromParts(updatedCategory.ID, updatedCategory.UserID, updatedCategory.Name, updatedCategory.Type, updatedCategory.IconUrl, updatedCategory.CreatedAt, updatedCategory.UpdatedAt)
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) deleteCategory(ctx *gin.Context) {

	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization payload not found")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	categoryIDStr := ctx.Param("id")
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid category ID format")))
		return
	}

	existingCategory, err := s.store.GetCategory(ctx, categoryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("category not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if existingCategory.UserID != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("access forbidden: cannot delete other user's category")))
		return
	}

	arg := db.DeleteCategoryParams{
		ID:     categoryID,
		UserID: authPayload.UserID,
	}

	err = s.store.DeleteCategory(ctx, arg)
	if err != nil {

		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}
