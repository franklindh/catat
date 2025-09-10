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

func createTestUserForTransaction(t *testing.T) User {
	arg := CreateUserParams{
		Email:    "test" + uuid.New().String() + "@example.com",
		Password: "hashed_password_123",
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	return user
}

func createTestAccountForTransaction(t *testing.T, userID pgtype.UUID) Account {
	arg := CreateAccountParams{
		UserID:  userID,
		Name:    "Test Account " + uuid.New().String()[:8],
		Type:    "Savings",
		Balance: createRandomNumeric("1000.00"),
	}

	account, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	return account
}

func createTestCategoryForTransaction(t *testing.T, userID pgtype.UUID) Category {
	arg := CreateCategoryParams{
		UserID: userID,
		Name:   "Test Category " + uuid.New().String()[:8],
		Type:   "expense",
	}

	category, err := testQueries.CreateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category)

	return category
}

func createTestTransaction(t *testing.T) Transaction {
	user := createTestUserForTransaction(t)
	account := createTestAccountForTransaction(t, user.ID)
	category := createTestCategoryForTransaction(t, user.ID)

	arg := CreateTransactionParams{
		UserID:          user.ID,
		AccountID:       account.ID,
		CategoryID:      category.ID,
		Amount:          createRandomNumeric("100.50"),
		Description:     "Test transaction description",
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	transaction, err := testQueries.CreateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)

	assert.Equal(t, arg.UserID, transaction.UserID)
	assert.Equal(t, arg.AccountID, transaction.AccountID)
	assert.Equal(t, arg.CategoryID, transaction.CategoryID)
	assertNumericEqual(t, arg.Amount, transaction.Amount, "amount should match")
	assert.Equal(t, arg.Description, transaction.Description)
	assert.WithinDuration(t, arg.TransactionDate.Time, transaction.TransactionDate.Time, time.Second)
	assert.WithinDuration(t, time.Now(), transaction.CreatedAt.Time, 5*time.Second)

	return transaction
}

func TestCreateTransaction(t *testing.T) {
	createTestTransaction(t)
}

func TestGetTransaction(t *testing.T) {
	createdTransaction := createTestTransaction(t)

	arg := GetTransactionParams{
		ID:     createdTransaction.ID,
		UserID: createdTransaction.UserID,
	}

	transaction, err := testQueries.GetTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)

	assert.Equal(t, createdTransaction.ID, transaction.ID)
	assert.Equal(t, createdTransaction.UserID, transaction.UserID)
	assert.Equal(t, createdTransaction.AccountID, transaction.AccountID)
	assert.Equal(t, createdTransaction.CategoryID, transaction.CategoryID)
	assertNumericEqual(t, createdTransaction.Amount, transaction.Amount, "amount should match")
	assert.Equal(t, createdTransaction.Description, transaction.Description)
	assert.Equal(t, createdTransaction.TransactionDate, transaction.TransactionDate)
	assert.Equal(t, createdTransaction.CreatedAt, transaction.CreatedAt)
}

func TestListTransactions(t *testing.T) {
	user := createTestUserForTransaction(t)
	account := createTestAccountForTransaction(t, user.ID)
	category := createTestCategoryForTransaction(t, user.ID)

	for i := 0; i < 5; i++ {
		arg := CreateTransactionParams{
			UserID:          user.ID,
			AccountID:       account.ID,
			CategoryID:      category.ID,
			Amount:          createRandomNumeric("50.00"),
			Description:     "Test transaction " + string(rune(i+65)),
			TransactionDate: pgtype.Timestamptz{Time: time.Now().Add(time.Duration(-i) * time.Hour), Valid: true},
		}

		_, err := testQueries.CreateTransaction(context.Background(), arg)
		require.NoError(t, err)
	}

	arg := ListTransactionsParams{
		UserID: user.ID,
		Limit:  3,
		Offset: 0,
	}

	listedTransactions, err := testQueries.ListTransactions(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, listedTransactions, 3)

	for i := 0; i < len(listedTransactions)-1; i++ {
		assert.True(t, listedTransactions[i].TransactionDate.Time.After(listedTransactions[i+1].TransactionDate.Time) ||
			listedTransactions[i].TransactionDate.Time.Equal(listedTransactions[i+1].TransactionDate.Time))
	}
}

func TestListTransactionsByAccount(t *testing.T) {
	user := createTestUserForTransaction(t)
	account1 := createTestAccountForTransaction(t, user.ID)
	account2 := createTestAccountForTransaction(t, user.ID)
	category := createTestCategoryForTransaction(t, user.ID)

	for i := 0; i < 3; i++ {
		arg := CreateTransactionParams{
			UserID:          user.ID,
			AccountID:       account1.ID,
			CategoryID:      category.ID,
			Amount:          createRandomNumeric("50.00"),
			Description:     "Account1 transaction " + string(rune(i+65)),
			TransactionDate: pgtype.Timestamptz{Time: time.Now().Add(time.Duration(-i) * time.Hour), Valid: true},
		}

		_, err := testQueries.CreateTransaction(context.Background(), arg)
		require.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		arg := CreateTransactionParams{
			UserID:          user.ID,
			AccountID:       account2.ID,
			CategoryID:      category.ID,
			Amount:          createRandomNumeric("75.00"),
			Description:     "Account2 transaction " + string(rune(i+65)),
			TransactionDate: pgtype.Timestamptz{Time: time.Now().Add(time.Duration(-i) * time.Hour), Valid: true},
		}

		_, err := testQueries.CreateTransaction(context.Background(), arg)
		require.NoError(t, err)
	}

	arg := ListTransactionsByAccountParams{
		UserID:    user.ID,
		AccountID: account1.ID,
		Limit:     10,
		Offset:    0,
	}

	transactions, err := testQueries.ListTransactionsByAccount(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transactions, 3)

	for _, transaction := range transactions {
		assert.Equal(t, account1.ID, transaction.AccountID)
		assert.Equal(t, user.ID, transaction.UserID)
	}
}

func TestListTransactionsByDateRange(t *testing.T) {
	user := createTestUserForTransaction(t)
	account := createTestAccountForTransaction(t, user.ID)
	category := createTestCategoryForTransaction(t, user.ID)

	dates := []time.Time{
		time.Now().AddDate(0, 0, -10),
		time.Now().AddDate(0, 0, -5),
		time.Now().AddDate(0, 0, -2),
		time.Now().AddDate(0, 0, 1),
	}

	for i, date := range dates {
		arg := CreateTransactionParams{
			UserID:          user.ID,
			AccountID:       account.ID,
			CategoryID:      category.ID,
			Amount:          createRandomNumeric("100.00"),
			Description:     "Date range test " + string(rune(i+65)),
			TransactionDate: pgtype.Timestamptz{Time: date, Valid: true},
		}

		_, err := testQueries.CreateTransaction(context.Background(), arg)
		require.NoError(t, err)
	}

	startDate := time.Now().AddDate(0, 0, -6)
	endDate := time.Now().AddDate(0, 0, 0)

	arg := ListTransactionsByDateRangeParams{
		UserID:            user.ID,
		TransactionDate:   pgtype.Timestamptz{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Timestamptz{Time: endDate, Valid: true},
	}

	transactions, err := testQueries.ListTransactionsByDateRange(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transactions, 2)

	for _, transaction := range transactions {
		assert.True(t, transaction.TransactionDate.Time.After(startDate) || transaction.TransactionDate.Time.Equal(startDate))
		assert.True(t, transaction.TransactionDate.Time.Before(endDate) || transaction.TransactionDate.Time.Equal(endDate))
	}
}

func TestUpdateTransaction(t *testing.T) {
	createdTransaction := createTestTransaction(t)

	arg := UpdateTransactionParams{
		ID:              createdTransaction.ID,
		AccountID:       createdTransaction.AccountID,
		CategoryID:      createdTransaction.CategoryID,
		Amount:          createRandomNumeric("200.75"),
		Description:     "Updated transaction description",
		TransactionDate: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		UserID:          createdTransaction.UserID,
	}

	updatedTransaction, err := testQueries.UpdateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedTransaction)

	assert.Equal(t, arg.AccountID, updatedTransaction.AccountID)
	assert.Equal(t, arg.CategoryID, updatedTransaction.CategoryID)
	assertNumericEqual(t, arg.Amount, updatedTransaction.Amount, "amount should match")
	assert.Equal(t, arg.Description, updatedTransaction.Description)

	assert.WithinDuration(t, arg.TransactionDate.Time, updatedTransaction.TransactionDate.Time, time.Millisecond)
}

func TestDeleteTransaction(t *testing.T) {
	createdTransaction := createTestTransaction(t)

	arg := DeleteTransactionParams{
		ID:     createdTransaction.ID,
		UserID: createdTransaction.UserID,
	}

	err := testQueries.DeleteTransaction(context.Background(), arg)
	require.NoError(t, err)

	_, err = testQueries.GetTransaction(context.Background(), GetTransactionParams{
		ID:     createdTransaction.ID,
		UserID: createdTransaction.UserID,
	})
	assert.Error(t, err)
}

func TestGetTransactionNotFound(t *testing.T) {
	user := createTestUserForTransaction(t)
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	_, err := testQueries.GetTransaction(context.Background(), GetTransactionParams{
		ID:     randomID,
		UserID: user.ID,
	})
	assert.Error(t, err)
}

func TestDeleteTransactionNotFound(t *testing.T) {
	user := createTestUserForTransaction(t)
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	err := testQueries.DeleteTransaction(context.Background(), DeleteTransactionParams{
		ID:     randomID,
		UserID: user.ID,
	})
	assert.NoError(t, err)
}
