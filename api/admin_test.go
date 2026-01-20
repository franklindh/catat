package api

import (
	"bytes"
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	admin := randomAdminUser()
	users := randomListUsersRows(5)

	testCases := []struct {
		name          string
		query         string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			query: "?page=1&limit=10",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListUsers(gomock.Any(), db.ListUsersParams{
						Limit:  10,
						Offset: 0,
					}).
					Times(1).
					Return(users, nil)
				store.EXPECT().
					CountUsers(gomock.Any()).
					Times(1).
					Return(int64(len(users)), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchListUsers(t, recorder.Body, users, int64(len(users)))
			},
		},
		{
			name:  "DefaultPagination",
			query: "",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListUsers(gomock.Any(), db.ListUsersParams{
						Limit:  20,
						Offset: 0,
					}).
					Times(1).
					Return(users, nil)
				store.EXPECT().
					CountUsers(gomock.Any()).
					Times(1).
					Return(int64(len(users)), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "WithSearch",
			query: "?search=test",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					SearchUsers(gomock.Any(), gomock.Any()).
					Times(1).
					Return(usersToSearchUsersRows(users), nil)
				store.EXPECT().
					CountUsers(gomock.Any()).
					Times(1).
					Return(int64(len(users)), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "Unauthorized",
			query: "",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsers(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().CountUsers(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:  "ForbiddenNonAdmin",
			query: "",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleUser, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsers(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().CountUsers(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:  "InternalErrorListUsers",
			query: "",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListUsers(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:  "InternalErrorCountUsers",
			query: "",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListUsers(gomock.Any(), gomock.Any()).
					Times(1).
					Return(users, nil)
				store.EXPECT().
					CountUsers(gomock.Any()).
					Times(1).
					Return(int64(0), sql.ErrConnDone)
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

			url := "/admin/users" + tc.query
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetUserByID(t *testing.T) {
	admin := randomAdminUser()
	user := randomUser()

	testCases := []struct {
		name          string
		userID        string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(userToGetUserRow(user), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAdminUser(t, recorder.Body, user)
			},
		},
		{
			name:   "InvalidUserID",
			userID: "invalid",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "NotFound",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(db.GetUserRow{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "Unauthorized",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:   "ForbiddenNonAdmin",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleUser, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(db.GetUserRow{}, sql.ErrConnDone)
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

			url := fmt.Sprintf("/admin/users/%s", tc.userID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestUpdateUserRole(t *testing.T) {
	admin := randomAdminUser()
	user := randomUser()

	testCases := []struct {
		name          string
		userID        string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK_UpdateToAdmin",
			userID: fmt.Sprintf("%d", user.ID),
			body: gin.H{
				"role": RoleAdmin,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				updatedUser := user
				updatedUser.Role = RoleAdmin
				store.EXPECT().
					UpdateUserRole(gomock.Any(), db.UpdateUserRoleParams{
						ID:   user.ID,
						Role: RoleAdmin,
					}).
					Times(1).
					Return(userToUpdateUserRoleRow(updatedUser), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:   "OK_UpdateToUser",
			userID: fmt.Sprintf("%d", user.ID),
			body: gin.H{
				"role": RoleUser,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				updatedUser := user
				updatedUser.Role = RoleUser
				store.EXPECT().
					UpdateUserRole(gomock.Any(), db.UpdateUserRoleParams{
						ID:   user.ID,
						Role: RoleUser,
					}).
					Times(1).
					Return(userToUpdateUserRoleRow(updatedUser), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:   "InvalidUserID",
			userID: "invalid",
			body: gin.H{
				"role": RoleAdmin,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUserRole(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "InvalidRole",
			userID: fmt.Sprintf("%d", user.ID),
			body: gin.H{
				"role": "invalid",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUserRole(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "MissingRole",
			userID: fmt.Sprintf("%d", user.ID),
			body:   gin.H{},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUserRole(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "NotFound",
			userID: fmt.Sprintf("%d", user.ID),
			body: gin.H{
				"role": RoleAdmin,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUserRole(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.UpdateUserRoleRow{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "Unauthorized",
			userID: fmt.Sprintf("%d", user.ID),
			body: gin.H{
				"role": RoleAdmin,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUserRole(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:   "ForbiddenNonAdmin",
			userID: fmt.Sprintf("%d", user.ID),
			body: gin.H{
				"role": RoleAdmin,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleUser, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUserRole(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			userID: fmt.Sprintf("%d", user.ID),
			body: gin.H{
				"role": RoleAdmin,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUserRole(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.UpdateUserRoleRow{}, sql.ErrConnDone)
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

			url := fmt.Sprintf("/admin/users/%s/role", tc.userID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json")

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteUserByAdmin(t *testing.T) {
	admin := randomAdminUser()
	user := randomUser()

	testCases := []struct {
		name          string
		userID        string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(userToGetUserRow(user), nil)
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:   "InvalidUserID",
			userID: "invalid",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "UserNotFound",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(db.GetUserRow{}, pgx.ErrNoRows)
				store.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "Unauthorized",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:   "ForbiddenNonAdmin",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleUser, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:   "InternalErrorGetUser",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(db.GetUserRow{}, sql.ErrConnDone)
				store.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "InternalErrorDeleteUser",
			userID: fmt.Sprintf("%d", user.ID),
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(userToGetUserRow(user), nil)
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Eq(user.ID)).
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

			url := fmt.Sprintf("/admin/users/%s", tc.userID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetAdminStats(t *testing.T) {
	admin := randomAdminUser()

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
				store.EXPECT().
					CountUsers(gomock.Any()).
					Times(1).
					Return(int64(100), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAdminStats(t, recorder.Body, 100)
			},
		},
		{
			name: "Unauthorized",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CountUsers(gomock.Any()).Times(0)
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
				store.EXPECT().CountUsers(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationHeaderBearerType, admin.ID, RoleAdmin, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CountUsers(gomock.Any()).
					Times(1).
					Return(int64(0), sql.ErrConnDone)
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

			url := "/admin/stats"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func randomAdminUser() db.User {
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

func randomListUsersRows(n int) []db.ListUsersRow {
	users := make([]db.ListUsersRow, n)
	for i := 0; i < n; i++ {
		users[i] = db.ListUsersRow{
			ID:           util.RandomInt(1, 1000000),
			Email:        util.RandomEmail(),
			Name:         pgtype.Text{String: util.RandomName(), Valid: true},
			AvatarUrl:    pgtype.Text{String: "https://example.com/avatar.jpg", Valid: true},
			GoogleAuthID: pgtype.Text{String: util.RandomString(12), Valid: true},
			Role:         RoleUser,
			CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
	}
	return users
}

func usersToSearchUsersRows(users []db.ListUsersRow) []db.SearchUsersRow {
	result := make([]db.SearchUsersRow, len(users))
	for i, u := range users {
		result[i] = db.SearchUsersRow{
			ID:           u.ID,
			Email:        u.Email,
			Name:         u.Name,
			AvatarUrl:    u.AvatarUrl,
			GoogleAuthID: u.GoogleAuthID,
			Role:         u.Role,
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
		}
	}
	return result
}

func userToUpdateUserRoleRow(user db.User) db.UpdateUserRoleRow {
	return db.UpdateUserRoleRow{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		AvatarUrl:    user.AvatarUrl,
		GoogleAuthID: user.GoogleAuthID,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func requireBodyMatchListUsers(t *testing.T, body *bytes.Buffer, users []db.ListUsersRow, totalCount int64) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var response listUsersResponse
	err = json.Unmarshal(data, &response)
	require.NoError(t, err)

	require.Equal(t, len(users), len(response.Users))
	require.Equal(t, totalCount, response.TotalCount)
}

func requireBodyMatchAdminUser(t *testing.T, body *bytes.Buffer, user db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser adminUserResponse
	err = json.Unmarshal(data, &gotUser)
	require.NoError(t, err)

	require.Equal(t, user.ID, gotUser.ID)
	require.Equal(t, user.GoogleAuthID.String, gotUser.GoogleID)
	require.Equal(t, user.Email, gotUser.Email)
	require.Equal(t, user.Name.String, gotUser.Name)
	require.Equal(t, user.AvatarUrl.String, gotUser.AvatarUrl)
	require.Equal(t, user.Role, gotUser.Role)
}

func requireBodyMatchAdminStats(t *testing.T, body *bytes.Buffer, totalUsers int64) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var response adminStatsResponse
	err = json.Unmarshal(data, &response)
	require.NoError(t, err)

	require.Equal(t, totalUsers, response.TotalUsers)
}
