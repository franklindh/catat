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

var accountID = uuid.MustParse("d2b072e4-e145-4398-bcf5-ed79b15c95b8")
var userID = uuid.MustParse("b25d7919-6071-422a-85f9-c88afb3f63ad")

func TestCreateAccountAPI(t *testing.T) {
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
				"name": "Test Account",
				"type": "depository"
		}`, userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.CreateAccountParams) (db.Account, error) {
						require.Equal(t, "Test Account", arg.Name)
						require.Equal(t, "depository", arg.Type)
						require.True(t, arg.UserID.Valid)
						return db.Account{
							ID:        pgtype.UUID{Bytes: [16]byte{}, Valid: true},
							UserID:    arg.UserID,
							Name:      "Test Account",
							Type:      "depository",
							Balance:   pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true},
							CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "InvalidUserID",
			body: `{
				"user_id": "invalid-uuid",
				"name": "Test Account",
				"type": "depository"
			}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "MissingName",
			body: fmt.Sprintf(`{
				"user_id": "%s",
				"type": "depository"
		}`, userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidType",
			body: fmt.Sprintf(`{
				"user_id": "%s",
				"name": "Test Account",
				"type": "icikiwir"
		}`, userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "EmptyBody",
			body: `{}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
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
					"name": "Test Account",
					"type": "depository"
			}`, userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Return(db.Account{}, sql.ErrConnDone).
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

			req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)

			tc.checkResponse(w)
		})
	}
}

func TestGetAccountAPI(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		accountID     string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  "/accounts/" + accountID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetAccountParams{
					ID:     pgtype.UUID{Bytes: accountID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(arg)).
					Return(db.Account{
						ID:        pgtype.UUID{Bytes: accountID, Valid: true},
						UserID:    pgtype.UUID{Bytes: userID, Valid: true},
						Name:      "Test Account",
						Type:      "depository",
						Balance:   pgtype.Numeric{Int: big.NewInt(1000000), Exp: -4, Valid: true},
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseAccount db.Account
				err := json.Unmarshal(recorder.Body.Bytes(), &responseAccount)
				require.NoError(t, err)
				require.Equal(t, "Test Account", responseAccount.Name)
				require.Equal(t, "depository", responseAccount.Type)
			},
		},
		{
			name: "NotFound",
			url:  "/accounts/" + accountID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetAccountParams{
					ID:     pgtype.UUID{Bytes: accountID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(arg)).
					Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "StoreError",
			url:  "/accounts/" + accountID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetAccountParams{
					ID:     pgtype.UUID{Bytes: accountID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(arg)).
					Return(db.Account{}, errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
		{
			name: "InvalidUUID",
			url:  "/accounts/invalid-uuid",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		// {
		// 	name: "InvalidUserID",
		// 	url:  "/accounts/" + accountID.String(),
		// 	setupMock: func(store *mockdb.MockStore) {
		// 		store.EXPECT().
		// 			GetAccount(gomock.Any(), gomock.Any()).
		// 			Times(0)
		// 	},
		// 	checkResponse: func(recorder *httptest.ResponseRecorder) {
		// 		require.Equal(t, http.StatusBadRequest, recorder.Code)
		// 		require.Contains(t, recorder.Body.String(), "uuid")
		// 	},
		// },
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

func TestListAccountsAPI(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  "/accounts?user_id=" + userID.String() + "&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
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

				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(listArg)).
					Return(accounts, nil)

				store.EXPECT().
					CountAccountsByUser(gomock.Any(), gomock.Eq(countArg)).
					Return(int64(2), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				_, exists := response["data"]
				require.True(t, exists)

				_, exists = response["pagination"]
				require.True(t, exists)
			},
		},
		{
			name: "StoreError",
			url:  "/accounts?user_id=" + userID.String() + "&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListAccountsParams{
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
					Limit:  10,
					Offset: 0,
				}

				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(listArg)).
					Return([]db.Account{}, errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
		{
			name: "CountError",
			url:  "/accounts?user_id=" + userID.String() + "&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
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

				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(listArg)).
					Return(accounts, nil)

				store.EXPECT().
					CountAccountsByUser(gomock.Any(), gomock.Eq(countArg)).
					Return(int64(0), errors.New("count error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				data, exists := response["data"]
				require.True(t, exists)
				require.Len(t, data, 1)
			},
		},
		{
			name: "InvalidUserID",
			url:  "/accounts?user_id=invalid-uuid&page=1&limit=10",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					CountAccountsByUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "DefaultPagination",
			url:  "/accounts?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
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

				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(listArg)).
					Return(accounts, nil)

				store.EXPECT().
					CountAccountsByUser(gomock.Any(), gomock.Eq(countArg)).
					Return(int64(1), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				pagination, exists := response["pagination"].(map[string]interface{})
				require.True(t, exists)
				require.Equal(t, float64(1), pagination["page"])
				require.Equal(t, float64(10), pagination["limit"])
				require.Equal(t, float64(1), pagination["totalPages"])
			},
		},
		{
			name: "LimitExceeded",
			url:  "/accounts?user_id=" + userID.String() + "&limit=150",
			setupMock: func(store *mockdb.MockStore) {
				listArg := db.ListAccountsParams{
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
					Limit:  100,
					Offset: 0,
				}
				countArg := pgtype.UUID{Bytes: userID, Valid: true}

				accounts := make([]db.Account, 0)

				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(listArg)).
					Return(accounts, nil)

				store.EXPECT().
					CountAccountsByUser(gomock.Any(), gomock.Eq(countArg)).
					Return(int64(0), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
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

func TestUpdateAccountAPI(t *testing.T) {
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
				"name": "Updated Account Name",
				"type": "credit"
			}`, accountID.String()),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.UpdateAccountParams{
					ID:     pgtype.UUID{Bytes: accountID, Valid: true},
					Name:   "Updated Account Name",
					Type:   "credit",
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				updatedAccount := db.Account{
					ID:        pgtype.UUID{Bytes: accountID, Valid: true},
					UserID:    pgtype.UUID{Bytes: userID, Valid: true},
					Name:      "Updated Account Name",
					Type:      "credit",
					Balance:   pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true},
					CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
				}
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Eq(arg)).
					Return(updatedAccount, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseAccount db.Account
				err := json.Unmarshal(recorder.Body.Bytes(), &responseAccount)
				require.NoError(t, err)
				require.Equal(t, "Updated Account Name", responseAccount.Name)
				require.Equal(t, "credit", responseAccount.Type)
			},
		},
		{
			name: "InvalidAccountID",
			body: `{
				"id": "invalid-uuid",
				"name": "Updated Name",
				"type": "depository"
			}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "MissingName",
			body: fmt.Sprintf(`{
				"id": "%s",
				"type": "depository"
			}`, accountID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "required")
			},
		},
		{
			name: "InvalidType",
			body: fmt.Sprintf(`{
				"id": "%s",
				"name": "Updated Name",
				"type": "invalid"
			}`, accountID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "oneof")
			},
		},
		{
			name: "EmptyBody",
			body: `{}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Any()).
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
				"name": "Updated Account Name",
				"type": "credit"
			}`, accountID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Any()).
					Return(db.Account{}, errors.New("database error"))
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

			req := httptest.NewRequest(http.MethodPut, "/accounts", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestDeleteAccountAPI(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  "/accounts/" + accountID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.DeleteAccountParams{
					ID:     pgtype.UUID{Bytes: accountID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(arg)).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				message, exists := response["message"]
				require.True(t, exists)
				require.Equal(t, "account deleted successfully", message)
			},
		},
		{
			name: "StoreError",
			url:  "/accounts/" + accountID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.DeleteAccountParams{
					ID:     pgtype.UUID{Bytes: accountID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(arg)).
					Return(errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
		{
			name: "InvalidUUID",
			url:  "/accounts/invalid-uuid",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Any()).
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
