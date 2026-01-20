package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type CategorySeed struct {
	Name string
	Type string
	Icon string
}

var DefaultCategories = []CategorySeed{

	{Name: "Makanan & Minuman", Type: "EXPENSE", Icon: "food"},
	{Name: "Transportasi", Type: "EXPENSE", Icon: "transport"},
	{Name: "Belanja Rutin", Type: "EXPENSE", Icon: "cart"},
	{Name: "Tagihan & Pulsa", Type: "EXPENSE", Icon: "bill"},
	{Name: "Hiburan", Type: "EXPENSE", Icon: "movie"},
	{Name: "Kesehatan", Type: "EXPENSE", Icon: "medical"},
	{Name: "Lain-lain", Type: "EXPENSE", Icon: "dots"},

	{Name: "Gaji Utama", Type: "INCOME", Icon: "money"},
	{Name: "Bonus/Tunjangan", Type: "INCOME", Icon: "gift"},
	{Name: "Investasi", Type: "INCOME", Icon: "chart"},
}

func (store *SQLStore) CreateDefaultCategories(ctx context.Context, userID int64) error {
	for _, cat := range DefaultCategories {
		_, err := store.CreateCategory(ctx, CreateCategoryParams{
			UserID:  userID,
			Name:    cat.Name,
			Type:    cat.Type,
			IconUrl: pgtype.Text{String: cat.Icon, Valid: true},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
