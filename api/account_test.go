package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mock "github.com/franklindh/catat/db/mock"
	db "github.com/franklindh/catat/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccountAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()
	accountID := uuid.New()

	expectedAccount := db.Account{
		ID:        pgtype.UUID{Bytes: accountID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Name:      "Test Account",
		Type:      "depository",
		Balance:   pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	mockStore.EXPECT().
		CreateAccount(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.CreateAccountParams) (db.Account, error) {

			assert.Equal(t, userID[:], arg.UserID.Bytes[:])
			assert.Equal(t, "Test Account", arg.Name)
			assert.Equal(t, "depository", arg.Type)
			return expectedAccount, nil
		})

	accountReq := createAccountRequest{
		UserID: userID.String(),
		Name:   "Test Account",
		Type:   "depository",
	}
	jsonReq, _ := json.Marshal(accountReq)

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseAccount db.Account
	err := json.Unmarshal(w.Body.Bytes(), &responseAccount)
	require.NoError(t, err)
	assert.Equal(t, expectedAccount.Name, responseAccount.Name)
	assert.Equal(t, expectedAccount.Type, responseAccount.Type)
}

func TestCreateAccountAPI_InvalidInput(t *testing.T) {
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
			body:          `{"user_id": "invalid-uuid", "name": "Test Account", "type": "depository"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "uuid",
		},
		{
			name:          "Missing Name",
			body:          `{"user_id": "550e8400-e29b-41d4-a716-446655440000", "type": "depository"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "required",
		},
		{
			name:          "Invalid Type",
			body:          `{"user_id": "550e8400-e29b-41d4-a716-446655440000", "name": "Test Account", "type": "invalid"}`,
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
			req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.errorContains)
		})
	}
}

func TestCreateAccountAPI_StoreError(t *testing.T) {
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
		CreateAccount(gomock.Any(), gomock.Any()).
		Return(db.Account{}, errors.New("database error"))

	accountReq := createAccountRequest{
		UserID: userID.String(),
		Name:   "Test Account",
		Type:   "depository",
	}
	jsonReq, _ := json.Marshal(accountReq)

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestGetAccountAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	accountID := uuid.New()
	userID := uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb")

	expectedAccount := db.Account{
		ID:        pgtype.UUID{Bytes: accountID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Name:      "Test Account",
		Type:      "depository",
		Balance:   pgtype.Numeric{Int: big.NewInt(1000000), Exp: -4, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	arg := db.GetAccountParams{
		ID:     pgtype.UUID{Bytes: accountID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		GetAccount(gomock.Any(), arg).
		Return(expectedAccount, nil)

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseAccount db.Account
	err := json.Unmarshal(w.Body.Bytes(), &responseAccount)
	require.NoError(t, err)
	assert.Equal(t, expectedAccount.ID, responseAccount.ID)
	assert.Equal(t, expectedAccount.Name, responseAccount.Name)
}

func TestGetAccountAPI_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	accountID := uuid.New()
	userID := uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb")

	arg := db.GetAccountParams{
		ID:     pgtype.UUID{Bytes: accountID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		GetAccount(gomock.Any(), arg).
		Return(db.Account{}, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAccountAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	accountID := uuid.New()
	userID := uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb")

	arg := db.GetAccountParams{
		ID:     pgtype.UUID{Bytes: accountID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		GetAccount(gomock.Any(), arg).
		Return(db.Account{}, errors.New("database error"))

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestGetAccountAPI_InvalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/accounts/invalid-uuid", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}

func TestListAccountsAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	accounts := []db.Account{
		{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
			Name:      "Account 1",
			Type:      "depository",
			Balance:   pgtype.Numeric{Int: big.NewInt(1000000), Exp: -4, Valid: true},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
		{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
			Name:      "Account 2",
			Type:      "credit",
			Balance:   pgtype.Numeric{Int: big.NewInt(-500000), Exp: -4, Valid: true},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}

	listArg := db.ListAccountsParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  10,
		Offset: 0,
	}

	countArg := pgtype.UUID{Bytes: userID, Valid: true}

	mockStore.EXPECT().
		ListAccounts(gomock.Any(), listArg).
		Return(accounts, nil)

	mockStore.EXPECT().
		CountAccountsByUser(gomock.Any(), countArg).
		Return(int64(2), nil)

	req := httptest.NewRequest(http.MethodGet, "/accounts?user_id="+userID.String()+"&page=1&limit=10", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	_, exists := response["data"]
	assert.True(t, exists)

	_, exists = response["pagination"]
	assert.True(t, exists)
}

func TestListAccountsAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	listArg := db.ListAccountsParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  10,
		Offset: 0,
	}

	mockStore.EXPECT().
		ListAccounts(gomock.Any(), listArg).
		Return([]db.Account{}, errors.New("database error"))

	req := httptest.NewRequest(http.MethodGet, "/accounts?user_id="+userID.String()+"&page=1&limit=10", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestListAccountsAPI_CountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	listArg := db.ListAccountsParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  10,
		Offset: 0,
	}

	countArg := pgtype.UUID{Bytes: userID, Valid: true}

	accounts := []db.Account{
		{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
			Name:      "Account 1",
			Type:      "depository",
			Balance:   pgtype.Numeric{Int: big.NewInt(1000000), Exp: -4, Valid: true},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}

	mockStore.EXPECT().
		ListAccounts(gomock.Any(), listArg).
		Return(accounts, nil)

	mockStore.EXPECT().
		CountAccountsByUser(gomock.Any(), countArg).
		Return(int64(0), errors.New("count error"))

	req := httptest.NewRequest(http.MethodGet, "/accounts?user_id="+userID.String()+"&page=1&limit=10", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, exists := response["data"]
	assert.True(t, exists)
	assert.Len(t, data, 1)

	pagination, exists := response["pagination"]
	assert.True(t, exists)
	assert.NotContains(t, pagination, "total")
}

func TestListAccountsAPI_InvalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/accounts?user_id=invalid-uuid&page=1&limit=10", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}

func TestListAccountsAPI_DefaultPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	accounts := []db.Account{
		{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
			Name:      "Account 1",
			Type:      "depository",
			Balance:   pgtype.Numeric{Int: big.NewInt(1000000), Exp: -4, Valid: true},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}

	listArg := db.ListAccountsParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  10,
		Offset: 0,
	}

	countArg := pgtype.UUID{Bytes: userID, Valid: true}

	mockStore.EXPECT().
		ListAccounts(gomock.Any(), listArg).
		Return(accounts, nil)

	mockStore.EXPECT().
		CountAccountsByUser(gomock.Any(), countArg).
		Return(int64(1), nil)

	req := httptest.NewRequest(http.MethodGet, "/accounts?user_id="+userID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	pagination, exists := response["pagination"].(map[string]interface{})
	assert.True(t, exists)
	assert.Equal(t, float64(1), pagination["page"])
	assert.Equal(t, float64(10), pagination["limit"])
	assert.Equal(t, float64(1), pagination["totalPages"])
}

func TestListAccountsAPI_LimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	userID := uuid.New()

	accounts := make([]db.Account, 0)

	listArg := db.ListAccountsParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  100,
		Offset: 0,
	}

	countArg := pgtype.UUID{Bytes: userID, Valid: true}

	mockStore.EXPECT().
		ListAccounts(gomock.Any(), listArg).
		Return(accounts, nil)

	mockStore.EXPECT().
		CountAccountsByUser(gomock.Any(), countArg).
		Return(int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/accounts?user_id="+userID.String()+"&limit=150", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateAccountAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	accountID := uuid.New()
	userID := uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb")

	updatedAccount := db.Account{
		ID:        pgtype.UUID{Bytes: accountID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Name:      "Updated Account Name",
		Type:      "credit",
		Balance:   pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	arg := db.UpdateAccountParams{
		ID:     pgtype.UUID{Bytes: accountID, Valid: true},
		Name:   "Updated Account Name",
		Type:   "credit",
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		UpdateAccount(gomock.Any(), arg).
		Return(updatedAccount, nil)

	updateReq := updateAccountRequest{
		ID:   accountID.String(),
		Name: "Updated Account Name",
		Type: "credit",
	}
	jsonReq, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/accounts", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseAccount db.Account
	err := json.Unmarshal(w.Body.Bytes(), &responseAccount)
	require.NoError(t, err)
	assert.Equal(t, updatedAccount.Name, responseAccount.Name)
	assert.Equal(t, updatedAccount.Type, responseAccount.Type)
}

func TestUpdateAccountAPI_InvalidInput(t *testing.T) {
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
			name:          "Invalid Account ID",
			body:          `{"id": "invalid-uuid", "name": "Updated Name", "type": "depository"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "uuid",
		},
		{
			name:          "Missing Name",
			body:          `{"id": "550e8400-e29b-41d4-a716-446655440000", "type": "depository"}`,
			expectedCode:  http.StatusBadRequest,
			errorContains: "required",
		},
		{
			name:          "Invalid Type",
			body:          `{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "Updated Name", "type": "invalid"}`,
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
			req := httptest.NewRequest(http.MethodPut, "/accounts", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.errorContains)
		})
	}
}

func TestUpdateAccountAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	accountID := uuid.New()
	userID := uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb")

	arg := db.UpdateAccountParams{
		ID:     pgtype.UUID{Bytes: accountID, Valid: true},
		Name:   "Updated Account Name",
		Type:   "credit",
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		UpdateAccount(gomock.Any(), arg).
		Return(db.Account{}, errors.New("database error"))

	updateReq := updateAccountRequest{
		ID:   accountID.String(),
		Name: "Updated Account Name",
		Type: "credit",
	}
	jsonReq, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/accounts", bytes.NewBuffer(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestDeleteAccountAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	accountID := uuid.New()
	userID := uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb")

	arg := db.DeleteAccountParams{
		ID:     pgtype.UUID{Bytes: accountID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		DeleteAccount(gomock.Any(), arg).
		Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	message, exists := response["message"]
	assert.True(t, exists)
	assert.Equal(t, "account deleted successfully", message)
}

func TestDeleteAccountAPI_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	accountID := uuid.New()
	userID := uuid.MustParse("022e7078-bf1c-4af0-b306-2bf92ba8f8eb")

	arg := db.DeleteAccountParams{
		ID:     pgtype.UUID{Bytes: accountID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}

	mockStore.EXPECT().
		DeleteAccount(gomock.Any(), arg).
		Return(errors.New("database error"))

	req := httptest.NewRequest(http.MethodDelete, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
}

func TestDeleteAccountAPI_InvalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	server := &Server{
		Store:  mockStore,
		Router: gin.Default(),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/accounts/invalid-uuid", nil)
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uuid")
}
