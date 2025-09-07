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

func createTestReceipt(t *testing.T, transactionID pgtype.UUID) Receipt {
	arg := CreateReceiptParams{
		TransactionID: transactionID,
		ImageUrl:      "https://icikiwir.com/receipt_" + uuid.New().String() + ".jpg",
		RawText:       pgtype.Text{String: "tes mic 123", Valid: true},
	}

	receipt, err := testQueries.CreateReceipt(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, receipt)

	assert.Equal(t, arg.TransactionID, receipt.TransactionID)
	assert.Equal(t, arg.ImageUrl, receipt.ImageUrl)
	assert.Equal(t, arg.RawText, receipt.RawText)
	assert.WithinDuration(t, time.Now(), receipt.CreatedAt.Time, 5*time.Second)

	return receipt
}

func TestCreateReceipt(t *testing.T) {
	transaction := createTestTransaction(t)
	createTestReceipt(t, transaction.ID)
}

func TestGetReceiptByTransactionID(t *testing.T) {
	transaction := createTestTransaction(t)
	createdReceipt := createTestReceipt(t, transaction.ID)

	receipt, err := testQueries.GetReceiptByTransactionID(context.Background(), transaction.ID)
	require.NoError(t, err)
	require.NotEmpty(t, receipt)

	assert.Equal(t, createdReceipt.ID, receipt.ID)
	assert.Equal(t, createdReceipt.TransactionID, receipt.TransactionID)
	assert.Equal(t, createdReceipt.ImageUrl, receipt.ImageUrl)
	assert.Equal(t, createdReceipt.RawText, receipt.RawText)
	assert.Equal(t, createdReceipt.CreatedAt, receipt.CreatedAt)
}

func TestDeleteReceipt(t *testing.T) {
	transaction := createTestTransaction(t)
	createdReceipt := createTestReceipt(t, transaction.ID)

	err := testQueries.DeleteReceipt(context.Background(), createdReceipt.ID)
	require.NoError(t, err)

	_, err = testQueries.GetReceiptByTransactionID(context.Background(), transaction.ID)
	assert.Error(t, err)
}

func TestGetReceiptByTransactionIDNotFound(t *testing.T) {
	randomTransactionID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	_, err := testQueries.GetReceiptByTransactionID(context.Background(), randomTransactionID)
	assert.Error(t, err)
}

func TestDeleteReceiptNotFound(t *testing.T) {
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	err := testQueries.DeleteReceipt(context.Background(), randomID)
	assert.NoError(t, err)
}

func TestCreateReceiptWithEmptyRawText(t *testing.T) {
	transaction := createTestTransaction(t)

	arg := CreateReceiptParams{
		TransactionID: transaction.ID,
		ImageUrl:      "https://icikiwir.com/receipt_" + uuid.New().String() + ".jpg",
		RawText:       pgtype.Text{String: "", Valid: false},
	}

	receipt, err := testQueries.CreateReceipt(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, receipt)

	assert.Equal(t, arg.TransactionID, receipt.TransactionID)
	assert.Equal(t, arg.ImageUrl, receipt.ImageUrl)
	assert.Equal(t, arg.RawText, receipt.RawText)
}
