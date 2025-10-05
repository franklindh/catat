package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateCategoryAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)

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
				"name":     category.Name,
				"icon_url": category.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				expectedArg := db.CreateCategoryParams{
					UserID:  user.ID,
					Name:    category.Name,
					IconUrl: category.IconUrl,
				}
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.CreateCategoryParams) (db.Category, error) {
						require.Equal(t, expectedArg.UserID, arg.UserID)
						require.Equal(t, expectedArg.Name, arg.Name)
						require.Equal(t, expectedArg.IconUrl, arg.IconUrl)
						return category, nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchCategory(t, recorder.Body, category)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"name":     category.Name,
				"icon_url": category.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"name":     category.Name,
				"icon_url": category.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

				userIDForAuth := util.PgxUUIDToGoogleUUID(user.ID)
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, userIDForAuth, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Category{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "MissingName",
			body: gin.H{
				"icon_url": category.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

				userIDForAuth := util.PgxUUIDToGoogleUUID(user.ID)
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, userIDForAuth, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
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

			url := "/categories"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetCategoriesAPI(t *testing.T) {
	n := 5
	user := randomUser()
	categories := make([]db.Category, n)
	for i := 0; i < n; i++ {
		categories[i] = randomCategory(user.ID)
	}

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), user.ID).
					Times(1).
					Return(categories, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCategories(t, recorder.Body, categories)
			},
		},
		{
			name: "NoAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "NotFound",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), user.ID).
					Times(1).
					Return([]db.Category{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCategories(t, recorder.Body, []db.Category{})
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), user.ID).
					Times(1).
					Return([]db.Category{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
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
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := "/categories"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetCategoryByIDAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)

	testCases := []struct {
		name          string
		categoryID    string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "OK",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCategory(t, recorder.Body, category)
			},
		},
		{
			name:       "NoAuthorization",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:       "InvalidID",
			categoryID: "invalid-uuid",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:       "NotFound",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.Category{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "Forbidden",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				otherUser := randomUser()
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(otherUser.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:       "InternalError",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.Category{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
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
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/categories/%s", tc.categoryID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestUpdateCategoryAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)
	updatedCategory := category
	updatedCategory.Name = util.RandomString(6)
	updatedCategory.IconUrl = pgtype.Text{String: util.RandomString(10), Valid: true}

	testCases := []struct {
		name          string
		categoryID    string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "OK",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)

				expectedArg := db.UpdateCategoryParams{
					ID:      category.ID,
					Name:    updatedCategory.Name,
					IconUrl: updatedCategory.IconUrl,
					UserID:  user.ID,
				}
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.UpdateCategoryParams) (db.Category, error) {
						require.Equal(t, expectedArg.ID, arg.ID)
						require.Equal(t, expectedArg.Name, arg.Name)
						require.Equal(t, expectedArg.IconUrl, arg.IconUrl)
						require.Equal(t, expectedArg.UserID, arg.UserID)
						return updatedCategory, nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCategory(t, recorder.Body, updatedCategory)
			},
		},
		{
			name:       "NoAuthorization",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:       "InvalidID",
			categoryID: "invalid-uuid",
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:       "MissingName",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			body: gin.H{
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

				userIDForAuth := util.PgxUUIDToGoogleUUID(user.ID)
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, userIDForAuth, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:       "NotFound",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.Category{}, sql.ErrNoRows)

				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "Forbidden",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				otherUser := randomUser()
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(otherUser.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)

				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:       "InternalError",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)

				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Category{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
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
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/categories/%s", tc.categoryID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteCategoryAPI(t *testing.T) {
	user := randomUser()
	category := randomCategory(user.ID)

	testCases := []struct {
		name          string
		categoryID    string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "OK",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)

				expectedArg := db.DeleteCategoryParams{
					ID:     category.ID,
					UserID: user.ID,
				}
				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.DeleteCategoryParams) error {
						require.Equal(t, expectedArg.ID, arg.ID)
						require.Equal(t, expectedArg.UserID, arg.UserID)
						return nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]string
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "category deleted successfully", response["message"])
			},
		},
		{
			name:       "NoAuthorization",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:       "InvalidID",
			categoryID: "invalid-uuid",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:       "NotFound",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.Category{}, sql.ErrNoRows)

				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "Forbidden",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				otherUser := randomUser()
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(otherUser.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)

				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:       "InternalError",
			categoryID: util.PgxUUIDToGoogleUUID(category.ID).String(),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, util.PgxUUIDToGoogleUUID(user.ID), time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(category, nil)

				store.EXPECT().
					DeleteCategory(gomock.Any(), gomock.Any()).
					Times(1).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
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
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/categories/%s", tc.categoryID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func requireBodyMatchCategory(t *testing.T, body *bytes.Buffer, category db.Category) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotCategory categoryResponse
	err = json.Unmarshal(data, &gotCategory)
	require.NoError(t, err)

	expectedID := util.PgxUUIDToGoogleUUID(category.ID)
	expectedUserID := util.PgxUUIDToGoogleUUID(category.UserID)

	require.Equal(t, expectedID, gotCategory.ID)
	require.Equal(t, expectedUserID.String(), gotCategory.UserID)
	require.Equal(t, category.Name, gotCategory.Name)
	require.Equal(t, category.IconUrl.String, gotCategory.IconURL)

}

func requireBodyMatchCategories(t *testing.T, body *bytes.Buffer, categories []db.Category) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotCategories []categoryResponse
	err = json.Unmarshal(data, &gotCategories)
	require.NoError(t, err)

	require.Equal(t, len(categories), len(gotCategories))

	for i := 0; i < len(categories); i++ {

		expectedID := util.PgxUUIDToGoogleUUID(categories[i].ID)
		expectedUserID := util.PgxUUIDToGoogleUUID(categories[i].UserID)

		require.Equal(t, expectedID, gotCategories[i].ID)
		require.Equal(t, expectedUserID.String(), gotCategories[i].UserID)
		require.Equal(t, categories[i].Name, gotCategories[i].Name)
		require.Equal(t, categories[i].IconUrl.String, gotCategories[i].IconURL)

	}
}

func randomCategory(userID pgtype.UUID) db.Category {
	return db.Category{
		ID:        util.GoogleUUIDToPgxUUID(uuid.New()),
		UserID:    userID,
		Name:      util.RandomString(6),
		IconUrl:   pgtype.Text{String: util.RandomString(10), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now()},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now()},
	}
}
