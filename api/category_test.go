package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

var categoryID = uuid.MustParse("96d7d741-5104-4db3-a8bd-29719d483226")

func TestCreateCategoryAPI(t *testing.T) {
	testCases := []struct {
		name          string
		body          string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: `{
				"user_id": "b25d7919-6071-422a-85f9-c88afb3f63ad",
				"name": "Test Category",
				"type": "income"
			}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.CreateCategoryParams) (db.Category, error) {
						require.Equal(t, "Test Category", arg.Name)
						require.Equal(t, "income", arg.Type)
						require.True(t, arg.UserID.Valid)
						return db.Category{
							ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
							UserID:    arg.UserID,
							Name:      "Test Category",
							Type:      "income",
							CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					}).Times(1)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)

				var responseCategory db.Category
				err := json.Unmarshal(recorder.Body.Bytes(), &responseCategory)
				require.NoError(t, err)
				require.Equal(t, "Test Category", responseCategory.Name)
				require.Equal(t, "income", responseCategory.Type)
			},
		},
		{
			name: "InvalidUserID",
			body: `{
				"user_id": "invalid-uuid",
				"name": "Test Category",
				"type": "income"
			}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "MissingName",
			body: `{
				"user_id": "b25d7919-6071-422a-85f9-c88afb3f63ad",
				"type": "income"
			}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidType",
			body: `{
				"user_id": "b25d7919-6071-422a-85f9-c88afb3f63ad",
				"name": "Test Category",
				"type": "invalid-type"
			}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
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
					CreateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "DatabaseError",
			body: `{
				"user_id": "b25d7919-6071-422a-85f9-c88afb3f63ad",
				"name": "Test Category",
				"type": "income"
			}`,
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					Return(db.Category{}, sql.ErrConnDone).
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

			req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestGetCategoryAPI(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  "/categories/" + categoryID.String() + "?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetCategoryParams{
					ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetCategory(gomock.Any(), gomock.Eq(arg)).
					Return(db.Category{
						ID:        pgtype.UUID{Bytes: categoryID, Valid: true},
						UserID:    pgtype.UUID{Bytes: userID, Valid: true},
						Name:      "Test Category",
						Type:      "income",
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseCategory db.Category
				err := json.Unmarshal(recorder.Body.Bytes(), &responseCategory)
				require.NoError(t, err)
				require.Equal(t, "Test Category", responseCategory.Name)
				require.Equal(t, "income", responseCategory.Type)
			},
		},
		{
			name: "InvalidURI",
			url:  "/categories/invalid-uuid?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "InvalidQuery",
			url:  "/categories/" + categoryID.String() + "?user_id=invalid-uuid",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "StoreError",
			url:  "/categories/" + categoryID.String() + "?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetCategoryParams{
					ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetCategory(gomock.Any(), gomock.Eq(arg)).
					Return(db.Category{}, errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
		{
			name: "NotFound",
			url:  "/categories/" + categoryID.String() + "?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.GetCategoryParams{
					ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					GetCategory(gomock.Any(), gomock.Eq(arg)).
					Return(db.Category{}, sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
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

func TestListCategoriesAPI(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  "/categories?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
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

				store.EXPECT().
					ListCategories(gomock.Any(), pgtype.UUID{Bytes: userID, Valid: true}).
					Return(categories, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseCategories []db.Category
				err := json.Unmarshal(recorder.Body.Bytes(), &responseCategories)
				require.NoError(t, err)
				require.Len(t, responseCategories, 2)
				require.Equal(t, "Income Category", responseCategories[0].Name)
				require.Equal(t, "Expense Category", responseCategories[1].Name)
			},
		},
		{
			name: "InvalidUserID",
			url:  "/categories?user_id=invalid-uuid",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListCategories(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "StoreError",
			url:  "/categories?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListCategories(gomock.Any(), pgtype.UUID{Bytes: userID, Valid: true}).
					Return([]db.Category{}, errors.New("database error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "database error")
			},
		},
		{
			name: "EmptyResult",
			url:  "/categories?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListCategories(gomock.Any(), pgtype.UUID{Bytes: userID, Valid: true}).
					Return([]db.Category{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseCategories []db.Category
				err := json.Unmarshal(recorder.Body.Bytes(), &responseCategories)
				require.NoError(t, err)
				require.Len(t, responseCategories, 0)
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

func TestUpdateCategoryAPI(t *testing.T) {
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
				"name": "Updated Category Name",
				"type": "expense",
				"user_id": "%s"
			}`, categoryID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.UpdateCategoryParams{
					ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
					Name:   "Updated Category Name",
					Type:   "expense",
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				updatedCategory := db.Category{
					ID:        pgtype.UUID{Bytes: categoryID, Valid: true},
					UserID:    pgtype.UUID{Bytes: userID, Valid: true},
					Name:      "Updated Category Name",
					Type:      "expense",
					UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
				}
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Eq(arg)).
					Return(updatedCategory, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var responseCategory db.Category
				err := json.Unmarshal(recorder.Body.Bytes(), &responseCategory)
				require.NoError(t, err)
				require.Equal(t, "Updated Category Name", responseCategory.Name)
				require.Equal(t, "expense", responseCategory.Type)
			},
		},
		{
			name: "InvalidCategoryID",
			body: fmt.Sprintf(`{
				"id": "invalid-uuid",
				"name": "Updated Name",
				"type": "income",
				"user_id": "%s"
			}`, userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "InvalidUserID",
			body: fmt.Sprintf(`{
				"id": "%s",
				"name": "Updated Name",
				"type": "income",
				"user_id": "invalid-uuid"
			}`, categoryID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
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
				"type": "income",
				"user_id": "%s"
			}`, categoryID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
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
				"type": "invalid",
				"user_id": "%s"
			}`, categoryID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
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
					UpdateCategory(gomock.Any(), gomock.Any()).
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
				"name": "Updated Category Name",
				"type": "expense",
				"user_id": "%s"
			}`, categoryID.String(), userID.String()),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Return(db.Category{}, errors.New("database error")).
					Times(1)
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

			req := httptest.NewRequest(http.MethodPut, "/categories", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestDeleteCategoryAPI(t *testing.T) {
	categoryID := uuid.New()
	userID := uuid.New()

	testCases := []struct {
		name          string
		url           string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  "/categories/" + categoryID.String() + "?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.DeleteCategoryParams{
					ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Eq(arg)).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				message, exists := response["message"]
				require.True(t, exists)
				require.Equal(t, "category deleted successfully", message)
			},
		},
		{
			name: "InvalidURI",
			url:  "/categories/invalid-uuid?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "InvalidQuery",
			url:  "/categories/" + categoryID.String() + "?user_id=invalid-uuid",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), "uuid")
			},
		},
		{
			name: "StoreError",
			url:  "/categories/" + categoryID.String() + "?user_id=" + userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.DeleteCategoryParams{
					ID:     pgtype.UUID{Bytes: categoryID, Valid: true},
					UserID: pgtype.UUID{Bytes: userID, Valid: true},
				}
				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Eq(arg)).
					Return(errors.New("database error"))
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

			req := httptest.NewRequest(http.MethodDelete, tc.url, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}
