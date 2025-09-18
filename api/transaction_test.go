package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/franklindh/catat/db/mock"
	db "github.com/franklindh/catat/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

var transactionID = uuid.MustParse("00f4bd09-93e6-4061-9188-9518104ff8a8")

func TestCreateTransactionAPI(t *testing.T) {
	transactionDate := time.Now()

	testCases := []struct {
		name          string
		body          string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: fmt.Sprintf(`{
				"user_id": "%s",
				"account_id": "%s",
				"category_id": "%s",
				"amount": 100.50,
				"description": "Test Transaction",
				"transaction_date": "%s"
			}`, userID.String(), accountID.String(), categoryID.String(), transactionDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.CreateTransactionParams) (db.Transaction, error) {
						require.True(t, arg.UserID.Valid)
						require.True(t, arg.AccountID.Valid)
						require.True(t, arg.CategoryID.Valid)
						require.Equal(t, "Test Transaction", arg.Description)
						return db.Transaction{
							ID:         pgtype.UUID{Bytes: [16]byte{}, Valid: true},
							UserID:     arg.UserID,
							AccountID:  arg.AccountID,
							CategoryID: arg.CategoryID,
							Amount: pgtype.Numeric{
								Int:   big.NewInt(1005000),
								Exp:   -4,
								Valid: true,
							},
							Description:     "Test Transaction",
							TransactionDate: pgtype.Timestamptz{Time: transactionDate, Valid: true},
							CreatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "InvalidUserID",
			body: fmt.Sprintf(`{
				"user_id": "invalid-uuid",
				"account_id": "%s",
				"category_id": "%s",
				"amount": 100.50,
				"description": "Test Transaction",
				"transaction_date": "%s"
			}`, accountID.String(), categoryID.String(), transactionDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidAccountID",
			body: fmt.Sprintf(`{
				"user_id": "%s",
				"account_id": "invalid-uuid",
				"category_id": "%s",
				"amount": 100.50,
				"description": "Test Transaction",
				"transaction_date": "%s"
			}`, userID.String(), categoryID.String(), transactionDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidCategoryID",
			body: fmt.Sprintf(`{
				"user_id": "%s",
				"account_id": "%s",
				"category_id": "invalid-uuid",
				"amount": 100.50,
				"description": "Test Transaction",
				"transaction_date": "%s"
			}`, userID.String(), accountID.String(), transactionDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "MissingRequiredFields",
			body: `{}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "DatabaseError",
			body: fmt.Sprintf(`{
				"user_id": "%s",
				"account_id": "%s",
				"category_id": "%s",
				"amount": 100.50,
				"description": "Test Transaction",
				"transaction_date": "%s"
			}`, userID.String(), accountID.String(), categoryID.String(), transactionDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateTransaction(gomock.Any(), gomock.Any()).
					Return(db.Transaction{}, sql.ErrConnDone).
					Times(1)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.setupMock(store)

			server := &Server{
				Store:  store,
				Router: gin.Default(),
			}
			server.setupRoutes()

			req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)

			tc.checkResponse(w)
		})
	}
}

func TestGetTransactionAPI(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  fmt.Sprintf("/transactions/%s?user_id=%s", transactionID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetTransactionParams{
					ID:     pgtype.UUID{Bytes: transactionID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Eq(arg)).
					Return(db.Transaction{
						ID:         pgtype.UUID{Bytes: transactionID, Valid: true},
						UserID:     pgtype.UUID{Bytes: userID, Valid: true},
						AccountID:  pgtype.UUID{Bytes: accountID, Valid: true},
						CategoryID: pgtype.UUID{Bytes: categoryID, Valid: true},
						Amount: pgtype.Numeric{
							Int:   big.NewInt(1005000),
							Exp:   -4,
							Valid: true,
						},
						Description:     "Test Transaction",
						TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						CreatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseTransaction db.Transaction
				err := json.Unmarshal(recorder.Body.Bytes(), &responseTransaction)
				require.NoError(t, err)
				require.Equal(t, "Test Transaction", responseTransaction.Description)
			},
		},
		{
			name: "NotFound",
			url:  fmt.Sprintf("/transactions/%s?user_id=%s", transactionID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetTransactionParams{
					ID:     pgtype.UUID{Bytes: transactionID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Eq(arg)).
					Return(db.Transaction{}, sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InvalidTransactionID",
			url:  fmt.Sprintf("/transactions/invalid-uuid?user_id=%s", userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "InvalidUserID",
			url:  fmt.Sprintf("/transactions/%s?user_id=invalid-uuid", transactionID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "StoreError",
			url:  fmt.Sprintf("/transactions/%s?user_id=%s", transactionID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetTransactionParams{
					ID:     pgtype.UUID{Bytes: transactionID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetTransaction(gomock.Any(), gomock.Eq(arg)).
					Return(db.Transaction{}, errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.setupMock(store)

			server := &Server{
				Store:  store,
				Router: gin.Default(),
			}
			server.setupRoutes()

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestListTransactionsAPI(t *testing.T) {

	baseTime := time.Now().Truncate(time.Second).UTC()
	startDate := baseTime.AddDate(0, 0, -7)
	endDate := baseTime

	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "ListTransactionsOK",
			url:  "/transactions?user_id=" + userID.String() + "&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListTransactionsParams{
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
					Limit:  10,
					Offset: 0,
				}

				transactions := []db.Transaction{
					{
						ID:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
						UserID:     pgtype.UUID{Bytes: userID, Valid: true},
						AccountID:  pgtype.UUID{Bytes: accountID, Valid: true},
						CategoryID: pgtype.UUID{Bytes: categoryID, Valid: true},
						Amount: pgtype.Numeric{
							Int:   big.NewInt(1005000),
							Exp:   -4,
							Valid: true,
						},
						Description: "Transaction 1",
						TransactionDate: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second).AddDate(0, 0, -1),
							Valid: true,
						},
						CreatedAt: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second).AddDate(0, 0, -1),
							Valid: true,
						},
					},
				}

				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Eq(listArg)).
					Return(transactions, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response []map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response, 1)
			},
		},
		{
			name: "ListTransactionsByAccountOK",
			url:  "/transactions/account?user_id=" + userID.String() + "&account_id=" + accountID.String() + "&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListTransactionsByAccountParams{
					UserID:    pgtype.UUID{Bytes: userID, Valid: true},
					AccountID: pgtype.UUID{Bytes: accountID, Valid: true},
					Limit:     10,
					Offset:    0,
				}

				transactions := []db.Transaction{
					{
						ID:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
						UserID:     pgtype.UUID{Bytes: userID, Valid: true},
						AccountID:  pgtype.UUID{Bytes: accountID, Valid: true},
						CategoryID: pgtype.UUID{Bytes: categoryID, Valid: true},
						Amount: pgtype.Numeric{
							Int:   big.NewInt(1005000),
							Exp:   -4,
							Valid: true,
						},
						Description: "Account Transaction 1",
						TransactionDate: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second).AddDate(0, 0, -1),
							Valid: true,
						},
						CreatedAt: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second).AddDate(0, 0, -1),
							Valid: true,
						},
					},
				}

				store.EXPECT().
					ListTransactionsByAccount(gomock.Any(), gomock.Eq(listArg)).
					Return(transactions, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response []map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response, 1)
			},
		},
		{
			name: "ListTransactionsByDateRangeOK",
			url: fmt.Sprintf("/transactions/date-range?user_id=%s&start_date=%s&end_date=%s",
				userID.String(),
				startDate.Format(time.RFC3339),
				endDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListTransactionsByDateRangeParams{
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
					TransactionDate: pgtype.Timestamptz{
						Time:  startDate,
						Valid: true,
					},
					TransactionDate_2: pgtype.Timestamptz{
						Time:  endDate,
						Valid: true,
					},
				}

				transactions := []db.Transaction{
					{
						ID:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
						UserID:     pgtype.UUID{Bytes: userID, Valid: true},
						AccountID:  pgtype.UUID{Bytes: accountID, Valid: true},
						CategoryID: pgtype.UUID{Bytes: categoryID, Valid: true},
						Amount: pgtype.Numeric{
							Int:   big.NewInt(1005000),
							Exp:   -4,
							Valid: true,
						},
						Description: "Date Range Transaction 1",
						TransactionDate: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second).AddDate(0, 0, -1),
							Valid: true,
						},
						CreatedAt: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second).AddDate(0, 0, -1),
							Valid: true,
						},
					},
				}

				store.EXPECT().
					ListTransactionsByDateRange(gomock.Any(), gomock.Eq(listArg)).
					Return(transactions, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response []map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response, 1)
			},
		},
		{
			name: "InvalidUserID",
			url:  "/transactions?user_id=invalid-uuid&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "InvalidAccountID",
			url:  "/transactions/account?user_id=" + userID.String() + "&account_id=invalid-uuid&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTransactionsByAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "InvalidStartDate",
			url: fmt.Sprintf("/transactions/date-range?user_id=%s&start_date=invalid&end_date=%s",
				userID.String(),
				endDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTransactionsByDateRange(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "StoreError",
			url:  "/transactions?user_id=" + userID.String() + "&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListTransactionsParams{
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
					Limit:  10,
					Offset: 0,
				}

				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Eq(listArg)).
					Return([]db.Transaction{}, errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
		{
			name: "DefaultPagination",
			url:  "/transactions?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListTransactionsParams{
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
					Limit:  10,
					Offset: 0,
				}

				transactions := []db.Transaction{
					{
						ID:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
						UserID:     pgtype.UUID{Bytes: userID, Valid: true},
						AccountID:  pgtype.UUID{Bytes: accountID, Valid: true},
						CategoryID: pgtype.UUID{Bytes: categoryID, Valid: true},
						Amount: pgtype.Numeric{
							Int:   big.NewInt(1005000),
							Exp:   -4,
							Valid: true,
						},
						Description: "Default Pagination Transaction",
						TransactionDate: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second),
							Valid: true,
						},
						CreatedAt: pgtype.Timestamptz{
							Time:  time.Now().Truncate(time.Second),
							Valid: true,
						},
					},
				}

				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Eq(listArg)).
					Return(transactions, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response []map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response, 1)
			},
		},
		{
			name: "LimitExceeded",
			url:  "/transactions?user_id=" + userID.String() + "&limit=150",
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListTransactionsParams{
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
					Limit:  100,
					Offset: 0,
				}

				transactions := make([]db.Transaction, 0)

				store.EXPECT().
					ListTransactions(gomock.Any(), gomock.Eq(listArg)).
					Return(transactions, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response []map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response, 0)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.setupMock(store)

			server := &Server{
				Store:  store,
				Router: gin.Default(),
			}
			server.setupRoutes()

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestUpdateTransactionAPI(t *testing.T) {

	transactionDate := time.Date(2025, 9, 18, 10, 13, 23, 0, time.UTC)

	testCases := []struct {
		name          string
		body          string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: fmt.Sprintf(`{
				"id": "%s",
				"user_id": "%s",
				"account_id": "%s",
				"category_id": "%s",
				"amount": 200.75,
				"description": "Updated Transaction",
				"transaction_date": "%s"
			}`, transactionID.String(), userID.String(), accountID.String(), categoryID.String(), transactionDate.Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.UpdateTransactionParams{
					ID:         pgtype.UUID{Bytes: transactionID, Valid: true},
					UserID:     pgtype.UUID{Bytes: userID, Valid: true},
					AccountID:  pgtype.UUID{Bytes: accountID, Valid: true},
					CategoryID: pgtype.UUID{Bytes: categoryID, Valid: true},
					Amount: pgtype.Numeric{
						Int:   big.NewInt(2007500),
						Exp:   -4,
						Valid: true,
					},
					Description:     "Updated Transaction",
					TransactionDate: pgtype.Timestamptz{Time: transactionDate, Valid: true},
				}
				updatedTransaction := db.Transaction{
					ID:         pgtype.UUID{Bytes: transactionID, Valid: true},
					UserID:     pgtype.UUID{Bytes: userID, Valid: true},
					AccountID:  pgtype.UUID{Bytes: accountID, Valid: true},
					CategoryID: pgtype.UUID{Bytes: categoryID, Valid: true},
					Amount: pgtype.Numeric{
						Int:   big.NewInt(2007500),
						Exp:   -4,
						Valid: true,
					},
					Description:     "Updated Transaction",
					TransactionDate: pgtype.Timestamptz{Time: transactionDate, Valid: true},
					CreatedAt:       pgtype.Timestamptz{Time: time.Now().Truncate(time.Second), Valid: true},
				}
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Eq(arg)).
					Return(updatedTransaction, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseTransaction db.Transaction
				err := json.Unmarshal(recorder.Body.Bytes(), &responseTransaction)
				require.NoError(t, err)
				require.Equal(t, "Updated Transaction", responseTransaction.Description)
			},
		},
		{
			name: "InvalidTransactionID",
			body: fmt.Sprintf(`{
				"id": "invalid-uuid",
				"user_id": "%s",
				"account_id": "%s",
				"category_id": "%s",
				"amount": 200.75,
				"description": "Updated Transaction",
				"transaction_date": "%s"
			}`, userID.String(), accountID.String(), categoryID.String(), time.Now().Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "MissingRequiredFields",
			body: `{}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "required")
			},
		},
		{
			name: "StoreError",
			body: fmt.Sprintf(`{
				"id": "%s",
				"user_id": "%s",
				"account_id": "%s",
				"category_id": "%s",
				"amount": 200.75,
				"description": "Updated Transaction",
				"transaction_date": "%s"
			}`, transactionID.String(), userID.String(), accountID.String(), categoryID.String(), time.Now().Format(time.RFC3339)),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTransaction(gomock.Any(), gomock.Any()).
					Return(db.Transaction{}, errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.setupMock(store)

			server := &Server{
				Store:  store,
				Router: gin.Default(),
			}
			server.setupRoutes()

			req := httptest.NewRequest(http.MethodPut, "/transactions", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestDeleteTransactionAPI(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  fmt.Sprintf("/transactions/%s?user_id=%s", transactionID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.DeleteTransactionParams{
					ID:     pgtype.UUID{Bytes: transactionID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Eq(arg)).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				message, exists := response["message"]
				require.True(t, exists)
				require.Equal(t, "transaction deleted successfully", message)
			},
		},
		{
			name: "StoreError",
			url:  fmt.Sprintf("/transactions/%s?user_id=%s", transactionID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.DeleteTransactionParams{
					ID:     pgtype.UUID{Bytes: transactionID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Eq(arg)).
					Return(errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
		{
			name: "InvalidTransactionID",
			url:  fmt.Sprintf("/transactions/invalid-uuid?user_id=%s", userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "InvalidUserID",
			url:  fmt.Sprintf("/transactions/%s?user_id=invalid-uuid", transactionID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteTransaction(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.setupMock(store)

			server := &Server{
				Store:  store,
				Router: gin.Default(),
			}
			server.setupRoutes()

			req := httptest.NewRequest(http.MethodDelete, tc.url, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}
