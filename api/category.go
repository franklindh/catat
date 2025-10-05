package api

import (
	"database/sql"
	"errors"
	"net/http"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type categoryResponse struct {
	ID        uuid.UUID          `json:"id"`
	UserID    string             `json:"user_id"`
	Name      string             `json:"name"`
	IconURL   string             `json:"icon_url"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

type createCategoryRequest struct {
	Name    string `json:"name" binding:"required"`
	IconURL string `json:"icon_url"`
}

type updateCategoryRequest struct {
	Name    string `json:"name" binding:"required"`
	IconURL string `json:"icon_url"`
}

func categoryToCategoryResponse(category db.Category) categoryResponse {
	return categoryResponse{
		ID:        util.PgxUUIDToGoogleUUID(category.ID),
		UserID:    util.PgxUUIDToGoogleUUID(category.UserID).String(),
		Name:      category.Name,
		IconURL:   category.IconUrl.String,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}

func categoriesToCategoryResponses(categories []db.Category) []categoryResponse {
	var responses []categoryResponse
	for _, category := range categories {
		responses = append(responses, categoryToCategoryResponse(category))
	}
	return responses
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

	arg := db.CreateCategoryParams{
		UserID:  util.GoogleUUIDToPgxUUID(authPayload.UserID),
		Name:    req.Name,
		IconUrl: pgtype.Text{String: req.IconURL, Valid: req.IconURL != ""},
	}

	category, err := s.store.CreateCategory(ctx, arg)
	if err != nil {

		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := categoryToCategoryResponse(category)
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

	categories, err := s.store.GetCategoriesByUser(ctx, util.GoogleUUIDToPgxUUID(authPayload.UserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusOK, []categoryResponse{})
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := categoriesToCategoryResponses(categories)
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
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	category, err := s.store.GetCategory(ctx, util.GoogleUUIDToPgxUUID(categoryID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if util.PgxUUIDToGoogleUUID(category.UserID) != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("forbidden: cannot access other user's category")))
		return
	}

	rsp := categoryToCategoryResponse(category)
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
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid category ID format")))
		return
	}

	var req updateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	existingCategory, err := s.store.GetCategory(ctx, util.GoogleUUIDToPgxUUID(categoryID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("category not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if util.PgxUUIDToGoogleUUID(existingCategory.UserID) != authPayload.UserID {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("forbidden: cannot access other user's category")))
		return
	}

	arg := db.UpdateCategoryParams{
		ID:      util.GoogleUUIDToPgxUUID(categoryID),
		Name:    req.Name,
		IconUrl: pgtype.Text{String: req.IconURL, Valid: req.IconURL != ""},
		UserID:  util.GoogleUUIDToPgxUUID(authPayload.UserID),
	}

	updatedCategory, err := s.store.UpdateCategory(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := categoryToCategoryResponse(updatedCategory)
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
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid category ID format")))
		return
	}

	existingCategory, err := s.store.GetCategory(ctx, util.GoogleUUIDToPgxUUID(categoryID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, util.ErrorResponse(errors.New("category not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	if existingCategory.UserID != util.GoogleUUIDToPgxUUID(authPayload.UserID) {
		ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("access forbidden: cannot delete other user's category")))
		return
	}

	arg := db.DeleteCategoryParams{
		ID:     util.GoogleUUIDToPgxUUID(categoryID),
		UserID: util.GoogleUUIDToPgxUUID(authPayload.UserID),
	}

	err = s.store.DeleteCategory(ctx, arg)
	if err != nil {

		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}
