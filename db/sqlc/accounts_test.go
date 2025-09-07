package db

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createRandomNumeric(value string) pgtype.Numeric {
	n := pgtype.Numeric{}
	_ = n.Scan(value)
	return n
}

// Fungsi helper untuk mengkonversi pgtype.Timestamptz ke time.Time
func toTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}

// Fungsi helper untuk membuat user test
func createTestUser(t *testing.T) User {
	arg := CreateUserParams{
		Email:        "test" + uuid.New().String() + "@example.com",
		PasswordHash: "hashed_password_123",
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	return user
}

// Fungsi helper untuk membandingkan numeric values
func assertNumericEqual(t *testing.T, expected, actual pgtype.Numeric, msg string) {
	t.Helper()

	// Convert both numerics to big.Rat for precise comparison
	expectedRat := numericToRat(expected)
	actualRat := numericToRat(actual)

	assert.Equal(t, expectedRat.Cmp(actualRat), 0, msg)
}

// Fungsi helper untuk mengkonversi pgtype.Numeric ke *big.Rat
func numericToRat(n pgtype.Numeric) *big.Rat {
	if !n.Valid {
		return new(big.Rat)
	}

	rat := new(big.Rat)
	if n.Int == nil {
		return rat
	}

	// Set numerator
	rat.SetFrac(n.Int, big.NewInt(1))

	// Apply exponent (divide by 10^abs(Exp) if Exp < 0, multiply by 10^Exp if Exp > 0)
	if n.Exp < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil)
		rat.Quo(rat, new(big.Rat).SetFrac(divisor, big.NewInt(1)))
	} else if n.Exp > 0 {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil)
		rat.Mul(rat, new(big.Rat).SetFrac(multiplier, big.NewInt(1)))
	}

	return rat
}

func createRandomAccount(t *testing.T) Account {
	// Buat user terlebih dahulu
	user := createTestUser(t)

	arg := CreateAccountParams{
		UserID:  user.ID,
		Name:    "Test Account",
		Type:    "Savings",
		Balance: createRandomNumeric("1000.00"),
	}

	account, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	assert.Equal(t, arg.Name, account.Name)
	assert.Equal(t, arg.Type, account.Type)

	// Compare numeric values
	assertNumericEqual(t, arg.Balance, account.Balance, "balance should match")

	assert.WithinDuration(t, time.Now(), toTime(account.CreatedAt), 5*time.Second)
	assert.WithinDuration(t, time.Now(), toTime(account.UpdatedAt), 5*time.Second)

	return account
}

func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

func TestGetAccount(t *testing.T) {
	createdAccount := createRandomAccount(t)

	arg := GetAccountParams{
		ID:     createdAccount.ID,
		UserID: createdAccount.UserID,
	}

	account, err := testQueries.GetAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	assert.Equal(t, createdAccount.ID, account.ID)
	assert.Equal(t, createdAccount.Name, account.Name)
	assert.Equal(t, createdAccount.Type, account.Type)

	// Compare numeric values
	assertNumericEqual(t, createdAccount.Balance, account.Balance, "balance should match")

	assert.Equal(t, createdAccount.CreatedAt, account.CreatedAt)
	assert.Equal(t, createdAccount.UpdatedAt, account.UpdatedAt)
}

func TestListAccounts(t *testing.T) {
	// Buat satu user untuk semua account
	user := createTestUser(t)

	// Create multiple accounts untuk user yang sama
	for i := 0; i < 3; i++ {
		arg := CreateAccountParams{
			UserID:  user.ID,
			Name:    "Test Account " + string(rune(i+65)), // A, B, C
			Type:    "Savings",
			Balance: createRandomNumeric("1000.00"),
		}

		account, err := testQueries.CreateAccount(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, account)
	}

	accounts, err := testQueries.ListAccounts(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, accounts, 3)

	for _, account := range accounts {
		assert.Equal(t, user.ID, account.UserID)
	}
}

func TestUpdateAccount(t *testing.T) {
	createdAccount := createRandomAccount(t)

	arg := UpdateAccountParams{
		ID:     createdAccount.ID,
		Name:   "Updated Account Name",
		Type:   "Checking",
		UserID: createdAccount.UserID,
	}

	updatedAccount, err := testQueries.UpdateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)

	assert.Equal(t, arg.Name, updatedAccount.Name)
	assert.Equal(t, arg.Type, updatedAccount.Type)

	// Compare numeric values (should remain unchanged)
	assertNumericEqual(t, createdAccount.Balance, updatedAccount.Balance, "balance should remain unchanged")

	// Compare time values properly
	createdTime := toTime(createdAccount.UpdatedAt)
	updatedTime := toTime(updatedAccount.UpdatedAt)
	assert.True(t, updatedTime.After(createdTime) || updatedTime.Equal(createdTime))
}

func TestDeleteAccount(t *testing.T) {
	createdAccount := createRandomAccount(t)

	arg := DeleteAccountParams{
		ID:     createdAccount.ID,
		UserID: createdAccount.UserID,
	}

	err := testQueries.DeleteAccount(context.Background(), arg)
	require.NoError(t, err)

	// Try to get the deleted account
	_, err = testQueries.GetAccount(context.Background(), GetAccountParams{
		ID:     createdAccount.ID,
		UserID: createdAccount.UserID,
	})
	assert.Error(t, err)
}

func TestUpdateAccountBalance(t *testing.T) {
	createdAccount := createRandomAccount(t)

	arg := UpdateAccountBalanceParams{
		ID:     createdAccount.ID,
		Amount: createRandomNumeric("500.00"),
	}

	updatedAccount, err := testQueries.UpdateAccountBalance(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)

	// Compare numeric values - should be 1500.00 (1000 + 500)
	expectedBalance := createRandomNumeric("1500.00")
	assertNumericEqual(t, expectedBalance, updatedAccount.Balance, "balance should be 1500.00")
}
