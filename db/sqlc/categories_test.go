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

// Fungsi helper untuk membuat user test
func createTestUserForCategory(t *testing.T) User {
	arg := CreateUserParams{
		Email:        "test" + uuid.New().String() + "@example.com",
		PasswordHash: "hashed_password_123",
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	return user
}

// Fungsi helper untuk membuat category test
func createTestCategory(t *testing.T, userID pgtype.UUID, parentID pgtype.UUID) Category {
	arg := CreateCategoryParams{
		UserID:   userID,
		Name:     "Test Category " + uuid.New().String()[:8],
		Type:     "expense",
		ParentID: parentID, // pgtype.UUID zero value untuk NULL
	}

	category, err := testQueries.CreateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, category)

	assert.Equal(t, arg.Name, category.Name)
	assert.Equal(t, arg.Type, category.Type)
	assert.Equal(t, arg.UserID, category.UserID)

	// Jika parentID tidak valid, maka category.ParentID juga tidak valid (NULL)
	if !arg.ParentID.Valid {
		assert.False(t, category.ParentID.Valid)
	} else {
		assert.Equal(t, arg.ParentID, category.ParentID)
	}

	assert.WithinDuration(t, time.Now(), toTime(category.CreatedAt), 5*time.Second)
	assert.WithinDuration(t, time.Now(), toTime(category.UpdatedAt), 5*time.Second)

	return category
}

func TestCreateCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createTestCategory(t, user.ID, pgtype.UUID{}) // parentID kosong (NULL)
}

func TestCreateCategoryWithParent(t *testing.T) {
	user := createTestUserForCategory(t)

	// Buat parent category
	parentCategory := createTestCategory(t, user.ID, pgtype.UUID{})

	// Buat child category
	childCategory := createTestCategory(t, user.ID, parentCategory.ID)

	assert.Equal(t, parentCategory.ID, childCategory.ParentID)
}

func TestGetCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createdCategory := createTestCategory(t, user.ID, pgtype.UUID{})

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
	assert.Equal(t, createdCategory.ParentID, category.ParentID)
	assert.Equal(t, createdCategory.CreatedAt, category.CreatedAt)
	assert.Equal(t, createdCategory.UpdatedAt, category.UpdatedAt)
}

func TestListCategories(t *testing.T) {
	user := createTestUserForCategory(t)

	// Buat beberapa parent categories (tanpa parent)
	for i := 0; i < 3; i++ {
		createTestCategory(t, user.ID, pgtype.UUID{})
	}

	// Buat parent category dan sub categories untuk test hierarki
	parentCategory := createTestCategory(t, user.ID, pgtype.UUID{})
	for i := 0; i < 2; i++ {
		createTestCategory(t, user.ID, parentCategory.ID)
	}

	// List hanya parent categories (parent_id IS NULL)
	categories, err := testQueries.ListCategories(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, categories, 4) // 3 parent categories + 1 parent category yang memiliki children

	for _, category := range categories {
		assert.Equal(t, user.ID, category.UserID)
		assert.False(t, category.ParentID.Valid) // Harus parent categories saja
	}
}

func TestListSubCategories(t *testing.T) {
	user := createTestUserForCategory(t)

	// Buat parent category
	parentCategory := createTestCategory(t, user.ID, pgtype.UUID{})

	// Buat beberapa sub categories
	for i := 0; i < 3; i++ {
		createTestCategory(t, user.ID, parentCategory.ID)
	}

	// List sub categories
	arg := ListSubCategoriesParams{
		UserID:   user.ID,
		ParentID: parentCategory.ID,
	}

	categories, err := testQueries.ListSubCategories(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, categories, 3)

	for _, category := range categories {
		assert.Equal(t, user.ID, category.UserID)
		assert.Equal(t, parentCategory.ID, category.ParentID)
	}
}

func TestUpdateCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createdCategory := createTestCategory(t, user.ID, pgtype.UUID{})

	arg := UpdateCategoryParams{
		ID:       createdCategory.ID,
		Name:     "Updated Category Name",
		Type:     "income",
		ParentID: pgtype.UUID{}, // NULL
		UserID:   user.ID,
	}

	updatedCategory, err := testQueries.UpdateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedCategory)

	assert.Equal(t, arg.Name, updatedCategory.Name)
	assert.Equal(t, arg.Type, updatedCategory.Type)
	assert.False(t, updatedCategory.ParentID.Valid) // Harus NULL

	// Compare time values properly
	createdTime := toTime(createdCategory.UpdatedAt)
	updatedTime := toTime(updatedCategory.UpdatedAt)
	assert.True(t, updatedTime.After(createdTime) || updatedTime.Equal(createdTime))
}

func TestUpdateCategoryWithParent(t *testing.T) {
	user := createTestUserForCategory(t)

	// Buat dua categories
	category1 := createTestCategory(t, user.ID, pgtype.UUID{})
	category2 := createTestCategory(t, user.ID, pgtype.UUID{})

	// Update category1 untuk memiliki parent category2
	arg := UpdateCategoryParams{
		ID:       category1.ID,
		Name:     category1.Name,
		Type:     category1.Type,
		ParentID: category2.ID,
		UserID:   user.ID,
	}

	updatedCategory, err := testQueries.UpdateCategory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedCategory)

	assert.Equal(t, category2.ID, updatedCategory.ParentID)
}

func TestDeleteCategory(t *testing.T) {
	user := createTestUserForCategory(t)
	createdCategory := createTestCategory(t, user.ID, pgtype.UUID{})

	arg := DeleteCategoryParams{
		ID:     createdCategory.ID,
		UserID: user.ID,
	}

	err := testQueries.DeleteCategory(context.Background(), arg)
	require.NoError(t, err)

	// Try to get the deleted category
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
	// Delete biasanya tidak error meski record tidak ditemukan
	assert.NoError(t, err)
}
