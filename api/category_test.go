package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mock "github.com/franklindh/catat/db/mock"
	db "github.com/franklindh/catat/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCategoryAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()
	categoryID := uuid.New()

	expectedCategory := db.Category{
		ID:        pgtype.UUID{Bytes: categoryID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Name:      "Test Category",
		Type:      "income",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	mockStore.EXPECT().
		CreateCategory(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.CreateCategoryParams) (db.Category, error) {
			assert.Equal(t, userID[:], arg.UserID.Bytes[:])
			assert.Equal(t, "Test Category", arg.Name)
			assert.Equal(t, "income", arg.Type)
			return expectedCategory, nil
		})

	categoryReq := createCategoryRequest{
		UserID: userID.String(),
		Name:   "Test Category",
		Type:   "income",
	}
	jsonReq, _ := json.Marshal(categoryReq)

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseCategory db.Category
	err := json.Unmarshal(w.Body.Bytes(), &responseCategory)
	require.NoError(t, err)
	assert.Equal(t, expectedCategory.Name, responseCategory.Name)
	assert.Equal(t, expectedCategory.Type, responseCategory.Type)
}

func TestCreateCategoryAPI_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	testCases := []struct {
		name          string
		body          string
		expectedCode  int
		errorContains string
	}{
		{
			name:          "Invalid User ID",
			body:          `{"user_id": "invalid-uuid", "name": "Test Category", "type": "income"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "uuid",
		},
		{
			name:          "Missing Name",
			body:          `{"user_id": "550e8400-e29b-41d4-a716-446655440000", "type": "income"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "required",
		},
		{
			name:          "Invalid Type",
			body:          `{"user_id": "550e8400-e29b-41d4-a716-446655440000", "name": "Test Category", "type": "invalid"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "oneof",
		},
		{
			name:          "Empty Body",
			body:          `{}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.errorContains)
		})
	}
}

func TestCreateCategoryAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	mockStore.EXPECT().
		CreateCategory(gomock.Any(), gomock.Any()).
		Return(db.Category{}, errors.New("database error"))

	categoryReq := createCategoryRequest{
		UserID: userID.String(),
		Name:   "Test Category",
		Type:   "income",
	}
	jsonReq, _ := json.Marshal(categoryReq)

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestGetCategoryAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()
	userID := uuid.New()

	expectedCategory := db.Category{
		ID:        pgtype.UUID{Bytes: categoryID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Name:      "Test Category",
		Type:      "income",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	arg := db.GetCategoryParams{
		ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		GetCategory(gomock.Any(), arg).
		Return(expectedCategory, nil)

	req := httptest.NewRequest(http.MethodGet, "/categories/"+categoryID.String()+"?user_id="+userID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseCategory db.Category
	err := json.Unmarshal(w.Body.Bytes(), &responseCategory)
	require.NoError(t, err)
	assert.Equal(t, expectedCategory.ID, responseCategory.ID)
	assert.Equal(t, expectedCategory.Name, responseCategory.Name)
}

func TestGetCategoryAPI_InvalidURI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/categories/invalid-uuid?user_id=550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}

func TestGetCategoryAPI_InvalidQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/categories/"+categoryID.String()+"?user_id=invalid-uuid", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}

func TestGetCategoryAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()
	userID := uuid.New()

	arg := db.GetCategoryParams{
		ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		GetCategory(gomock.Any(), arg).
		Return(db.Category{}, errors.New("database error"))

	req := httptest.NewRequest(http.MethodGet, "/categories/"+categoryID.String()+"?user_id="+userID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestListCategoriesAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	categories := []db.Category{
		{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
			Name:      "Income Category",
			Type:      "income",
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
		{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
			Name:      "Expense Category",
			Type:      "expense",
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}

	mockStore.EXPECT().
		ListCategories(gomock.Any(), pgtype.UUID{Bytes: userID, Valid: true}).
		Return(categories, nil)

	req := httptest.NewRequest(http.MethodGet, "/categories?user_id="+userID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseCategories []db.Category
	err := json.Unmarshal(w.Body.Bytes(), &responseCategories)
	require.NoError(t, err)
	assert.Len(t, responseCategories, 2)
	assert.Equal(t, categories[0].Name, responseCategories[0].Name)
	assert.Equal(t, categories[1].Name, responseCategories[1].Name)
}

func TestListCategoriesAPI_InvalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/categories?user_id=invalid-uuid", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}

func TestListCategoriesAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	mockStore.EXPECT().
		ListCategories(gomock.Any(), pgtype.UUID{Bytes: userID, Valid: true}).
		Return([]db.Category{}, errors.New("database error"))

	req := httptest.NewRequest(http.MethodGet, "/categories?user_id="+userID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestUpdateCategoryAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()
	userID := uuid.New()

	updatedCategory := db.Category{
		ID:        pgtype.UUID{Bytes: categoryID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Name:      "Updated Category Name",
		Type:      "expense",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	arg := db.UpdateCategoryParams{
		ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
		Name:   "Updated Category Name",
		Type:   "expense",
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		UpdateCategory(gomock.Any(), arg).
		Return(updatedCategory, nil)

	updateReq := updateCategoryRequest{
		ID:     categoryID.String(),
		Name:   "Updated Category Name",
		Type:   "expense",
		UserID: userID.String(),
	}
	jsonReq, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/categories", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseCategory db.Category
	err := json.Unmarshal(w.Body.Bytes(), &responseCategory)
	require.NoError(t, err)
	assert.Equal(t, updatedCategory.Name, responseCategory.Name)
	assert.Equal(t, updatedCategory.Type, responseCategory.Type)
}

func TestUpdateCategoryAPI_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	testCases := []struct {
		name          string
		body          string
		expectedCode  int
		errorContains string
	}{
		{
			name:          "Invalid Category ID",
			body:          `{"id": "invalid-uuid", "name": "Updated Name", "type": "income", "user_id": "550e8400-e29b-41d4-a716-446655440000"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "uuid",
		},
		{
			name:          "Invalid User ID",
			body:          `{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "Updated Name", "type": "income", "user_id": "invalid-uuid"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "uuid",
		},
		{
			name:          "Missing Name",
			body:          `{"id": "550e8400-e29b-41d4-a716-446655440000", "type": "income", "user_id": "550e8400-e29b-41d4-a716-446655440000"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "required",
		},
		{
			name:          "Invalid Type",
			body:          `{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "Updated Name", "type": "invalid", "user_id": "550e8400-e29b-41d4-a716-446655440000"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "oneof",
		},
		{
			name:          "Empty Body",
			body:          `{}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/categories", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.errorContains)
		})
	}
}

func TestUpdateCategoryAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()
	userID := uuid.New()

	arg := db.UpdateCategoryParams{
		ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
		Name:   "Updated Category Name",
		Type:   "expense",
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		UpdateCategory(gomock.Any(), arg).
		Return(db.Category{}, errors.New("database error"))

	updateReq := updateCategoryRequest{
		ID:     categoryID.String(),
		Name:   "Updated Category Name",
		Type:   "expense",
		UserID: userID.String(),
	}
	jsonReq, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/categories", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestDeleteCategoryAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()
	userID := uuid.New()

	arg := db.DeleteCategoryParams{
		ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		DeleteCategory(gomock.Any(), arg).
		Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/categories/"+categoryID.String()+"?user_id="+userID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	message, exists := response["message"]
	assert.True(t, exists)
	assert.Equal(t, "category deleted successfully", message)
}

func TestDeleteCategoryAPI_InvalidURI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/categories/invalid-uuid?user_id=550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}

func TestDeleteCategoryAPI_InvalidQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/categories/"+categoryID.String()+"?user_id=invalid-uuid", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}

func TestDeleteCategoryAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	categoryID := uuid.New()
	userID := uuid.New()

	arg := db.DeleteCategoryParams{
		ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		DeleteCategory(gomock.Any(), arg).
		Return(errors.New("database error"))

	req := httptest.NewRequest(http.MethodDelete, "/categories/"+categoryID.String()+"?user_id="+userID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}
