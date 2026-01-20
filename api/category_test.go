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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				expectedArg := db.CreateCategoryParams{
					UserID:  user.ID,
					Name:    category.Name,
					Type:    category.Type,
					IconUrl: category.IconUrl,
				}
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.CreateCategoryParams) (db.CreateCategoryRow, error) {
						require.Equal(t, expectedArg.UserID, arg.UserID)
						require.Equal(t, expectedArg.Name, arg.Name)
						require.Equal(t, expectedArg.Type, arg.Type)
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

				userIDForAuth := user.ID
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, userIDForAuth, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateCategory(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateCategoryRow{}, sql.ErrConnDone)
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

				userIDForAuth := user.ID
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, userIDForAuth, user.Role, time.Minute)
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
	admin := randomAdminUserForCategory()
	categories := make([]db.GetCategoriesByUserRow, n)
	for i := 0; i < n; i++ {
		categories[i] = randomGetCategoriesByUserRow(admin.ID)
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.GetCategoriesByUserParams{
					UserID: admin.ID,
					Type:   "EXPENSE",
				}
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), gomock.Eq(arg)).
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
			name: "ForbiddenNonAdmin",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleUser, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "NotFound",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.GetCategoriesByUserParams{
					UserID: admin.ID,
					Type:   "EXPENSE",
				}
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return([]db.GetCategoriesByUserRow{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCategories(t, recorder.Body, []db.GetCategoriesByUserRow{})
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.GetCategoriesByUserParams{
					UserID: admin.ID,
					Type:   "EXPENSE",
				}
				store.EXPECT().
					GetCategoriesByUser(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return([]db.GetCategoriesByUserRow{}, sql.ErrConnDone)
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

			url := "/admin/categories"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func randomAdminUserForCategory() db.User {
	return db.User{
		ID:           util.RandomInt(1, 1000000),
		Email:        util.RandomEmail(),
		Name:         pgtype.Text{String: util.RandomName(), Valid: true},
		AvatarUrl:    pgtype.Text{String: "https://example.com/avatar.jpg", Valid: true},
		GoogleAuthID: pgtype.Text{String: util.RandomString(12), Valid: true},
		Role:         RoleAdmin,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
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
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCategory(t, recorder.Body, category)
			},
		},
		{
			name:       "NoAuthorization",
			categoryID: fmt.Sprintf("%d", category.ID),
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
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
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.GetCategoryRow{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "Forbidden",
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				otherUser := randomUser()
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, otherUser.ID, otherUser.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:       "InternalError",
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.GetCategoryRow{}, sql.ErrConnDone)
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
			categoryID: fmt.Sprintf("%d", category.ID),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)

				expectedArg := db.UpdateCategoryParams{
					ID:      category.ID,
					Name:    updatedCategory.Name,
					IconUrl: updatedCategory.IconUrl,
					UserID:  user.ID,
				}
				updatedCategoryRow := db.UpdateCategoryRow{
					ID:        category.ID,
					UserID:    user.ID,
					Name:      updatedCategory.Name,
					Type:      category.Type,
					IconUrl:   updatedCategory.IconUrl,
					CreatedAt: category.CreatedAt,
					UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
				}
				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, arg db.UpdateCategoryParams) (db.UpdateCategoryRow, error) {
						require.Equal(t, expectedArg.ID, arg.ID)
						require.Equal(t, expectedArg.Name, arg.Name)
						require.Equal(t, expectedArg.IconUrl, arg.IconUrl)
						require.Equal(t, expectedArg.UserID, arg.UserID)
						return updatedCategoryRow, nil
					}).
					Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				updatedCategoryRow := db.UpdateCategoryRow{
					ID:        category.ID,
					UserID:    user.ID,
					Name:      updatedCategory.Name,
					Type:      category.Type,
					IconUrl:   updatedCategory.IconUrl,
					CreatedAt: category.CreatedAt,
					UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
				}
				requireBodyMatchCategory(t, recorder.Body, updatedCategoryRow)
			},
		},
		{
			name:       "NoAuthorization",
			categoryID: fmt.Sprintf("%d", category.ID),
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
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
			categoryID: fmt.Sprintf("%d", category.ID),
			body: gin.H{
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {

				userIDForAuth := user.ID
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, userIDForAuth, user.Role, time.Minute)
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
			categoryID: fmt.Sprintf("%d", category.ID),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.GetCategoryRow{}, sql.ErrNoRows)

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
			categoryID: fmt.Sprintf("%d", category.ID),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				otherUser := randomUser()
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, otherUser.ID, otherUser.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)

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
			categoryID: fmt.Sprintf("%d", category.ID),
			body: gin.H{
				"name":     updatedCategory.Name,
				"icon_url": updatedCategory.IconUrl.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)

				store.EXPECT().
					UpdateCategory(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.UpdateCategoryRow{}, sql.ErrConnDone)
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
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)

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
			categoryID: fmt.Sprintf("%d", category.ID),
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
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
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
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(db.GetCategoryRow{}, sql.ErrNoRows)

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
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				otherUser := randomUser()
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, otherUser.ID, otherUser.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)

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
			categoryID: fmt.Sprintf("%d", category.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCategory(gomock.Any(), category.ID).
					Times(1).
					Return(categoryToGetCategoryRow(category), nil)

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

func requireBodyMatchCategory(t *testing.T, body *bytes.Buffer, category interface{}) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotCategory categoryResponse
	err = json.Unmarshal(data, &gotCategory)
	require.NoError(t, err)

	switch c := category.(type) {
	case db.CreateCategoryRow:
		require.Equal(t, c.ID, gotCategory.ID)
		require.Equal(t, c.UserID, gotCategory.UserID)
		require.Equal(t, c.Name, gotCategory.Name)
		require.Equal(t, c.Type, gotCategory.Type)
		require.Equal(t, c.IconUrl.String, gotCategory.IconURL)
	case db.GetCategoriesByUserRow:
		require.Equal(t, c.ID, gotCategory.ID)
		require.Equal(t, c.UserID, gotCategory.UserID)
		require.Equal(t, c.Name, gotCategory.Name)
		require.Equal(t, c.Type, gotCategory.Type)
		require.Equal(t, c.IconUrl.String, gotCategory.IconURL)
	case db.GetCategoryRow:
		require.Equal(t, c.ID, gotCategory.ID)
		require.Equal(t, c.UserID, gotCategory.UserID)
		require.Equal(t, c.Name, gotCategory.Name)
		require.Equal(t, c.Type, gotCategory.Type)
		require.Equal(t, c.IconUrl.String, gotCategory.IconURL)
	case db.UpdateCategoryRow:
		require.Equal(t, c.ID, gotCategory.ID)
		require.Equal(t, c.UserID, gotCategory.UserID)
		require.Equal(t, c.Name, gotCategory.Name)
		require.Equal(t, c.Type, gotCategory.Type)
		require.Equal(t, c.IconUrl.String, gotCategory.IconURL)
	}
}

func requireBodyMatchCategories(t *testing.T, body *bytes.Buffer, categories []db.GetCategoriesByUserRow) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotCategories []categoryResponse
	err = json.Unmarshal(data, &gotCategories)
	require.NoError(t, err)

	require.Equal(t, len(categories), len(gotCategories))

	for i := 0; i < len(categories); i++ {
		require.Equal(t, categories[i].ID, gotCategories[i].ID)
		require.Equal(t, categories[i].UserID, gotCategories[i].UserID)
		require.Equal(t, categories[i].Name, gotCategories[i].Name)
		require.Equal(t, categories[i].Type, gotCategories[i].Type)
		require.Equal(t, categories[i].IconUrl.String, gotCategories[i].IconURL)
	}
}

func randomCategory(userID int64) db.CreateCategoryRow {
	return db.CreateCategoryRow{
		ID:        util.RandomInt(1, 1000000),
		UserID:    userID,
		Type:      "EXPENSE",
		Name:      util.RandomString(6),
		IconUrl:   pgtype.Text{String: util.RandomString(10), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now()},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now()},
	}
}

func randomGetCategoriesByUserRow(userID int64) db.GetCategoriesByUserRow {
	return db.GetCategoriesByUserRow{
		ID:        util.RandomInt(1, 1000000),
		UserID:    userID,
		Type:      "EXPENSE",
		Name:      util.RandomString(6),
		IconUrl:   pgtype.Text{String: util.RandomString(10), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now()},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now()},
	}
}

func categoryToGetCategoryRow(category db.CreateCategoryRow) db.GetCategoryRow {
	return db.GetCategoryRow{
		ID:        category.ID,
		UserID:    category.UserID,
		Name:      category.Name,
		Type:      category.Type,
		IconUrl:   category.IconUrl,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}
