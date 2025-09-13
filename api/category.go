package api

import (
	"net/http"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
)

type createCategoryRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required,oneof=income expense"`
}

type listCategoriesRequest struct {
	UserID string `form:"user_id" binding:"required,uuid"`
}

type updateCategoryRequest struct {
	ID     string `json:"id" binding:"required,uuid"`
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required,oneof=income expense"`
	UserID string `json:"user_id" binding:"required,uuid"`
}

func (s *Server) createCategory(ctx *gin.Context) {
	var req createCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.CreateCategoryParams{
		UserID: userID,
		Name:   req.Name,
		Type:   req.Type,
	}

	category, err := s.Store.CreateCategory(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, category)
}

func (s *Server) getCategory(ctx *gin.Context) {
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

	categoryID, err := util.ParseUUID(uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid category ID format"))
		return
	}

	userID, err := util.ParseUUID(queryReq.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.GetCategoryParams{
		ID:     categoryID,
		UserID: userID,
	}

	category, err := s.Store.GetCategory(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, category)
}

func (s *Server) listCategories(ctx *gin.Context) {
	var req listCategoriesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	categories, err := s.Store.ListCategories(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, categories)
}

func (s *Server) updateCategory(ctx *gin.Context) {
	var req updateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(err))
		return
	}

	categoryID, err := util.ParseUUID(req.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid category ID format"))
		return
	}

	userID, err := util.ParseUUID(req.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.UpdateCategoryParams{
		ID:     categoryID,
		Name:   req.Name,
		Type:   req.Type,
		UserID: userID,
	}

	category, err := s.Store.UpdateCategory(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, category)
}

func (s *Server) deleteCategory(ctx *gin.Context) {
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

	categoryID, err := util.ParseUUID(uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid category ID format"))
		return
	}

	userID, err := util.ParseUUID(queryReq.UserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponseWithMessage("invalid user ID format"))
		return
	}

	arg := db.DeleteCategoryParams{
		ID:     categoryID,
		UserID: userID,
	}

	err = s.Store.DeleteCategory(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}
