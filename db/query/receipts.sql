-- name: CreateReceipt :one
INSERT INTO "receipts" (
  transaction_id,
  image_url,
  raw_text
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetReceiptByTransactionID :one
SELECT * FROM "receipts"
WHERE transaction_id = $1;

-- name: DeleteReceipt :exec
DELETE FROM "receipts"
WHERE id = $1;