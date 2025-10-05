package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/franklindh/catat/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomTransactionForTest(t *testing.T, userID, categoryID pgtype.UUID) Transaction {
	arg := CreateTransactionParams{
		UserID:          userID,
		CategoryID:      categoryID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	transaction, err := testStore.CreateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)

	require.Equal(t, arg.UserID, transaction.UserID)
	require.Equal(t, arg.CategoryID, transaction.CategoryID)
	require.Equal(t, arg.Amount, transaction.Amount)
	require.Equal(t, arg.Description, transaction.Description)

	require.WithinDuration(t, arg.TransactionDate.Time, transaction.TransactionDate.Time, 1*time.Microsecond, "TransactionDate should be within 1 microsecond")

	require.NotZero(t, transaction.ID)
	require.NotZero(t, transaction.CreatedAt)

	return transaction
}

func TestCreateTransaction(t *testing.T) {
	user := createRandomUserForTest(t)
	category := createRandomCategoryForTest(t, user.ID)

	createRandomTransactionForTest(t, user.ID, category.ID)
}

func TestGetTransactionById(t *testing.T) {
	user := createRandomUserForTest(t)
	category := createRandomCategoryForTest(t, user.ID)

	transaction := createRandomTransactionForTest(t, user.ID, category.ID)

	transaction2, err := testStore.GetTransaction(context.Background(), transaction.ID)
	require.NoError(t, err)
	require.NotEmpty(t, transaction2)

	require.Equal(t, transaction.ID, transaction2.ID)
	require.Equal(t, transaction.UserID, transaction2.UserID)
	require.Equal(t, transaction.CategoryID, transaction2.CategoryID)
	require.Equal(t, transaction.Amount, transaction2.Amount)
	require.Equal(t, transaction.Description, transaction2.Description)
	require.Equal(t, transaction.TransactionDate.Time.Unix(), transaction2.TransactionDate.Time.Unix())
	require.Equal(t, transaction.CreatedAt.Time.Unix(), transaction2.CreatedAt.Time.Unix())
}

func TestGetTransactions(t *testing.T) {
	user := createRandomUserForTest(t)
	category := createRandomCategoryForTest(t, user.ID)

	n := 5
	transactions := make([]Transaction, n)
	for i := 0; i < n; i++ {
		transaction := createRandomTransactionForTest(t, user.ID, category.ID)

		arg := CreateTransactionParams{
			UserID:          user.ID,
			CategoryID:      category.ID,
			Amount:          transaction.Amount,
			Description:     transaction.Description,
			TransactionDate: transaction.TransactionDate,
		}
		transaction, err := testStore.CreateTransaction(context.Background(), arg)
		require.NoError(t, err)
		transactions[i] = transaction
	}

	arg := GetTransactionsParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	}

	transactions2, err := testStore.GetTransactions(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transactions2)

	for _, trans := range transactions2 {
		require.Equal(t, user.ID, trans.UserID)
	}

	require.GreaterOrEqual(t, len(transactions2), n)
}
func TestGetTransaction(t *testing.T) {
	user := createRandomUserForTest(t)
	category := createRandomCategoryForTest(t, user.ID)
	transaction := createRandomTransactionForTest(t, user.ID, category.ID)

	transaction, err := testStore.GetTransaction(context.Background(), transaction.ID)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)

	require.Equal(t, user.ID, transaction.UserID)
	require.Equal(t, category.ID, transaction.CategoryID)
	require.NotZero(t, transaction.ID)
	require.NotZero(t, transaction.CreatedAt)
	require.NotZero(t, transaction.Amount)
	require.NotZero(t, transaction.TransactionDate)
	require.NotEmpty(t, transaction.Description)
}

func TestGetTransactionsByUserInDateRange(t *testing.T) {

	user := createRandomUserForTest(t)

	outsideStart := time.Now().AddDate(0, -1, 0)

	argOutside := CreateTransactionParams{
		UserID:          user.ID,
		CategoryID:      createRandomCategoryForTest(t, user.ID).ID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: "Outside Range", Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: outsideStart.Add(12 * time.Hour), Valid: true},
	}
	_, err := testStore.CreateTransaction(context.Background(), argOutside)
	require.NoError(t, err)

	startRange := time.Now().AddDate(0, 0, -5)
	endRange := time.Now().AddDate(0, 0, 5)
	argInRange1 := CreateTransactionParams{
		UserID:          user.ID,
		CategoryID:      createRandomCategoryForTest(t, user.ID).ID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: "In Range 1", Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: startRange.Add(1 * time.Hour), Valid: true},
	}
	_, err = testStore.CreateTransaction(context.Background(), argInRange1)
	require.NoError(t, err)

	argInRange2 := CreateTransactionParams{
		UserID:          user.ID,
		CategoryID:      createRandomCategoryForTest(t, user.ID).ID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: "In Range 2", Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: endRange.Add(-1 * time.Hour), Valid: true},
	}
	_, err = testStore.CreateTransaction(context.Background(), argInRange2)
	require.NoError(t, err)

	arg := GetTransactionsByDateRangeParams{
		UserID:            user.ID,
		TransactionDate:   pgtype.Timestamptz{Time: startRange, Valid: true},
		TransactionDate_2: pgtype.Timestamptz{Time: endRange, Valid: true},
	}

	transactions, err := testStore.GetTransactionsByDateRange(context.Background(), arg)
	require.NoError(t, err)

	require.Len(t, transactions, 2)

	for _, trans := range transactions {
		require.True(t, !trans.TransactionDate.Time.Before(startRange))
		require.True(t, !trans.TransactionDate.Time.After(endRange))
		require.Equal(t, user.ID, trans.UserID)
	}
}

func TestUpdateTransaction(t *testing.T) {
	user := createRandomUserForTest(t)
	category := createRandomCategoryForTest(t, user.ID)
	category2 := createRandomCategoryForTest(t, user.ID)
	require.NotEqual(t, category.ID, category2.ID, "Categories should be different for a meaningful test")
	transaction := createRandomTransactionForTest(t, user.ID, category.ID)

	arg := UpdateTransactionParams{
		ID:              transaction.ID,
		UserID:          user.ID,
		CategoryID:      category.ID,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	updatedTransaction, err := testStore.UpdateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedTransaction)

	require.Equal(t, transaction.ID, updatedTransaction.ID)
	require.Equal(t, user.ID, updatedTransaction.UserID)
	require.Equal(t, category.ID, updatedTransaction.CategoryID)

	require.NotEqual(t, transaction.Amount, updatedTransaction.Amount)
	require.NotEqual(t, transaction.Description, updatedTransaction.Description)
	require.NotEqual(t, transaction.TransactionDate, updatedTransaction.TransactionDate)
}

func TestDeleteTransaction(t *testing.T) {
	user := createRandomUserForTest(t)
	category := createRandomCategoryForTest(t, user.ID)

	transaction := createRandomTransactionForTest(t, user.ID, category.ID)

	err := testStore.DeleteTransaction(context.Background(), DeleteTransactionParams{
		ID:     transaction.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	_, err = testStore.GetTransaction(context.Background(), transaction.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows), "Expected sql.ErrNoRows, got %v", err)
}
