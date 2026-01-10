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

func createRandomTransactionTest(t *testing.T, userID int64, categoryID int64, transactionType string) CreateTransactionRow {
	categoryIDParam := pgtype.Int8{Int64: categoryID, Valid: true}
	arg := CreateTransactionParams{
		UserID:          userID,
		CategoryID:      categoryIDParam,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Type:            transactionType,
	}

	transaction, err := testStore.CreateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)

	require.Equal(t, arg.UserID, transaction.UserID)
	require.Equal(t, arg.Type, transaction.Type)
	require.Equal(t, arg.Description, transaction.Description)

	require.WithinDuration(t, arg.TransactionDate.Time, transaction.TransactionDate.Time, 1*time.Microsecond, "TransactionDate should be within 1 microsecond")

	require.NotZero(t, transaction.ID)
	require.NotZero(t, transaction.CreatedAt)

	return transaction
}

func TestCreateTransaction(t *testing.T) {
	user := createRandomUserTest(t)
	category := createRandomCategoryTest(t, user.ID, "EXPENSE")

	createRandomTransactionTest(t, user.ID, category.ID, "EXPENSE")
}

func TestGetTransactionById(t *testing.T) {
	user := createRandomUserTest(t)
	category := createRandomCategoryTest(t, user.ID, "EXPENSE")

	transaction1 := createRandomTransactionTest(t, user.ID, category.ID, "EXPENSE")

	transaction2, err := testStore.GetTransaction(context.Background(), GetTransactionParams{
		ID:     transaction1.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, transaction2)

	require.Equal(t, transaction1.ID, transaction2.ID)
	require.Equal(t, transaction1.UserID, transaction2.UserID)
	require.Equal(t, transaction1.Type, transaction2.Type)
	require.Equal(t, transaction1.Description, transaction2.Description)
	require.Equal(t, transaction1.TransactionDate.Time.Unix(), transaction2.TransactionDate.Time.Unix())
}

func TestListTransactions(t *testing.T) {
	user := createRandomUserTest(t)
	category := createRandomCategoryTest(t, user.ID, "EXPENSE")

	n := 5
	transactions1 := make([]CreateTransactionRow, n)
	for i := 0; i < n; i++ {
		transactions1[i] = createRandomTransactionTest(t, user.ID, category.ID, "EXPENSE")
	}

	arg := ListTransactionsParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	}

	transactions2, err := testStore.ListTransactions(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transactions2)

	require.GreaterOrEqual(t, len(transactions2), n)
	for _, trans := range transactions2 {
		require.Equal(t, user.ID, trans.UserID)
	}
}

func TestGetTransaction(t *testing.T) {
	user := createRandomUserTest(t)
	category := createRandomCategoryTest(t, user.ID, "EXPENSE")
	transaction1 := createRandomTransactionTest(t, user.ID, category.ID, "EXPENSE")

	transaction2, err := testStore.GetTransaction(context.Background(), GetTransactionParams{
		ID:     transaction1.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, transaction2)

	require.Equal(t, transaction1.ID, transaction2.ID)
	require.Equal(t, user.ID, transaction2.UserID)
	require.True(t, category.ID == transaction2.CategoryID.Int64 || !transaction2.CategoryID.Valid)
	require.NotZero(t, transaction2.ID)
	require.NotZero(t, transaction2.Amount)
	require.NotZero(t, transaction2.TransactionDate)
	require.NotEmpty(t, transaction2.Description)
}

func TestCreateTransactionWithTransferType(t *testing.T) {
	user := createRandomUserTest(t)

	arg := CreateTransactionParams{
		UserID:          user.ID,
		CategoryID:      pgtype.Int8{Valid: false},
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: "Income from transfer", Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Type:            "INCOME",
	}

	transaction, err := testStore.CreateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)
	require.Equal(t, "INCOME", transaction.Type)
	require.Equal(t, user.ID, transaction.UserID)
}

func TestUpdateTransaction(t *testing.T) {
	user := createRandomUserTest(t)
	category1 := createRandomCategoryTest(t, user.ID, "EXPENSE")
	category2 := createRandomCategoryTest(t, user.ID, "EXPENSE")
	require.NotEqual(t, category1.ID, category2.ID, "Categories should be different for a meaningful test")
	transaction1 := createRandomTransactionTest(t, user.ID, category1.ID, "EXPENSE")

	category2Param := pgtype.Int8{Int64: category2.ID, Valid: true}
	arg := UpdateTransactionParams{
		ID:              transaction1.ID,
		UserID:          user.ID,
		CategoryID:      category2Param,
		Amount:          util.RandomBalance(),
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Type:            "INCOME",
	}

	updatedTransaction, err := testStore.UpdateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedTransaction)

	require.Equal(t, transaction1.ID, updatedTransaction.ID)
	require.Equal(t, user.ID, updatedTransaction.UserID)
	require.Equal(t, category2.ID, updatedTransaction.CategoryID.Int64)

	require.NotEqual(t, transaction1.Amount.Int.Cmp(updatedTransaction.Amount.Int), 0)
	require.NotEqual(t, transaction1.Description, updatedTransaction.Description)
	require.NotEqual(t, transaction1.Type, updatedTransaction.Type)
}

func TestDeleteTransaction(t *testing.T) {
	user := createRandomUserTest(t)
	category := createRandomCategoryTest(t, user.ID, "EXPENSE")

	transaction := createRandomTransactionTest(t, user.ID, category.ID, "EXPENSE")

	err := testStore.DeleteTransaction(context.Background(), DeleteTransactionParams{
		ID:     transaction.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	_, err = testStore.GetTransaction(context.Background(), GetTransactionParams{
		ID:     transaction.ID,
		UserID: user.ID,
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows), "Expected sql.ErrNoRows, got %v", err)

	arg := ListTransactionsParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	}
	transactions, err := testStore.ListTransactions(context.Background(), arg)
	require.NoError(t, err)
	for _, trans := range transactions {
		require.NotEqual(t, transaction.ID, trans.ID)
	}
}

func TestCreateTransactionWithDifferentTypes(t *testing.T) {
	user := createRandomUserTest(t)
	expenseCategory := createRandomCategoryTest(t, user.ID, "EXPENSE")
	incomeCategory := createRandomCategoryTest(t, user.ID, "INCOME")

	expenseTransaction := createRandomTransactionTest(t, user.ID, expenseCategory.ID, "EXPENSE")
	require.Equal(t, "EXPENSE", expenseTransaction.Type)

	incomeTransaction := createRandomTransactionTest(t, user.ID, incomeCategory.ID, "INCOME")
	require.Equal(t, "INCOME", incomeTransaction.Type)
}
