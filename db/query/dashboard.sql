-- name: GetDashboardSummary :one
SELECT 
    COALESCE(SUM(CASE WHEN type = 'INCOME' THEN amount ELSE 0 END), 0)::numeric AS total_income,
    COALESCE(SUM(CASE WHEN type = 'EXPENSE' THEN amount ELSE 0 END), 0)::numeric AS total_expense
FROM transactions
WHERE user_id = $1 
  AND transaction_date >= $2 
  AND transaction_date <= $3
  AND deleted_at IS NULL;

-- name: GetTotalBalance :one
SELECT 
    (COALESCE(SUM(CASE WHEN type = 'INCOME' THEN amount ELSE 0 END), 0) - 
     COALESCE(SUM(CASE WHEN type = 'EXPENSE' THEN amount ELSE 0 END), 0))::numeric AS current_balance
FROM transactions
WHERE user_id = $1 
  AND deleted_at IS NULL;

-- name: GetExpenseByCategory :many
SELECT 
    c.name AS category_name, 
    c.icon_url,
    SUM(t.amount)::numeric AS total_amount,
    COUNT(t.id) AS transaction_count
FROM transactions t
JOIN categories c ON t.category_id = c.id
WHERE t.user_id = $1 
  AND t.type = 'EXPENSE'
  AND t.transaction_date >= $2 
  AND t.transaction_date <= $3
  AND t.deleted_at IS NULL
GROUP BY c.name, c.icon_url
ORDER BY total_amount DESC
LIMIT 5;

-- name: GetDailyExpenseTrend :many
SELECT 
    DATE(transaction_date)::date AS date,
    SUM(amount)::numeric AS total_amount
FROM transactions
WHERE user_id = $1 
  AND type = 'EXPENSE'
  AND transaction_date >= $2 
  AND transaction_date <= $3
  AND deleted_at IS NULL
GROUP BY DATE(transaction_date)
ORDER BY date ASC;