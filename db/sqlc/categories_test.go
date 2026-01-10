package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/franklindh/catat/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomCategoryTest(t *testing.T, userID int64, categoryType string) CreateCategoryRow {
	arg := CreateCategoryParams{
		UserID:  userID,
		Name:    util.RandomString(10),
		Type:    categoryType,
		IconUrl: pgtype.Text{String: util.RandomString(20), Valid: true},
	}

	category, err := testStore.CreateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category)

	require.Equal(t, arg.UserID, category.UserID)
	require.Equal(t, arg.Name, category.Name)
	require.Equal(t, arg.Type, category.Type)
	require.Equal(t, arg.IconUrl, category.IconUrl)

	require.NotZero(t, category.ID)
	require.NotZero(t, category.CreatedAt)

	return category
}

func TestCreateCategory(t *testing.T) {
	user := createRandomUserTest(t)
	createRandomCategoryTest(t, user.ID, "EXPENSE")
}

func TestGetCategory(t *testing.T) {
	user := createRandomUserTest(t)
	category1 := createRandomCategoryTest(t, user.ID, "EXPENSE")

	category2, err := testStore.GetCategory(context.Background(), category1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, category2)

	require.Equal(t, category1.ID, category2.ID)
	require.Equal(t, category1.UserID, category2.UserID)
	require.Equal(t, category1.Name, category2.Name)
	require.Equal(t, category1.Type, category2.Type)
	require.Equal(t, category1.IconUrl, category2.IconUrl)
	require.Equal(t, category1.CreatedAt, category2.CreatedAt)
	require.Equal(t, category1.UpdatedAt, category2.UpdatedAt)
}

func TestGetCategoriesByUser(t *testing.T) {
	user := createRandomUserTest(t)

	n := 5
	categories1 := make([]CreateCategoryRow, n)
	categoryType := "EXPENSE"
	for i := 0; i < n; i++ {
		categories1[i] = createRandomCategoryTest(t, user.ID, categoryType)
	}

	arg := GetCategoriesByUserParams{
		UserID: user.ID,
		Type:   categoryType,
	}
	categories2, err := testStore.GetCategoriesByUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, categories2)

	require.Len(t, categories2, n)
	for _, category := range categories2 {
		require.Equal(t, user.ID, category.UserID)
		require.Equal(t, categoryType, category.Type)
	}
}

func TestGetCategoriesByUserWithDifferentTypes(t *testing.T) {
	user := createRandomUserTest(t)

	expenseCount := 3
	for i := 0; i < expenseCount; i++ {
		createRandomCategoryTest(t, user.ID, "EXPENSE")
	}

	incomeCount := 2
	for i := 0; i < incomeCount; i++ {
		createRandomCategoryTest(t, user.ID, "INCOME")
	}

	expenseArg := GetCategoriesByUserParams{
		UserID: user.ID,
		Type:   "EXPENSE",
	}
	expenseCategories, err := testStore.GetCategoriesByUser(context.Background(), expenseArg)
	require.NoError(t, err)
	require.Len(t, expenseCategories, expenseCount)
	for _, category := range expenseCategories {
		require.Equal(t, "EXPENSE", category.Type)
	}

	incomeArg := GetCategoriesByUserParams{
		UserID: user.ID,
		Type:   "INCOME",
	}
	incomeCategories, err := testStore.GetCategoriesByUser(context.Background(), incomeArg)
	require.NoError(t, err)
	require.Len(t, incomeCategories, incomeCount)
	for _, category := range incomeCategories {
		require.Equal(t, "INCOME", category.Type)
	}
}

func TestUpdateCategory(t *testing.T) {
	user := createRandomUserTest(t)
	category1 := createRandomCategoryTest(t, user.ID, "EXPENSE")

	newName := util.RandomString(12)
	newIconUrl := pgtype.Text{String: util.RandomString(25), Valid: true}
	newType := "INCOME"

	arg := UpdateCategoryParams{
		ID:      category1.ID,
		Name:    newName,
		IconUrl: newIconUrl,
		Type:    newType,
		UserID:  user.ID,
	}

	category2, err := testStore.UpdateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category2)

	require.Equal(t, category1.ID, category2.ID)
	require.Equal(t, user.ID, category2.UserID)
	require.Equal(t, newName, category2.Name)
	require.Equal(t, newType, category2.Type)
	require.Equal(t, newIconUrl, category2.IconUrl)

	require.Equal(t, category1.CreatedAt, category2.CreatedAt)

	require.True(t, category2.UpdatedAt.Time.After(category1.UpdatedAt.Time))
}

func TestDeleteCategory(t *testing.T) {
	user := createRandomUserTest(t)
	category1 := createRandomCategoryTest(t, user.ID, "EXPENSE")

	err := testStore.DeleteCategory(context.Background(), DeleteCategoryParams{
		ID:     category1.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	_, err = testStore.GetCategory(context.Background(), category1.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows), "Expected sql.ErrNoRows, got %v", err)

	arg := GetCategoriesByUserParams{
		UserID: user.ID,
		Type:   category1.Type,
	}
	categories, err := testStore.GetCategoriesByUser(context.Background(), arg)
	require.NoError(t, err)
	for _, category := range categories {
		require.NotEqual(t, category1.ID, category.ID)
	}
}
