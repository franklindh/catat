package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	mockdb "github.com/franklindh/catat/db/mock"
	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

var userID = uuid.MustParse("b25d7919-6071-422a-85f9-c88afb3f63ad")

type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x any) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPasswordHash(e.password, arg.Password)
	if err != nil {
		return false
	}

	e.arg.Password = arg.Password
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		body          gin.H
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"email":    user.Email,
				"name":     user.Name,
				"password": password,
			},
			setupMock: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Email:    user.Email,
					Name:     user.Name,
					Password: password,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
					Times(1).
					Return(db.CreateUserRow{
						ID:        pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4}, Valid: true},
						Email:     user.Email,
						Name:      user.Name,
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"email":    "invalid-email",
				"name":     user.Name,
				"password": password,
			},
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "MissingName",
			body: gin.H{
				"email":    user.Email,
				"password": password,
			},
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ShortPassword",
			body: gin.H{
				"email":    user.Email,
				"name":     user.Name,
				"password": "123",
			},
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "EmptyBody",
			body: gin.H{},
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "StoreError",
			body: gin.H{
				"email":    user.Email,
				"name":     user.Name,
				"password": password,
			},
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.CreateUserRow{}, sql.ErrConnDone)
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

			jsonData, err := json.Marshal(tc.body)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestGetUserByIDAPI(t *testing.T) {
	testCases := []struct {
		name          string
		userID        string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				userUUID := pgtype.UUID{Bytes: userID, Valid: true}
				store.EXPECT().
					GetUserByID(gomock.Any(), gomock.Eq(userUUID)).
					Return(db.GetUserByIDRow{
						ID:        pgtype.UUID{Bytes: userID, Valid: true},
						Email:     "test@example.com",
						Name:      "Test User",
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:   "InvalidUserID",
			userID: "invalid-uuid",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByID(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "UserNotFound",
			userID: userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				userUUID := pgtype.UUID{Bytes: userID, Valid: true}
				store.EXPECT().
					GetUserByID(gomock.Any(), gomock.Eq(userUUID)).
					Return(db.GetUserByIDRow{}, sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "StoreError",
			userID: userID.String(),
			setupMock: func(store *mockdb.MockStore) {
				userUUID := pgtype.UUID{Bytes: userID, Valid: true}
				store.EXPECT().
					GetUserByID(gomock.Any(), gomock.Eq(userUUID)).
					Return(db.GetUserByIDRow{}, sql.ErrConnDone)
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

			req := httptest.NewRequest(http.MethodGet, "/users/"+tc.userID, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestGetUserByEmailAPI(t *testing.T) {
	testCases := []struct {
		name          string
		email         string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			email: "test@example.com",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq("test@example.com")).
					Return(db.GetUserByEmailRow{
						ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
						Email:     "test@example.com",
						Name:      "Test User",
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "InvalidEmail",
			email: "invalid-email",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "UserNotFound",
			email: "notfound@example.com",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq("notfound@example.com")).
					Return(db.GetUserByEmailRow{}, sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:  "StoreError",
			email: "error@example.com",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq("error@example.com")).
					Return(db.GetUserByEmailRow{}, sql.ErrConnDone)
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

			req := httptest.NewRequest(http.MethodGet, "/users?email="+tc.email, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestListUsersAPI(t *testing.T) {
	n := 5
	users := make([]db.ListUsersRow, n)
	for i := 0; i < n; i++ {
		users[i] = db.ListUsersRow{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			Email:     fmt.Sprintf("user%d@example.com", i),
			Name:      fmt.Sprintf("User %d", i),
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
	}

	testCases := []struct {
		name          string
		query         string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			query: "page=1&limit=5",
			setupMock: func(store *mockdb.MockStore) {
				arg := db.ListUsersParams{
					Limit:  5,
					Offset: 0,
				}
				store.EXPECT().
					ListUsers(gomock.Any(), gomock.Eq(arg)).
					Return(users, nil)
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
			name:  "DefaultPagination",
			query: "",
			setupMock: func(store *mockdb.MockStore) {
				arg := db.ListUsersParams{
					Limit:  10,
					Offset: 0,
				}
				store.EXPECT().
					ListUsers(gomock.Any(), gomock.Eq(arg)).
					Return(users, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "LimitExceeded",
			query: "limit=150",
			setupMock: func(store *mockdb.MockStore) {
				arg := db.ListUsersParams{
					Limit:  100,
					Offset: 0,
				}
				store.EXPECT().
					ListUsers(gomock.Any(), gomock.Eq(arg)).
					Return(users, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "StoreError",
			query: "page=1&limit=5",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListUsers(gomock.Any(), gomock.Any()).
					Return([]db.ListUsersRow{}, sql.ErrConnDone)
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

			url := "/users/list"
			if tc.query != "" {
				url = "/users/list?" + tc.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestUpdateUserAPI(t *testing.T) {
	email := util.GetRandomEmail()
	name := util.GetRandomName()

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
				"email": "%s",
				"name": "%s"
			}`, userID.String(), email.String, name.String),
			setupMock: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					ID:    pgtype.UUID{Bytes: userID, Valid: true},
					Email: pgtype.Text{String: email.String, Valid: true},
					Name:  pgtype.Text{String: name.String, Valid: true},
				}
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Eq(arg)).
					Return(db.UpdateUserRow{
						ID:        pgtype.UUID{Bytes: userID, Valid: true},
						Email:     email.String,
						Name:      name.String,
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "InvalidUserID",
			body: fmt.Sprintf(`{
				"id": "invalid-uuid",
				"email": "%s",
				"name": "%s"
			}`, email.String, name.String),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: fmt.Sprintf(`{
				"id": "%s",
				"email": "invalid-email",
				"name": "%s"
			}`, userID.String(), name.String),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "MissingName",
			body: fmt.Sprintf(`{
				"id": "%s",
				"email": "%s",
			}`, userID.String(), email.String),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
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
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "StoreError",
			body: fmt.Sprintf(`{
				"id": "%s",
				"email": "%s",
				"name": "%s"
			}`, userID.String(), email.String, name.String),
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Return(db.UpdateUserRow{}, sql.ErrConnDone)
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

			req := httptest.NewRequest(http.MethodPatch, "/users", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func TestDeleteUserAPI(t *testing.T) {
	testCases := []struct {
		name          string
		userID        string
		setupMock     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: userID.String(),
			setupMock: func(store *mockdb.MockStore) {

				userUUID := pgtype.UUID{Bytes: userID, Valid: true}
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Eq(userUUID)).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				message, exists := response["message"]
				require.True(t, exists)
				require.Equal(t, "user deleted successfully", message)
			},
		},
		{
			name:   "InvalidUserID",
			userID: "invalid-uuid",
			setupMock: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "StoreError",
			userID: userID.String(),
			setupMock: func(store *mockdb.MockStore) {

				userUUID := pgtype.UUID{Bytes: userID, Valid: true}
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Eq(userUUID)).
					Return(sql.ErrConnDone)
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

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tc.userID, nil)
			w := httptest.NewRecorder()

			server.Router.ServeHTTP(w, req)
			tc.checkResponse(w)
		})
	}
}

func randomUser(t *testing.T) (user db.User, password string) {
	password = util.GetRandomName().String
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		Email:    util.GetRandomEmail().String,
		Name:     util.GetRandomName().String,
		Password: hashedPassword,
	}
	return
}
