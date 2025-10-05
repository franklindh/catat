package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/franklindh/catat/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomUserForTest(t *testing.T) User {
	arg := CreateUserParams{
		GoogleID:  util.RandomString(12),
		Email:     util.RandomEmail(),
		Name:      pgtype.Text{String: util.RandomString(6), Valid: true},
		Balance:   util.RandomBalance(),
		AvatarUrl: pgtype.Text{String: util.RandomString(20), Valid: true},
	}

	user, err := testStore.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.GoogleID, user.GoogleID)
	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.Name, user.Name)
	require.Equal(t, arg.Balance, user.Balance)
	require.Equal(t, arg.AvatarUrl, user.AvatarUrl)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUserForTest(t)
}

func TestGetUser(t *testing.T) {
	user1 := createRandomUserForTest(t)

	user2, err := testStore.GetUser(context.Background(), user1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.GoogleID, user2.GoogleID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Name, user2.Name)
	require.Equal(t, user1.AvatarUrl, user2.AvatarUrl)
	require.Equal(t, user1.Balance, user2.Balance)
	require.Equal(t, user1.CreatedAt, user2.CreatedAt)
	require.Equal(t, user1.UpdatedAt, user2.UpdatedAt)
}

func TestGetUserByEmail(t *testing.T) {
	user1 := createRandomUserForTest(t)

	user2, err := testStore.GetUserByEmail(context.Background(), user1.Email)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.GoogleID, user2.GoogleID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Name, user2.Name)
	require.Equal(t, user1.Balance, user2.Balance)
	require.Equal(t, user1.AvatarUrl, user2.AvatarUrl)
	require.Equal(t, user1.CreatedAt, user2.CreatedAt)
	require.Equal(t, user1.UpdatedAt, user2.UpdatedAt)
}

func TestGetUserByGoogleID(t *testing.T) {
	user1 := createRandomUserForTest(t)

	user2, err := testStore.GetUserByGoogleID(context.Background(), user1.GoogleID)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.GoogleID, user2.GoogleID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Name, user2.Name)
	require.Equal(t, user1.Balance, user2.Balance)
	require.Equal(t, user1.AvatarUrl, user2.AvatarUrl)
	require.Equal(t, user1.CreatedAt.Time.Unix(), user2.CreatedAt.Time.Unix())
	require.Equal(t, user1.UpdatedAt.Time.Unix(), user2.UpdatedAt.Time.Unix())
}

func TestUpdateUser(t *testing.T) {
	user1 := createRandomUserForTest(t)

	newName := pgtype.Text{String: util.RandomString(8), Valid: true}
	newAvatarUrl := pgtype.Text{String: util.RandomString(25), Valid: true}

	arg := UpdateUserParams{
		ID:        user1.ID,
		Name:      newName,
		AvatarUrl: newAvatarUrl,
	}

	err := testStore.UpdateUser(context.Background(), arg)
	require.NoError(t, err)

	user2, err := testStore.GetUser(context.Background(), user1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, newName, user2.Name)
	require.Equal(t, newAvatarUrl, user2.AvatarUrl)

	require.Equal(t, user1.GoogleID, user2.GoogleID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Balance, user2.Balance)

	require.True(t, user2.UpdatedAt.Time.After(user1.UpdatedAt.Time))
}

func TestUpdateUserBalance(t *testing.T) {
	user1 := createRandomUserForTest(t)

	newBalance := util.RandomBalance()

	arg := UpdateUserBalanceParams{
		ID:      user1.ID,
		Balance: newBalance,
	}

	err := testStore.UpdateUserBalance(context.Background(), arg)
	require.NoError(t, err)

	user2, err := testStore.GetUser(context.Background(), user1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, newBalance, user2.Balance)

	require.Equal(t, user1.GoogleID, user2.GoogleID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Name, user2.Name)
	require.Equal(t, user1.AvatarUrl, user2.AvatarUrl)

	require.True(t, user2.UpdatedAt.Time.After(user1.UpdatedAt.Time))
}

func TestDeleteUser(t *testing.T) {

	user1 := createRandomUserForTest(t)

	err := testStore.DeleteUser(context.Background(), user1.ID)
	require.NoError(t, err)

	_, err = testStore.GetUser(context.Background(), user1.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows), "Expected sql.ErrNoRows, got %v", err)
}
