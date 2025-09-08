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

func toTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}

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

func assertNumericEqual(t *testing.T, expected, actual pgtype.Numeric, msg string) {
	t.Helper()

	expectedRat := numericToRat(expected)
	actualRat := numericToRat(actual)

	assert.Equal(t, 0, expectedRat.Cmp(actualRat), msg)
}

func numericToRat(n pgtype.Numeric) *big.Rat {
	if !n.Valid {
		return new(big.Rat)
	}

	rat := new(big.Rat)
	if n.Int == nil {
		return rat
	}

	rat.SetFrac(n.Int, big.NewInt(1))

	if n.Exp < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil)
		rat.Quo(rat, new(big.Rat).SetFrac(divisor, big.NewInt(1)))
	} else if n.Exp > 0 {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil)
		rat.Mul(rat, new(big.Rat).SetFrac(multiplier, big.NewInt(1)))
	}

	return rat
}

func TestListAccountsPaginated(t *testing.T) {

	user := createTestUser(t)

	accountNames := []string{"Account A", "Account B", "Account C", "Account D", "Account E"}

	for _, name := range accountNames {
		arg := CreateAccountParams{
			UserID:  user.ID,
			Name:    name,
			Type:    "Savings",
			Balance: createRandomNumeric("1000.00"),
		}

		account, err := testQueries.CreateAccount(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, account)
	}

	accounts, err := testQueries.ListAccounts(context.Background(), ListAccountsParams{
		UserID: user.ID,
		Limit:  2,
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	assert.Equal(t, "Account A", accounts[0].Name)
	assert.Equal(t, "Account B", accounts[1].Name)

	accounts, err = testQueries.ListAccounts(context.Background(), ListAccountsParams{
		UserID: user.ID,
		Limit:  2,
		Offset: 2,
	})
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	assert.Equal(t, "Account C", accounts[0].Name)
	assert.Equal(t, "Account D", accounts[1].Name)

	accounts, err = testQueries.ListAccounts(context.Background(), ListAccountsParams{
		UserID: user.ID,
		Limit:  2,
		Offset: 4,
	})
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "Account E", accounts[0].Name)
}

func TestCountAccountsByUser(t *testing.T) {

	user := createTestUser(t)

	initialCount, err := testQueries.CountAccountsByUser(context.Background(), user.ID)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		arg := CreateAccountParams{
			UserID:  user.ID,
			Name:    "Test Account " + string(rune(i+65)),
			Type:    "Savings",
			Balance: createRandomNumeric("1000.00"),
		}

		account, err := testQueries.CreateAccount(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, account)
	}

	finalCount, err := testQueries.CountAccountsByUser(context.Background(), user.ID)
	require.NoError(t, err)

	assert.Equal(t, initialCount+3, finalCount)
}

func TestCreateAccount(t *testing.T) {

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
	assert.Equal(t, arg.UserID, account.UserID)

	assertNumericEqual(t, arg.Balance, account.Balance, "balance should match")

	assert.WithinDuration(t, time.Now(), toTime(account.CreatedAt), 5*time.Second)
	assert.WithinDuration(t, time.Now(), toTime(account.UpdatedAt), 5*time.Second)
}

func TestGetAccount(t *testing.T) {

	user := createTestUser(t)

	arg := CreateAccountParams{
		UserID:  user.ID,
		Name:    "Test Account",
		Type:    "Savings",
		Balance: createRandomNumeric("1000.00"),
	}

	createdAccount, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	getArg := GetAccountParams{
		ID:     createdAccount.ID,
		UserID: user.ID,
	}

	account, err := testQueries.GetAccount(context.Background(), getArg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	assert.Equal(t, createdAccount.ID, account.ID)
	assert.Equal(t, createdAccount.Name, account.Name)
	assert.Equal(t, createdAccount.Type, account.Type)
	assert.Equal(t, createdAccount.UserID, account.UserID)

	assertNumericEqual(t, createdAccount.Balance, account.Balance, "balance should match")

	assert.Equal(t, createdAccount.CreatedAt, account.CreatedAt)
	assert.Equal(t, createdAccount.UpdatedAt, account.UpdatedAt)
}

func TestGetAccountNotFound(t *testing.T) {

	user := createTestUser(t)

	accountUser := createTestUser(t)
	accountArg := CreateAccountParams{
		UserID:  accountUser.ID,
		Name:    "Test Account",
		Type:    "Savings",
		Balance: createRandomNumeric("1000.00"),
	}

	createdAccount, err := testQueries.CreateAccount(context.Background(), accountArg)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	getArg := GetAccountParams{
		ID:     createdAccount.ID,
		UserID: user.ID,
	}

	_, err = testQueries.GetAccount(context.Background(), getArg)
	assert.Error(t, err)
}

func TestListAccounts(t *testing.T) {

	user := createTestUser(t)

	accountNames := []string{"Account A", "Account B", "Account C", "Account D", "Account E"}

	for _, name := range accountNames {
		arg := CreateAccountParams{
			UserID:  user.ID,
			Name:    name,
			Type:    "Savings",
			Balance: createRandomNumeric("1000.00"),
		}

		account, err := testQueries.CreateAccount(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, account)
	}

	accounts, err := testQueries.ListAccounts(context.Background(), ListAccountsParams{
		UserID: user.ID,
		Limit:  2,
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	assert.Equal(t, "Account A", accounts[0].Name)
	assert.Equal(t, "Account B", accounts[1].Name)

	accounts, err = testQueries.ListAccounts(context.Background(), ListAccountsParams{
		UserID: user.ID,
		Limit:  2,
		Offset: 2,
	})
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	assert.Equal(t, "Account C", accounts[0].Name)
	assert.Equal(t, "Account D", accounts[1].Name)

	accounts, err = testQueries.ListAccounts(context.Background(), ListAccountsParams{
		UserID: user.ID,
		Limit:  2,
		Offset: 4,
	})
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "Account E", accounts[0].Name)
}

func TestUpdateAccount(t *testing.T) {

	user := createTestUser(t)

	arg := CreateAccountParams{
		UserID:  user.ID,
		Name:    "Test Account",
		Type:    "Savings",
		Balance: createRandomNumeric("1000.00"),
	}

	createdAccount, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	updateArg := UpdateAccountParams{
		ID:     createdAccount.ID,
		Name:   "Updated Account Name",
		Type:   "Checking",
		UserID: user.ID,
	}

	updatedAccount, err := testQueries.UpdateAccount(context.Background(), updateArg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)

	assert.Equal(t, updateArg.Name, updatedAccount.Name)
	assert.Equal(t, updateArg.Type, updatedAccount.Type)
	assert.Equal(t, user.ID, updatedAccount.UserID)

	assertNumericEqual(t, createdAccount.Balance, updatedAccount.Balance, "balance should remain unchanged")

	createdTime := toTime(createdAccount.UpdatedAt)
	updatedTime := toTime(updatedAccount.UpdatedAt)
	assert.True(t, updatedTime.After(createdTime) || updatedTime.Equal(createdTime))
}

func TestDeleteAccount(t *testing.T) {

	user := createTestUser(t)

	arg := CreateAccountParams{
		UserID:  user.ID,
		Name:    "Test Account",
		Type:    "Savings",
		Balance: createRandomNumeric("1000.00"),
	}

	createdAccount, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	deleteArg := DeleteAccountParams{
		ID:     createdAccount.ID,
		UserID: user.ID,
	}

	err = testQueries.DeleteAccount(context.Background(), deleteArg)
	require.NoError(t, err)

	getArg := GetAccountParams{
		ID:     createdAccount.ID,
		UserID: user.ID,
	}

	_, err = testQueries.GetAccount(context.Background(), getArg)
	assert.Error(t, err)
}

func TestUpdateAccountBalance(t *testing.T) {

	user := createTestUser(t)

	arg := CreateAccountParams{
		UserID:  user.ID,
		Name:    "Test Account",
		Type:    "Savings",
		Balance: createRandomNumeric("1000.00"),
	}

	createdAccount, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	updateArg := UpdateAccountBalanceParams{
		ID:     createdAccount.ID,
		Amount: createRandomNumeric("500.00"),
	}

	updatedAccount, err := testQueries.UpdateAccountBalance(context.Background(), updateArg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)

	expectedBalance := createRandomNumeric("1500.00")
	assertNumericEqual(t, expectedBalance, updatedAccount.Balance, "balance should be 1500.00")
}
