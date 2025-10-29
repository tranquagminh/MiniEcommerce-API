package postgres

import (
	"context"
	"errors"
	"fmt"
	"product-service/internal/application"
	"product-service/internal/domain"

	"gorm.io/gorm"
)

var _ application.CategoryRepository = (*CategoryRepository)(nil)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) WithTx(tx *gorm.DB) application.CategoryRepository {
	return &CategoryRepository{db: tx}
}

func (r *CategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	model := &CategoryModel{}
	model.FromDomain(category)

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		if IsDuplicateError(result.Error) {
			return ErrDuplicateSlug
		}
		return fmt.Errorf("failed to create category: %w", result.Error)
	}

	category.ID = model.ID
	category.CreatedAt = model.CreatedAt
	category.UpdatedAt = model.UpdatedAt

	return nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uint) (*domain.Category, error) {
	var model CategoryModel

	err := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children").
		First(&model, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category by id: %w", err)
	}

	return model.ToDomain(), nil
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	var model CategoryModel

	err := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children").
		Where("slug = ?", slug).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category by slug: %w", err)
	}

	return model.ToDomain(), nil
}

func (r *CategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	model := &CategoryModel{}
	model.FromDomain(category)

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		if IsDuplicateError(result.Error) {
			return ErrDuplicateSlug
		}
		return fmt.Errorf("failed to update category: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrCategoryNotFound
	}

	category.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&CategoryModel{}, id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete category: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrCategoryNotFound
	}

	return nil
}

func (r *CategoryRepository) List(ctx context.Context, activeOnly bool) ([]*domain.Category, error) {
	var models []*CategoryModel

	query := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children")

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	err := query.Order("display_order ASC, name ASC").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	categories := make([]*domain.Category, len(models))
	for i, model := range models {
		categories[i] = model.ToDomain()
	}

	return categories, nil
}

func (r *CategoryRepository) GetRootCategories(ctx context.Context, activeOnly bool) ([]*domain.Category, error) {
	var models []*CategoryModel

	query := r.db.WithContext(ctx).
		Where("parent_id IS NULL").
		Preload("Children")

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	err := query.Order("display_order ASC, name ASC").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get root categories: %w", err)
	}

	categories := make([]*domain.Category, len(models))
	for i, model := range models {
		categories[i] = model.ToDomain()
	}

	return categories, nil
}

func (r *CategoryRepository) GetChildren(ctx context.Context, parentID uint) ([]*domain.Category, error) {
	var models []*CategoryModel

	err := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("display_order ASC, name ASC").
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get children categories: %w", err)
	}

	categories := make([]*domain.Category, len(models))
	for i, model := range models {
		categories[i] = model.ToDomain()
	}

	return categories, nil
}

func (r *CategoryRepository) ExistsSlug(ctx context.Context, slug string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&CategoryModel{}).
		Where("slug = ?", slug)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check slug exists: %w", err)
	}

	return count > 0, nil
}
