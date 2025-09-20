package db

import (
	"context"
	"testing"
	"time"

	"github.com/franklindh/catat/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestUserForCategory(t *testing.T) User {
	arg := CreateUserParams{
		Email:    util.GetRandomEmail().String,
		Password: util.GetRandomName().String,
	}

	createUserRow, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, createUserRow)

	user := User{
		ID:        createUserRow.ID,
		Email:     createUserRow.Email,
		Name:      createUserRow.Name,
		CreatedAt: createUserRow.CreatedAt,
	}

	return user
}

func createTestCategory(t *testing.T, userID pgtype.UUID) Category {
	arg := CreateCategoryParams{
		UserID: userID,
		Name:   "Test Category " + uuid.New().String()[:8],
		Type:   "expense",
	}

	category, err := testQueries.CreateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category)

	assert.Equal(t, arg.Name, category.Name)
	assert.Equal(t, arg.Type, category.Type)
	assert.Equal(t, arg.UserID, category.UserID)

	assert.WithinDuration(t, time.Now(), toTime(category.CreatedAt), 5*time.Second)
	assert.WithinDuration(t, time.Now(), toTime(category.UpdatedAt), 5*time.Second)

	return category
}

func TestCreateCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createTestCategory(t, user.ID)
}

func TestGetCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createdCategory := createTestCategory(t, user.ID)

	arg := GetCategoryParams{
		ID:     createdCategory.ID,
		UserID: user.ID,
	}

	category, err := testQueries.GetCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category)

	assert.Equal(t, createdCategory.ID, category.ID)
	assert.Equal(t, createdCategory.Name, category.Name)
	assert.Equal(t, createdCategory.Type, category.Type)
	assert.Equal(t, createdCategory.UserID, category.UserID)
	assert.Equal(t, createdCategory.CreatedAt, category.CreatedAt)
	assert.Equal(t, createdCategory.UpdatedAt, category.UpdatedAt)
}

func TestListCategories(t *testing.T) {
	user := createTestUserForCategory(t)

	for i := 0; i < 3; i++ {
		createTestCategory(t, user.ID)
	}

	categories, err := testQueries.ListCategories(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, categories, 3)

	for _, category := range categories {
		assert.Equal(t, user.ID, category.UserID)
	}
}

func TestUpdateCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createdCategory := createTestCategory(t, user.ID)

	arg := UpdateCategoryParams{
		ID:     createdCategory.ID,
		Name:   "Updated Category Name",
		Type:   "income",
		UserID: user.ID,
	}

	updatedCategory, err := testQueries.UpdateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedCategory)

	assert.Equal(t, arg.Name, updatedCategory.Name)
	assert.Equal(t, arg.Type, updatedCategory.Type)

	createdTime := toTime(createdCategory.UpdatedAt)
	updatedTime := toTime(updatedCategory.UpdatedAt)
	assert.True(t, updatedTime.After(createdTime) || updatedTime.Equal(createdTime))
}

func TestDeleteCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createdCategory := createTestCategory(t, user.ID)

	arg := DeleteCategoryParams{
		ID:     createdCategory.ID,
		UserID: user.ID,
	}

	err := testQueries.DeleteCategory(context.Background(), arg)
	require.NoError(t, err)

	_, err = testQueries.GetCategory(context.Background(), GetCategoryParams{
		ID:     createdCategory.ID,
		UserID: user.ID,
	})
	assert.Error(t, err)
}

func TestGetCategoryNotFound(t *testing.T) {
	user := createTestUserForCategory(t)
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	_, err := testQueries.GetCategory(context.Background(), GetCategoryParams{
		ID:     randomID,
		UserID: user.ID,
	})
	assert.Error(t, err)
}

func TestDeleteCategoryNotFound(t *testing.T) {
	user := createTestUserForCategory(t)
	randomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	err := testQueries.DeleteCategory(context.Background(), DeleteCategoryParams{
		ID:     randomID,
		UserID: user.ID,
	})

	assert.NoError(t, err)
}
