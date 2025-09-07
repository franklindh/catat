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
		Email:        "test" + uuid.New().String() + "@example.com",
		PasswordHash: "hashed_password_123",
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	assert.Equal(t, arg.Email, user.Email)
	assert.Equal(t, arg.PasswordHash, user.PasswordHash)
	assert.WithinDuration(t, time.Now(), toTime(user.CreatedAt), 5*time.Second)
	assert.WithinDuration(t, time.Now(), toTime(user.UpdatedAt), 5*time.Second)

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
	assert.Equal(t, createdUser.PasswordHash, user.PasswordHash)
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
	assert.Equal(t, createdUser.PasswordHash, user.PasswordHash)
	assert.Equal(t, createdUser.CreatedAt, user.CreatedAt)
	assert.Equal(t, createdUser.UpdatedAt, user.UpdatedAt)
}

func TestGetUserByEmailNotFound(t *testing.T) {
	_, err := testQueries.GetUserByEmail(context.Background(), "nonexistent@example.com")
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
		Email:        user.Email,
		PasswordHash: "another_hash",
	}

	_, err := testQueries.CreateUser(context.Background(), arg)
	assert.Error(t, err)
}
