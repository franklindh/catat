package db

import (
	"context"
	"testing"
	"time"

	"github.com/franklindh/catat/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	hashedPassword, err := util.HashPassword(util.GetRandomName().String)
	require.NoError(t, err)

	arg := CreateUserParams{
		Email:    util.GetRandomEmail().String,
		Name:     util.GetRandomName().String,
		Password: hashedPassword,
	}

	createUserRow, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createUserRow)

	assert.Equal(t, arg.Email, createUserRow.Email)
	assert.WithinDuration(t, time.Now(), createUserRow.CreatedAt.Time, 5*time.Second)

	user := User{
		ID:        createUserRow.ID,
		Email:     createUserRow.Email,
		Name:      createUserRow.Name,
		CreatedAt: createUserRow.CreatedAt,
	}

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUserByEmail(t *testing.T) {
	createdUser := createRandomUser(t)

	user, err := testQueries.GetUserByEmail(context.Background(), createdUser.Email)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	assert.Equal(t, createdUser.ID, user.ID)
	assert.Equal(t, createdUser.Email, user.Email)

	assert.WithinDuration(t, createdUser.CreatedAt.Time, user.CreatedAt.Time, time.Second)
}

func TestGetUserByID(t *testing.T) {
	createdUser := createRandomUser(t)

	user, err := testQueries.GetUserByID(context.Background(), createdUser.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	assert.Equal(t, createdUser.ID, user.ID)
	assert.Equal(t, createdUser.Email, user.Email)

	assert.WithinDuration(t, createdUser.CreatedAt.Time, user.CreatedAt.Time, time.Second)
}

func TestGetUserByEmailNotFound(t *testing.T) {
	_, err := testQueries.GetUserByEmail(context.Background(), "tes@icikiwir.com")
	assert.Error(t, err)
}

func TestGetUserByIDNotFound(t *testing.T) {
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	_, err := testQueries.GetUserByID(context.Background(), randomID)
	assert.Error(t, err)
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	user := createRandomUser(t)

	arg := CreateUserParams{
		Email:    user.Email,
		Name:     user.Name,
		Password: "another_hash",
	}

	_, err := testQueries.CreateUser(context.Background(), arg)
	assert.Error(t, err)
}

func TestUpdateUser(t *testing.T) {
	user := createRandomUser(t)

	newRandomEmail := util.GetRandomEmail()
	newRandomName := util.GetRandomName()

	arg := UpdateUserParams{
		ID:    user.ID,
		Email: newRandomEmail,
		Name:  newRandomName,
	}

	updatedUser, err := testQueries.UpdateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)

	assert.Equal(t, user.ID, updatedUser.ID)
	assert.Equal(t, newRandomEmail.String, updatedUser.Email)

	assert.WithinDuration(t, user.CreatedAt.Time, updatedUser.CreatedAt.Time, time.Second)

	assert.True(t, updatedUser.UpdatedAt.Time.After(user.UpdatedAt.Time) ||
		updatedUser.UpdatedAt.Time.Equal(user.UpdatedAt.Time))
}

func TestUpdateUserNotFound(t *testing.T) {
	userID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	newRandomEmail := util.GetRandomEmail()
	newRandomName := util.GetRandomName()

	arg := UpdateUserParams{
		ID:    userID,
		Email: newRandomEmail,
		Name:  newRandomName,
	}

	_, err := testQueries.UpdateUser(context.Background(), arg)
	assert.Error(t, err)
}

func TestDeleteUser(t *testing.T) {
	user := createRandomUser(t)

	err := testQueries.DeleteUser(context.Background(), user.ID)
	require.NoError(t, err)

	_, err = testQueries.GetUserByID(context.Background(), user.ID)
	assert.Error(t, err)
}

func TestDeleteUserNotFound(t *testing.T) {
	user := createRandomUser(t)

	err := testQueries.DeleteUser(context.Background(), user.ID)
	require.NoError(t, err)
}

func TestListUsers(t *testing.T) {
	for i := 0; i < 5; i++ {
		createRandomUser(t)
	}

	arg := ListUsersParams{
		Limit:  3,
		Offset: 0,
	}

	users, err := testQueries.ListUsers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, users, 3)

	arg.Offset = 2
	users, err = testQueries.ListUsers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, users, 3)
}

func TestListUsersEmpty(t *testing.T) {
	arg := ListUsersParams{
		Limit:  10,
		Offset: 0,
	}

	users, err := testQueries.ListUsers(context.Background(), arg)
	require.NoError(t, err)

	assert.NotNil(t, users)
}
