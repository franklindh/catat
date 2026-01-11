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
	"strconv"
	"testing"
	"time"

	mockdb "github.com/franklindh/catat/db/mock"
	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func formatNumericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return "0"
	}
	val, err := n.Value()
	if err != nil {
		return "0"
	}
	if str, ok := val.(string); ok {
		return str
	}
	if rat, ok := val.(*big.Rat); ok {
		return rat.FloatString(4)
	}
	return "0"
}

func randomTransaction(userID int64, categoryID *int64) db.CreateTransactionRow {
	var catID pgtype.Int8
	if categoryID != nil {
		catID = pgtype.Int8{Int64: *categoryID, Valid: true}
	}
	return db.CreateTransactionRow{
		ID:              util.RandomInt(1, 1000000),
		UserID:          userID,
		CategoryID:      catID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Type:            "EXPENSE",
		CreatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func randomGetTransactionRow(userID int64, categoryID *int64) db.GetTransactionRow {
	var catID pgtype.Int8
	if categoryID != nil {
		catID = pgtype.Int8{Int64: *categoryID, Valid: true}
	}
	return db.GetTransactionRow{
		ID:              util.RandomInt(1, 1000000),
		UserID:          userID,
		CategoryID:      catID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Type:            "EXPENSE",
	}
}

func requireBodyMatchTransaction(t *testing.T, body *bytes.Buffer, transactionID int64, userID int64, categoryID *int64, amount pgtype.Numeric, description string, transactionDate pgtype.Timestamptz, transactionType string) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotTransaction transactionResponse
	err = json.Unmarshal(data, &gotTransaction)
	require.NoError(t, err)

	require.Equal(t, transactionID, gotTransaction.ID)
	require.Equal(t, userID, gotTransaction.UserID)
	if categoryID != nil {
		require.NotNil(t, gotTransaction.CategoryID)
		require.Equal(t, *categoryID, *gotTransaction.CategoryID)
	} else {
		require.Nil(t, gotTransaction.CategoryID)
	}

	transAmountRat, _ := amount.Value()
	gotAmountRat, _ := gotTransaction.Amount.Value()
	if transRat, ok := transAmountRat.(*big.Rat); ok {
		if gotRat, ok := gotAmountRat.(*big.Rat); ok {
			require.Equal(t, 0, transRat.Cmp(gotRat), "Amount values should be numerically equal")
		} else if gotStr, ok := gotAmountRat.(string); ok {

			gotRat, _, _ := big.ParseFloat(gotStr, 10, 0, big.ToNearestEven)
			transFloat, _ := new(big.Float).SetRat(transRat).Float64()
			gotFloat, _ := gotRat.Float64()
			require.InDelta(t, transFloat, gotFloat, 0.0001, "Amount values should be numerically equal")
		}
	}
	require.Equal(t, description, gotTransaction.Description)
	require.Equal(t, transactionType, gotTransaction.Type)

	require.WithinDuration(t, transactionDate.Time, gotTransaction.TransactionDate.Time, time.Microsecond)
}

func requireBodyMatchTransactions(t *testing.T, body *bytes.Buffer, transactions []db.ListTransactionsRow) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotTransactions []transactionResponse
	err = json.Unmarshal(data, &gotTransactions)
	require.NoError(t, err)
	require.Len(t, gotTransactions, len(transactions))

	for i := range transactions {
		require.Equal(t, transactions[i].ID, gotTransactions[i].ID)
		require.Equal(t, transactions[i].UserID, gotTransactions[i].UserID)
		if transactions[i].CategoryID.Valid {
			require.NotNil(t, gotTransactions[i].CategoryID)
			require.Equal(t, transactions[i].CategoryID.Int64, *gotTransactions[i].CategoryID)
		} else {
			require.Nil(t, gotTransactions[i].CategoryID)
		}

		transAmountRat, _ := transactions[i].Amount.Value()
		gotAmountRat, _ := gotTransactions[i].Amount.Value()
		if transRat, ok := transAmountRat.(*big.Rat); ok {
			if gotRat, ok := gotAmountRat.(*big.Rat); ok {
				require.Equal(t, 0, transRat.Cmp(gotRat), "Amount values should be numerically equal for item %d", i)
			} else if gotStr, ok := gotAmountRat.(string); ok {

				gotRat, _, _ := big.ParseFloat(gotStr, 10, 0, big.ToNearestEven)
				transFloat, _ := new(big.Float).SetRat(transRat).Float64()
				gotFloat, _ := gotRat.Float64()
				require.InDelta(t, transFloat, gotFloat, 0.0001, "Amount values should be numerically equal for item %d", i)
			}
		}
		require.Equal(t, transactions[i].Description.String, gotTransactions[i].Description)
		require.Equal(t, transactions[i].Type, gotTransactions[i].Type)
		require.WithinDuration(t, transactions[i].TransactionDate.Time, gotTransactions[i].TransactionDate.Time, time.Microsecond)
	}
}

func TestCreateTransactionAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)
	categoryID := category.ID
	transaction := randomTransaction(user.ID, &categoryID)

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
				"category_id":      categoryID,
				"amount":           formatNumericToString(transaction.Amount),
				"description":      transaction.Description.String,
				"transaction_date": transaction.TransactionDate.Time.Format(time.RFC3339),
				"type":             transaction.Type,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), user.ID).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.CreateTransactionParams) (db.CreateTransactionRow, error) {
						require.Equal(t, user.ID, arg.UserID)
						require.True(t, arg.CategoryID.Valid)
						require.Equal(t, categoryID, arg.CategoryID.Int64)

						argAmountRat, _ := arg.Amount.Value()
						transAmountRat, _ := transaction.Amount.Value()
						if argRat, ok := argAmountRat.(*big.Rat); ok {
							if transRat, ok := transAmountRat.(*big.Rat); ok {
								require.Equal(t, 0, argRat.Cmp(transRat), "Amount values should be numerically equal")
							}
						} else {

							require.True(t, arg.Amount.Valid == transaction.Amount.Valid)
						}
						require.Equal(t, transaction.Description, arg.Description)
						require.Equal(t, transaction.Type, arg.Type)

						require.WithinDuration(t, transaction.TransactionDate.Time, arg.TransactionDate.Time, time.Second)
						return transaction, nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchTransaction(t, recorder.Body, transaction.ID, transaction.UserID, &categoryID, transaction.Amount, transaction.Description.String, transaction.TransactionDate, transaction.Type)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"category_id":      categoryID,
				"amount":           formatNumericToString(transaction.Amount),
				"description":      transaction.Description.String,
				"transaction_date": transaction.TransactionDate.Time.Format(time.RFC3339),
				"type":             transaction.Type,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)
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
				"category_id":      categoryID,
				"amount":           formatNumericToString(transaction.Amount),
				"description":      transaction.Description.String,
				"transaction_date": transaction.TransactionDate.Time.Format(time.RFC3339),
				"type":             transaction.Type,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), user.ID).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateTransactionRow{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidAmount",
			body: gin.H{
				"amount": "invalid",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)
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
	otherUser := randomUser()
	category := randomCategory(user.ID)
	categoryID := category.ID
	transaction := randomGetTransactionRow(user.ID, &categoryID)

	testCases := []struct {
		name          string
		transactionID string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:          "OK",
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(transaction, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var catID *int64
				if transaction.CategoryID.Valid {
					catID = &transaction.CategoryID.Int64
				}
				requireBodyMatchTransaction(t, recorder.Body, transaction.ID, transaction.UserID, catID, transaction.Amount, transaction.Description.String, transaction.TransactionDate, transaction.Type)
			},
		},
		{
			name:          "NoAuthorization",
			transactionID: strconv.FormatInt(transaction.ID, 10),
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:          "Forbidden",
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, otherUser.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: otherUser.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:          "InternalError",
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:          "InvalidID",
			transactionID: "invalid-id",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
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
	transactions := make([]db.ListTransactionsRow, n)
	for i := 0; i < n; i++ {
		category := randomCategory(user.ID)
		categoryID := category.ID
		transactions[i] = db.ListTransactionsRow{
			ID:              util.RandomInt(1, 1000000),
			UserID:          user.ID,
			CategoryID:      pgtype.Int8{Int64: categoryID, Valid: true},
			Amount:          util.RandomBalance(),
			Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
			TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			Type:            "EXPENSE",
		}
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListTransactionsParams{
					UserID: user.ID,
					Limit:  int32(n),
					Offset: 0,
				}
				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Eq(arg)).
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
					ListTransactions(gomock.Any(), gomock.Any()).
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListTransactionsParams{
					UserID: user.ID,
					Limit:  int32(n),
					Offset: 0,
				}
				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return([]db.ListTransactionsRow{}, sql.ErrConnDone)
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Any()).
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Any()).
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

			url := "/transaction"
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
	otherUser := randomUser()
	category1 := randomCategory(user.ID)
	category2 := randomCategory(user.ID)
	category1ID := category1.ID
	category2ID := category2.ID
	transaction := randomGetTransactionRow(user.ID, &category1ID)
	updatedTransaction := db.UpdateTransactionRow{
		ID:              transaction.ID,
		UserID:          transaction.UserID,
		CategoryID:      pgtype.Int8{Int64: category2ID, Valid: true},
		Amount:          transaction.Amount,
		Description:     pgtype.Text{String: "Updated Description", Valid: true},
		TransactionDate: transaction.TransactionDate,
		Type:            transaction.Type,
	}

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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			body: gin.H{
				"category_id": category2ID,
				"description": updatedTransaction.Description.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(transaction, nil)

				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.UpdateTransactionParams) (db.UpdateTransactionRow, error) {
						require.Equal(t, transaction.ID, arg.ID)
						require.Equal(t, user.ID, arg.UserID)
						require.True(t, arg.CategoryID.Valid)
						require.Equal(t, category2ID, arg.CategoryID.Int64)
						require.Equal(t, transaction.Amount, arg.Amount)
						require.Equal(t, updatedTransaction.Description, arg.Description)
						require.Equal(t, transaction.TransactionDate, arg.TransactionDate)
						return updatedTransaction, nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var catID *int64
				if updatedTransaction.CategoryID.Valid {
					catID = &updatedTransaction.CategoryID.Int64
				}
				requireBodyMatchTransaction(t, recorder.Body, updatedTransaction.ID, updatedTransaction.UserID, catID, updatedTransaction.Amount, updatedTransaction.Description.String, updatedTransaction.TransactionDate, updatedTransaction.Type)
			},
		},
		{
			name:          "NoAuthorization",
			transactionID: strconv.FormatInt(transaction.ID, 10),
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, pgx.ErrNoRows)
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, otherUser.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: otherUser.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, pgx.ErrNoRows)
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:          "InternalErrorOnGet",
			transactionID: strconv.FormatInt(transaction.ID, 10),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, sql.ErrConnDone)
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(transaction, nil)
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.UpdateTransactionRow{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:          "InvalidID",
			transactionID: "invalid-id",
			body: gin.H{
				"description": "Updated Description",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
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
	otherUser := randomUser()
	category := randomCategory(user.ID)
	categoryID := category.ID
	transaction := randomGetTransactionRow(user.ID, &categoryID)

	testCases := []struct {
		name          string
		transactionID string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:          "OK",
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, pgx.ErrNoRows)
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, otherUser.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: otherUser.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, pgx.ErrNoRows)
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:          "InternalErrorOnGet",
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
					Times(1).
					Return(db.GetTransactionRow{}, sql.ErrConnDone)
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
			transactionID: strconv.FormatInt(transaction.ID, 10),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), db.GetTransactionParams{
						ID:     transaction.ID,
						UserID: user.ID,
					}).
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
			transactionID: "invalid-id",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, time.Minute)
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
