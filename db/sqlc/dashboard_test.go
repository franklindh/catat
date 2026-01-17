package db

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/franklindh/catat/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createTransactionWithDate(t *testing.T, userID int64, categoryID int64, transactionType string, transactionDate time.Time, amount pgtype.Numeric) CreateTransactionRow {
	categoryIDParam := pgtype.Int8{Int64: categoryID, Valid: true}
	arg := CreateTransactionParams{
		UserID:          userID,
		CategoryID:      categoryIDParam,
		Amount:          amount,
		Description:     pgtype.Text{String: util.RandomString(20), Valid: true},
		TransactionDate: pgtype.Timestamptz{Time: transactionDate, Valid: true},
		Type:            transactionType,
	}

	transaction, err := testStore.CreateTransaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transaction)

	return transaction
}

func TestGetDailyExpenseTrend(t *testing.T) {
	user := createRandomUserTest(t)
	category := createRandomCategoryTest(t, user.ID, "EXPENSE")

	now := time.Now()
	date1 := time.Date(now.Year(), now.Month(), now.Day()-2, 12, 0, 0, 0, time.UTC)
	date2 := time.Date(now.Year(), now.Month(), now.Day()-1, 12, 0, 0, 0, time.UTC)
	date3 := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)

	amount1 := pgtype.Numeric{Int: big.NewInt(10000), Exp: -4, Valid: true}
	amount2 := pgtype.Numeric{Int: big.NewInt(20000), Exp: -4, Valid: true}
	amount3 := pgtype.Numeric{Int: big.NewInt(15000), Exp: -4, Valid: true}

	createTransactionWithDate(t, user.ID, category.ID, "EXPENSE", date1, amount1)
	createTransactionWithDate(t, user.ID, category.ID, "EXPENSE", date2, amount2)
	createTransactionWithDate(t, user.ID, category.ID, "EXPENSE", date3, amount3)

	startDate := pgtype.Timestamptz{Time: date1.Add(-24 * time.Hour), Valid: true}
	endDate := pgtype.Timestamptz{Time: date3.Add(24 * time.Hour), Valid: true}

	arg := GetDailyExpenseTrendParams{
		UserID:            user.ID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	}

	trend, err := testStore.GetDailyExpenseTrend(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, trend)
	require.GreaterOrEqual(t, len(trend), 3)

	for i := 1; i < len(trend); i++ {
		require.True(t, trend[i].Date.Time.After(trend[i-1].Date.Time) || trend[i].Date.Time.Equal(trend[i-1].Date.Time))
	}
}

func TestGetDailyExpenseTrendEmpty(t *testing.T) {
	user := createRandomUserTest(t)

	now := time.Now()
	startDate := pgtype.Timestamptz{Time: now.Add(-7 * 24 * time.Hour), Valid: true}
	endDate := pgtype.Timestamptz{Time: now, Valid: true}

	arg := GetDailyExpenseTrendParams{
		UserID:            user.ID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	}

	trend, err := testStore.GetDailyExpenseTrend(context.Background(), arg)
	require.NoError(t, err)
	require.Empty(t, trend)
}

func TestGetDashboardSummary(t *testing.T) {
	user := createRandomUserTest(t)
	expenseCategory := createRandomCategoryTest(t, user.ID, "EXPENSE")
	incomeCategory := createRandomCategoryTest(t, user.ID, "INCOME")

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)

	incomeAmount := pgtype.Numeric{Int: big.NewInt(100000), Exp: -4, Valid: true}
	expenseAmount1 := pgtype.Numeric{Int: big.NewInt(30000), Exp: -4, Valid: true}
	expenseAmount2 := pgtype.Numeric{Int: big.NewInt(20000), Exp: -4, Valid: true}

	createTransactionWithDate(t, user.ID, incomeCategory.ID, "INCOME", date, incomeAmount)
	createTransactionWithDate(t, user.ID, expenseCategory.ID, "EXPENSE", date, expenseAmount1)
	createTransactionWithDate(t, user.ID, expenseCategory.ID, "EXPENSE", date, expenseAmount2)

	startDate := pgtype.Timestamptz{Time: date.Add(-24 * time.Hour), Valid: true}
	endDate := pgtype.Timestamptz{Time: date.Add(24 * time.Hour), Valid: true}

	arg := GetDashboardSummaryParams{
		UserID:            user.ID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	}

	summary, err := testStore.GetDashboardSummary(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, summary)

	require.True(t, summary.TotalIncome.Valid)
	require.Equal(t, 0, summary.TotalIncome.Int.Cmp(big.NewInt(100000)))

	require.True(t, summary.TotalExpense.Valid)
	require.Equal(t, 0, summary.TotalExpense.Int.Cmp(big.NewInt(50000)))
}

func TestGetDashboardSummaryEmpty(t *testing.T) {
	user := createRandomUserTest(t)

	now := time.Now()
	startDate := pgtype.Timestamptz{Time: now.Add(-7 * 24 * time.Hour), Valid: true}
	endDate := pgtype.Timestamptz{Time: now, Valid: true}

	arg := GetDashboardSummaryParams{
		UserID:            user.ID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	}

	summary, err := testStore.GetDashboardSummary(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, summary)

	require.True(t, summary.TotalIncome.Valid)
	require.True(t, summary.TotalExpense.Valid)
	require.Equal(t, 0, summary.TotalIncome.Int.Cmp(big.NewInt(0)))
	require.Equal(t, 0, summary.TotalExpense.Int.Cmp(big.NewInt(0)))
}

func TestGetExpenseByCategory(t *testing.T) {
	user := createRandomUserTest(t)
	category1 := createRandomCategoryTest(t, user.ID, "EXPENSE")
	category2 := createRandomCategoryTest(t, user.ID, "EXPENSE")
	category3 := createRandomCategoryTest(t, user.ID, "EXPENSE")

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)

	amount1a := pgtype.Numeric{Int: big.NewInt(50000), Exp: -4, Valid: true}
	amount1b := pgtype.Numeric{Int: big.NewInt(30000), Exp: -4, Valid: true}
	amount2 := pgtype.Numeric{Int: big.NewInt(20000), Exp: -4, Valid: true}
	amount3 := pgtype.Numeric{Int: big.NewInt(10000), Exp: -4, Valid: true}

	createTransactionWithDate(t, user.ID, category1.ID, "EXPENSE", date, amount1a)
	createTransactionWithDate(t, user.ID, category1.ID, "EXPENSE", date, amount1b)
	createTransactionWithDate(t, user.ID, category2.ID, "EXPENSE", date, amount2)
	createTransactionWithDate(t, user.ID, category3.ID, "EXPENSE", date, amount3)

	startDate := pgtype.Timestamptz{Time: date.Add(-24 * time.Hour), Valid: true}
	endDate := pgtype.Timestamptz{Time: date.Add(24 * time.Hour), Valid: true}

	arg := GetExpenseByCategoryParams{
		UserID:            user.ID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	}

	expenses, err := testStore.GetExpenseByCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, expenses)
	require.LessOrEqual(t, len(expenses), 5)

	for i := 1; i < len(expenses); i++ {
		prevTotal := expenses[i-1].TotalAmount.Int
		currTotal := expenses[i].TotalAmount.Int
		require.True(t, prevTotal.Cmp(currTotal) >= 0, "Results should be ordered by total_amount DESC")
	}

	found := false
	for _, exp := range expenses {
		if exp.CategoryName == category1.Name {
			require.Equal(t, 0, exp.TotalAmount.Int.Cmp(big.NewInt(80000)))
			require.Equal(t, int64(2), exp.TransactionCount)
			found = true
		}
	}
	require.True(t, found, "Category1 should be in the results")
}

func TestGetExpenseByCategoryEmpty(t *testing.T) {
	user := createRandomUserTest(t)

	now := time.Now()
	startDate := pgtype.Timestamptz{Time: now.Add(-7 * 24 * time.Hour), Valid: true}
	endDate := pgtype.Timestamptz{Time: now, Valid: true}

	arg := GetExpenseByCategoryParams{
		UserID:            user.ID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	}

	expenses, err := testStore.GetExpenseByCategory(context.Background(), arg)
	require.NoError(t, err)
	require.Empty(t, expenses)
}

func TestGetTotalBalance(t *testing.T) {
	user := createRandomUserTest(t)
	expenseCategory := createRandomCategoryTest(t, user.ID, "EXPENSE")
	incomeCategory := createRandomCategoryTest(t, user.ID, "INCOME")

	date := time.Now()

	incomeAmount1 := pgtype.Numeric{Int: big.NewInt(100000), Exp: -4, Valid: true}
	incomeAmount2 := pgtype.Numeric{Int: big.NewInt(50000), Exp: -4, Valid: true}
	expenseAmount1 := pgtype.Numeric{Int: big.NewInt(30000), Exp: -4, Valid: true}
	expenseAmount2 := pgtype.Numeric{Int: big.NewInt(20000), Exp: -4, Valid: true}

	createTransactionWithDate(t, user.ID, incomeCategory.ID, "INCOME", date, incomeAmount1)
	createTransactionWithDate(t, user.ID, incomeCategory.ID, "INCOME", date, incomeAmount2)
	createTransactionWithDate(t, user.ID, expenseCategory.ID, "EXPENSE", date, expenseAmount1)
	createTransactionWithDate(t, user.ID, expenseCategory.ID, "EXPENSE", date, expenseAmount2)

	balance, err := testStore.GetTotalBalance(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, balance.Valid)

	expectedBalance := big.NewInt(100000)
	require.Equal(t, 0, balance.Int.Cmp(expectedBalance), "Balance should be 100000")
}

func TestGetTotalBalanceEmpty(t *testing.T) {
	user := createRandomUserTest(t)

	balance, err := testStore.GetTotalBalance(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, balance.Valid)

	require.Equal(t, 0, balance.Int.Cmp(big.NewInt(0)))
}

func TestGetTotalBalanceOnlyIncome(t *testing.T) {
	user := createRandomUserTest(t)
	incomeCategory := createRandomCategoryTest(t, user.ID, "INCOME")

	date := time.Now()
	incomeAmount := pgtype.Numeric{Int: big.NewInt(100000), Exp: -4, Valid: true}

	createTransactionWithDate(t, user.ID, incomeCategory.ID, "INCOME", date, incomeAmount)

	balance, err := testStore.GetTotalBalance(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, balance.Valid)

	require.Equal(t, 0, balance.Int.Cmp(big.NewInt(100000)))
	require.True(t, balance.Int.Cmp(big.NewInt(0)) > 0)
}

func TestGetTotalBalanceOnlyExpense(t *testing.T) {
	user := createRandomUserTest(t)
	expenseCategory := createRandomCategoryTest(t, user.ID, "EXPENSE")

	date := time.Now()
	expenseAmount := pgtype.Numeric{Int: big.NewInt(50000), Exp: -4, Valid: true}

	createTransactionWithDate(t, user.ID, expenseCategory.ID, "EXPENSE", date, expenseAmount)

	balance, err := testStore.GetTotalBalance(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, balance.Valid)

	require.Equal(t, 0, balance.Int.Cmp(big.NewInt(-50000)))
	require.True(t, balance.Int.Cmp(big.NewInt(0)) < 0)
}

func TestGetTotalBalanceUserIsolation(t *testing.T) {
	user1 := createRandomUserTest(t)
	user2 := createRandomUserTest(t)
	incomeCategory1 := createRandomCategoryTest(t, user1.ID, "INCOME")
	incomeCategory2 := createRandomCategoryTest(t, user2.ID, "INCOME")

	date := time.Now()
	amount1 := pgtype.Numeric{Int: big.NewInt(100000), Exp: -4, Valid: true}
	amount2 := pgtype.Numeric{Int: big.NewInt(50000), Exp: -4, Valid: true}

	createTransactionWithDate(t, user1.ID, incomeCategory1.ID, "INCOME", date, amount1)
	createTransactionWithDate(t, user2.ID, incomeCategory2.ID, "INCOME", date, amount2)

	balance1, err := testStore.GetTotalBalance(context.Background(), user1.ID)
	require.NoError(t, err)
	require.True(t, balance1.Valid)
	require.Equal(t, 0, balance1.Int.Cmp(big.NewInt(100000)))

	balance2, err := testStore.GetTotalBalance(context.Background(), user2.ID)
	require.NoError(t, err)
	require.True(t, balance2.Valid)
	require.Equal(t, 0, balance2.Int.Cmp(big.NewInt(50000)))
}
