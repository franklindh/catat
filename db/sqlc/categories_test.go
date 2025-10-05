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

func createRandomCategoryForTest(t *testing.T, userID pgtype.UUID) Category {
	arg := CreateCategoryParams{
		UserID:  userID,
		Name:    util.RandomString(10),
		IconUrl: pgtype.Text{String: util.RandomString(20), Valid: true},
	}

	category, err := testStore.CreateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category)

	require.Equal(t, arg.UserID, category.UserID)
	require.Equal(t, arg.Name, category.Name)
	require.Equal(t, arg.IconUrl, category.IconUrl)

	require.NotZero(t, category.ID)
	require.NotZero(t, category.CreatedAt)

	return category
}

func TestCreateCategory(t *testing.T) {
	user := createRandomUserForTest(t)

	createRandomCategoryForTest(t, user.ID)
}

func TestGetCategory(t *testing.T) {
	user := createRandomUserForTest(t)
	category1 := createRandomCategoryForTest(t, user.ID)

	category2, err := testStore.GetCategory(context.Background(), category1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, category2)

	require.Equal(t, category1.ID, category2.ID)
	require.Equal(t, category1.UserID, category2.UserID)
	require.Equal(t, category1.Name, category2.Name)
	require.Equal(t, category1.IconUrl, category2.IconUrl)
	require.Equal(t, category1.CreatedAt, category2.CreatedAt)
	require.Equal(t, category1.UpdatedAt, category2.UpdatedAt)
}

func TestGetCategoriesByUser(t *testing.T) {
	user := createRandomUserForTest(t)

	n := 5
	categories1 := make([]Category, n)
	for i := 0; i < n; i++ {
		categories1[i] = createRandomCategoryForTest(t, user.ID)
	}

	categories2, err := testStore.GetCategoriesByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, categories2)

	require.Len(t, categories2, n)
	for _, category := range categories2 {
		require.Equal(t, user.ID, category.UserID)
	}

}

func TestGetCategoryByName(t *testing.T) {
	user := createRandomUserForTest(t)

	category1 := createRandomCategoryForTest(t, user.ID)
	nameToFind := category1.Name

	arg := GetCategoryByNameParams{
		UserID: user.ID,
		Name:   nameToFind,
	}
	category2, err := testStore.GetCategoryByName(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category2)

	require.Equal(t, category1.ID, category2.ID)
	require.Equal(t, category1.UserID, category2.UserID)
	require.Equal(t, category1.Name, category2.Name)
	require.Equal(t, category1.IconUrl, category2.IconUrl)
	require.Equal(t, category1.CreatedAt, category2.CreatedAt)
	require.Equal(t, category1.UpdatedAt, category2.UpdatedAt)
}

func TestUpdateCategory(t *testing.T) {
	user := createRandomUserForTest(t)
	category1 := createRandomCategoryForTest(t, user.ID)

	newName := util.RandomString(12)
	newIconUrl := pgtype.Text{String: util.RandomString(25), Valid: true}

	arg := UpdateCategoryParams{
		ID:      category1.ID,
		Name:    newName,
		IconUrl: newIconUrl,
		UserID:  user.ID,
	}

	category2, err := testStore.UpdateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category2)

	require.Equal(t, category1.ID, category2.ID)
	require.Equal(t, user.ID, category2.UserID)
	require.Equal(t, newName, category2.Name)
	require.Equal(t, newIconUrl, category2.IconUrl)

	require.Equal(t, category1.CreatedAt, category2.CreatedAt)

	require.True(t, category2.UpdatedAt.Time.After(category1.UpdatedAt.Time))
}

func TestDeleteCategory(t *testing.T) {
	user := createRandomUserForTest(t)
	category1 := createRandomCategoryForTest(t, user.ID)

	err := testStore.DeleteCategory(context.Background(), DeleteCategoryParams{
		ID:     category1.ID,
		UserID: user.ID,
	})
	require.NoError(t, err)

	_, err = testStore.GetCategory(context.Background(), category1.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows), "Expected sql.ErrNoRows, got %v", err)
}
