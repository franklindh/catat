package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/franklindh/catat/db/mock"
	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func randomTransaction(userID, categoryID pgtype.UUID) db.Transaction {
	return db.Transaction{
		ID:              util.GoogleUUIDToPgxUUID(uuid.New()),
		UserID:          userID,
		CategoryID:      categoryID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CreatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func requireBodyMatchTransaction(t *testing.T, body *bytes.Buffer, transaction db.Transaction) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotTransaction transactionResponse
	err = json.Unmarshal(data, &gotTransaction)
	require.NoError(t, err)

	require.Equal(t, util.PgxUUIDToGoogleUUID(transaction.ID), gotTransaction.ID)
	require.Equal(t, util.PgxUUIDToGoogleUUID(transaction.UserID).String(), gotTransaction.UserID)
	require.Equal(t, util.PgxUUIDToGoogleUUID(transaction.CategoryID).String(), gotTransaction.CategoryID)

	transAmountRat, _ := transaction.Amount.Value()
	gotAmountRat, _ := gotTransaction.Amount.Value()
	require.Equal(t, 0, transAmountRat.(*big.Rat).Cmp(gotAmountRat.(*big.Rat)), "Amount values should be numerically equal")
	require.Equal(t, transaction.Description.String, gotTransaction.Description)

	require.WithinDuration(t, transaction.TransactionDate.Time, gotTransaction.TransactionDate.Time, time.Microsecond)

	require.WithinDuration(t, transaction.CreatedAt.Time, gotTransaction.CreatedAt.Time, time.Second)
}

func requireBodyMatchTransactions(t *testing.T, body *bytes.Buffer, transactions []db.Transaction) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotTransactions []transactionResponse
	err = json.Unmarshal(data, &gotTransactions)
	require.NoError(t, err)
	require.Len(t, gotTransactions, len(transactions))

	for i := range transactions {
		require.Equal(t, util.PgxUUIDToGoogleUUID(transactions[i].ID), gotTransactions[i].ID)
		require.Equal(t, util.PgxUUIDToGoogleUUID(transactions[i].UserID).String(), gotTransactions[i].UserID)
		require.Equal(t, util.PgxUUIDToGoogleUUID(transactions[i].CategoryID).String(), gotTransactions[i].CategoryID)
		transAmountRat, _ := transactions[i].Amount.Value()
		gotAmountRat, _ := gotTransactions[i].Amount.Value()
		require.Equal(t, 0, transAmountRat.(*big.Rat).Cmp(gotAmountRat.(*big.Rat)), "Amount values should be numerically equal for item %d", i)
		require.Equal(t, transactions[i].Description.String, gotTransactions[i].Description)
		require.WithinDuration(t, transactions[i].TransactionDate.Time, gotTransactions[i].TransactionDate.Time, time.Microsecond)

		require.WithinDuration(t, transactions[i].CreatedAt.Time, gotTransactions[i].CreatedAt.Time, time.Second)
	}
}

func TestCreateTransactionAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)
	transaction := randomTransaction(user.ID, category.ID)

	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"category_id":      util.PgxUUIDToGoogleUUID(category.ID).String(),
				"amount":           transaction.Amount,
				"description":      transaction.Description.String,
				"transaction_date": transaction.TransactionDate,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.CreateTransactionParams) (db.Transaction, error) {

						require.Equal(t, user.ID, arg.UserID)
						require.Equal(t, category.ID, arg.CategoryID)

						argAmountRat, _ := arg.Amount.Value()
						transAmountRat, _ := transaction.Amount.Value()
						require.Equal(t, 0, argAmountRat.(*big.Rat).Cmp(transAmountRat.(*big.Rat)), "Amount values should be numerically equal")
						require.Equal(t, transaction.Description, arg.Description)

						require.WithinDuration(t, transaction.TransactionDate.Time, arg.TransactionDate.Time, time.Microsecond)
						return transaction, nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchTransaction(t, recorder.Body, transaction)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"category_id":      util.PgxUUIDToGoogleUUID(category.ID).String(),
				"amount":           transaction.Amount,
				"description":      transaction.Description.String,
				"transaction_date": transaction.TransactionDate,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"category_id":      util.PgxUUIDToGoogleUUID(category.ID).String(),
				"amount":           transaction.Amount,
				"description":      transaction.Description.String,
				"transaction_date": transaction.TransactionDate,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Transaction{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidCategoryID",
			body: gin.H{
				"category_id": "invalid-uuid",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/transactions"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetTransactionAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)
	transaction := randomTransaction(user.ID, category.ID)

	testCases := []struct {
		name          string
		transactionID string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:          "OK",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTransaction(t, recorder.Body, transaction)
			},
		},
		{
			name:          "NoAuthorization",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:          "NotFound",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(db.Transaction{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:          "Forbidden",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:          "InternalError",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(db.Transaction{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:          "InvalidID",
			transactionID: "invalid-uuid",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/transactions/%s", tc.transactionID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListTransactionsAPI(t *testing.T) {
	user := randomUser()
	n := 5
	transactions := make([]db.Transaction, n)
	for i := 0; i < n; i++ {
		category := randomCategory(user.ID)
		transactions[i] = randomTransaction(user.ID, category.ID)
	}

	type Query struct {
		page_id   int32
		page_size int32
	}

	testCases := []struct {
		name          string
		query         Query
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			query: Query{
				page_id:   1,
				page_size: int32(n),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.GetTransactionsParams{
					UserID: user.ID,
					Limit:  int32(n),
					Offset: 0,
				}
				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(transactions, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTransactions(t, recorder.Body, transactions)
			},
		},
		{
			name: "NoAuthorization",
			query: Query{
				page_id:   1,
				page_size: int32(n),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError",
			query: Query{
				page_id:   1,
				page_size: int32(n),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.GetTransactionsParams{
					UserID: user.ID,
					Limit:  int32(n),
					Offset: 0,
				}
				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return([]db.Transaction{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPageID",
			query: Query{
				page_id:   0,
				page_size: int32(n),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidPageSize",
			query: Query{
				page_id:   1,
				page_size: 101,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := "/transactions"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			q := request.URL.Query()
			q.Add("page_id", fmt.Sprintf("%d", tc.query.page_id))
			q.Add("page_size", fmt.Sprintf("%d", tc.query.page_size))
			request.URL.RawQuery = q.Encode()

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestUpdateTransactionAPI(t *testing.T) {
	user := randomUser()
	category1 := randomCategory(user.ID)
	category2 := randomCategory(user.ID)
	transaction := randomTransaction(user.ID, category1.ID)
	updatedTransaction := transaction
	updatedTransaction.CategoryID = category2.ID
	updatedTransaction.Description = pgtype.Text{String: "Updated Description", Valid: true}

	testCases := []struct {
		name          string
		transactionID string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:          "OK",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			body: gin.H{
				"category_id": util.PgxUUIDToGoogleUUID(category2.ID).String(),
				"description": updatedTransaction.Description.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)

				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.UpdateTransactionParams) (db.Transaction, error) {

						require.Equal(t, transaction.ID, arg.ID)
						require.Equal(t, user.ID, arg.UserID)
						require.Equal(t, category2.ID, arg.CategoryID)
						require.Equal(t, transaction.Amount, arg.Amount)
						require.Equal(t, updatedTransaction.Description, arg.Description)
						require.Equal(t, transaction.TransactionDate, arg.TransactionDate)
						return updatedTransaction, nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTransaction(t, recorder.Body, updatedTransaction)
			},
		},
		{
			name:          "NoAuthorization",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:          "NotFound",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(db.Transaction{}, pgx.ErrNoRows)
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:          "Forbidden",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

				otherUser := randomUser()
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(otherUser.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:          "InternalErrorOnGet",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(db.Transaction{}, sql.ErrConnDone)
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:          "InternalErrorOnUpdate",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)

				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Transaction{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:          "InvalidID",
			transactionID: "invalid-uuid",
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/transactions/%s", tc.transactionID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteTransactionAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)
	transaction := randomTransaction(user.ID, category.ID)

	testCases := []struct {
		name          string
		transactionID string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:          "OK",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)

				arg := db.DeleteTransactionParams{
					ID:     transaction.ID,
					UserID: user.ID,
				}
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]string
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "transaction deleted successfully", response["message"])
			},
		},
		{
			name:          "NoAuthorization",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:          "NotFoundOnGet",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(db.Transaction{}, pgx.ErrNoRows)
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:          "Forbidden",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:          "InternalErrorOnGet",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(db.Transaction{}, sql.ErrConnDone)
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:          "InternalErrorOnDelete",
			transactionID: util.PgxUUIDToGoogleUUID(transaction.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetTransaction(gomock.Any(), transaction.ID).
					Times(1).
					Return(transaction, nil)

				arg := db.DeleteTransactionParams{
					ID:     transaction.ID,
					UserID: user.ID,
				}
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:          "InvalidID",
			transactionID: "invalid-uuid",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/transactions/%s", tc.transactionID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
