package db

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/franklindh/catat/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLStore_CreateUser(t *testing.T) {
	store := NewStore(testDB)

	arg := CreateUserParams{
		Email:    util.GetRandomEmail().String,
		Name:     util.GetRandomName().String,
		Password: "password123",
	}

	user, err := store.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	assert.Equal(t, arg.Email, user.Email)
	assert.Equal(t, arg.Name, user.Name)
	assert.WithinDuration(t, time.Now(), user.CreatedAt.Time, 5*time.Second)
}

func TestSQLStore_GetUserByID(t *testing.T) {
	store := NewStore(testDB)

	arg := CreateUserParams{
		Email:    util.GetRandomEmail().String,
		Name:     util.GetRandomName().String,
		Password: "password123",
	}

	createdUser, err := store.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdUser)

	foundUser, err := store.GetUserByID(context.Background(), createdUser.ID)
	require.NoError(t, err)
	require.NotEmpty(t, foundUser)

	assert.Equal(t, createdUser.ID, foundUser.ID)
	assert.Equal(t, createdUser.Email, foundUser.Email)
	assert.Equal(t, createdUser.Name, foundUser.Name)
	assert.Equal(t, createdUser.CreatedAt, foundUser.CreatedAt)
}

func TestSQLStore_GetUserByEmail(t *testing.T) {
	store := NewStore(testDB)

	arg := CreateUserParams{
		Email:    util.GetRandomEmail().String,
		Name:     util.GetRandomName().String,
		Password: "password123",
	}

	createdUser, err := store.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdUser)

	foundUser, err := store.GetUserByEmail(context.Background(), createdUser.Email)
	require.NoError(t, err)
	require.NotEmpty(t, foundUser)

	assert.Equal(t, createdUser.ID, foundUser.ID)
	assert.Equal(t, createdUser.Email, foundUser.Email)
	assert.Equal(t, createdUser.Name, foundUser.Name)
}

func TestSQLStore_UpdateUser(t *testing.T) {
	email := util.GetRandomEmail()
	name := util.GetRandomName()

	store := NewStore(testDB)

	arg := CreateUserParams{
		Email:    email.String,
		Name:     name.String,
		Password: "password123",
	}

	createdUser, err := store.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdUser)

	updateArg := UpdateUserParams{
		ID:    createdUser.ID,
		Email: email,
		Name:  name,
	}

	updatedUser, err := store.UpdateUser(context.Background(), updateArg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)

	assert.Equal(t, createdUser.ID, updatedUser.ID)
	assert.Equal(t, email.String, updatedUser.Email)
	assert.Equal(t, name.String, updatedUser.Name)
}

func TestSQLStore_DeleteUser(t *testing.T) {
	store := NewStore(testDB)

	arg := CreateUserParams{
		Email:    "test" + uuid.New().String() + "@example.com",
		Name:     "Test User",
		Password: "password123",
	}

	createdUser, err := store.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdUser)

	err = store.DeleteUser(context.Background(), createdUser.ID)
	require.NoError(t, err)

	_, err = store.GetUserByID(context.Background(), createdUser.ID)
	assert.Error(t, err)
}

func TestSQLStore_CreateAccount(t *testing.T) {
	store := NewStore(testDB)

	userArg := CreateUserParams{
		Email:    "test" + uuid.New().String() + "@example.com",
		Name:     "Test User",
		Password: "password123",
	}

	user, err := store.CreateUser(context.Background(), userArg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	accountArg := CreateAccountParams{
		UserID:  user.ID,
		Name:    "Test Account",
		Type:    "depository",
		Balance: pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true},
	}

	account, err := store.CreateAccount(context.Background(), accountArg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	assert.Equal(t, accountArg.UserID, account.UserID)
	assert.Equal(t, accountArg.Name, account.Name)
	assert.True(t, account.Balance.Valid)
	assert.Equal(t, accountArg.Type, account.Type)

	if account.Balance.Int != nil {
		assert.Equal(t, int64(0), account.Balance.Int.Int64())
	}
	assert.Equal(t, int32(0), account.Balance.Exp)
}

func TestSQLStore_CreateCategory(t *testing.T) {
	store := NewStore(testDB)

	userArg := CreateUserParams{
		Email:    "test" + uuid.New().String() + "@example.com",
		Name:     "Test User",
		Password: "password123",
	}

	user, err := store.CreateUser(context.Background(), userArg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	categoryArg := CreateCategoryParams{
		UserID: user.ID,
		Name:   "Test Category",
		Type:   "expense",
	}

	category, err := store.CreateCategory(context.Background(), categoryArg)
	require.NoError(t, err)
	require.NotEmpty(t, category)

	assert.Equal(t, categoryArg.UserID, category.UserID)
	assert.Equal(t, categoryArg.Name, category.Name)
	assert.Equal(t, categoryArg.Type, category.Type)
}

func TestSQLStore_CreateTransaction(t *testing.T) {
	store := NewStore(testDB)

	userArg := CreateUserParams{
		Email:    "test" + uuid.New().String() + "@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	user, err := store.CreateUser(context.Background(), userArg)
	require.NoError(t, err)

	accountArg := CreateAccountParams{
		UserID:  user.ID,
		Name:    "Test Account",
		Type:    "depository",
		Balance: pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true},
	}
	account, err := store.CreateAccount(context.Background(), accountArg)
	require.NoError(t, err)

	categoryArg := CreateCategoryParams{
		UserID: user.ID,
		Name:   "Test Category",
		Type:   "expense",
	}
	category, err := store.CreateCategory(context.Background(), categoryArg)
	require.NoError(t, err)

	amount := pgtype.Numeric{Int: big.NewInt(-250000000), Exp: -4, Valid: true}
	txArg := CreateTransactionParams{
		UserID:          user.ID,
		AccountID:       account.ID,
		CategoryID:      category.ID,
		Amount:          amount,
		Description:     "Test transaction",
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	transaction, err := store.CreateTransaction(context.Background(), txArg)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)

	assert.Equal(t, txArg.UserID, transaction.UserID)
	assert.Equal(t, txArg.AccountID, transaction.AccountID)
	assert.Equal(t, txArg.CategoryID, transaction.CategoryID)
	assert.Equal(t, txArg.Description, transaction.Description)
}
