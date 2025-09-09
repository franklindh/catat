package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	arg := CreateUserParams{
		Email:    "test" + uuid.New().String() + "@icikiwir.com",
		Name:     "Jamaludin",
		Password: "hashed_password_123",
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	assert.Equal(t, arg.Email, user.Email)
	assert.WithinDuration(t, time.Now(), user.CreatedAt.Time, 5*time.Second)
	assert.WithinDuration(t, time.Now(), user.UpdatedAt.Time, 5*time.Second)

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
	assert.Equal(t, createdUser.Password, user.Password)
	assert.Equal(t, createdUser.CreatedAt, user.CreatedAt)
	assert.Equal(t, createdUser.UpdatedAt, user.UpdatedAt)
}

func TestGetUserByID(t *testing.T) {
	createdUser := createRandomUser(t)

	user, err := testQueries.GetUserByID(context.Background(), createdUser.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	assert.Equal(t, createdUser.ID, user.ID)
	assert.Equal(t, createdUser.Email, user.Email)
	assert.Equal(t, createdUser.Password, user.Password)
	assert.Equal(t, createdUser.CreatedAt, user.CreatedAt)
	assert.Equal(t, createdUser.UpdatedAt, user.UpdatedAt)
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

	oldUser := createRandomUser(t)

	newEmail := "updated" + uuid.New().String() + "@icikiwir.com"

	nameChanged := oldUser.Name + " " + "berubah"

	arg := UpdateUserParams{
		ID:    oldUser.ID,
		Email: newEmail,
		Name:  nameChanged,
	}

	updatedUser, err := testQueries.UpdateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)

	assert.Equal(t, oldUser.ID, updatedUser.ID)
	assert.Equal(t, newEmail, updatedUser.Email)
	assert.Equal(t, oldUser.CreatedAt, updatedUser.CreatedAt)

	assert.True(t, updatedUser.UpdatedAt.Time.Equal(oldUser.UpdatedAt.Time) ||
		updatedUser.UpdatedAt.Time.After(oldUser.UpdatedAt.Time))
}

func TestUpdateUserNotFound(t *testing.T) {
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	arg := UpdateUserParams{
		ID:    randomID,
		Email: "updated@icikiwir.com",
		Name:  "test",
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
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	err := testQueries.DeleteUser(context.Background(), randomID)
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
