package postgres

import (
	"context"
	"errors"
	"fmt"
	"product-service/internal/application"
	"product-service/internal/domain"

	"gorm.io/gorm"
)

var _ application.ProductRepository = (*ProductRepository)(nil)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) WithTx(tx *gorm.DB) application.ProductRepository {
	return &ProductRepository{db: tx}
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	model := &ProductModel{}
	model.FromDomain(product)

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		if IsDuplicateError(result.Error) {
			return ErrDuplicateProduct
		}
		return fmt.Errorf("failed to create product: %w", result.Error)
	}

	product.ID = model.ID
	product.CreatedAt = model.CreatedAt
	product.UpdatedAt = model.UpdatedAt

	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uint) (*domain.Product, error) {
	var model ProductModel

	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Images").
		Preload("Variants").
		Preload("Tags").
		First(&model, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product by id: %w", err)
	}

	return model.ToDomain(), nil
}

func (r *ProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	var model ProductModel

	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Images").
		Preload("Variants").
		Preload("Tags").
		Where("slug = ?", slug).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product by slug: %w", err)
	}

	return model.ToDomain(), nil
}

func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	model := &ProductModel{}
	model.FromDomain(product)

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		if IsDuplicateError(result.Error) {
			return ErrDuplicateSKU
		}
		return fmt.Errorf("failed to update product: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}

	product.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *ProductRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&ProductModel{}).
		Where("id = ?", id).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("failed to update fields: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}

	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&ProductModel{}, id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete product: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}

	return nil
}

func (r *ProductRepository) List(ctx context.Context, filters *application.ProductFilters) ([]*domain.Product, int64, error) {
	var models []*ProductModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ProductModel{})

	// Apply filters
	if filters.CategoryID > 0 {
		query = query.Where("category_id = ?", filters.CategoryID)
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.IsFeatured != nil {
		query = query.Where("is_featured = ?", *filters.IsFeatured)
	}

	if filters.InStock {
		query = query.Where("stock_quantity > 0")
	}

	if filters.MinPrice > 0 {
		query = query.Where("price >= ?", filters.MinPrice)
	}

	if filters.MaxPrice > 0 {
		query = query.Where("price <= ?", filters.MaxPrice)
	}

	if filters.Search != "" {
		search := "%" + filters.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", search, search)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Apply sorting
	switch filters.Sort {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	case "name":
		query = query.Order("name ASC")
	case "newest":
		query = query.Order("created_at DESC")
	default:
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (filters.Page - 1) * filters.PageSize
	query = query.Offset(offset).Limit(filters.PageSize)

	// Preload relationships
	query = query.
		Preload("Category").
		Preload("Images").
		Preload("Tags")

	// Execute query
	err := query.Find(&models).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}

	products := make([]*domain.Product, len(models))
	for i, model := range models {
		products[i] = model.ToDomain()
	}

	return products, total, nil
}

func (r *ProductRepository) ExistsSKU(ctx context.Context, sku string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&ProductModel{}).
		Where("sku = ?", sku)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check SKU exists: %w", err)
	}

	return count > 0, nil
}

func (r *ProductRepository) ExistsSlug(ctx context.Context, slug string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&ProductModel{}).
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

func (r *ProductRepository) UpdateStock(ctx context.Context, productID uint, quantity int) error {
	result := r.db.WithContext(ctx).
		Model(&ProductModel{}).
		Where("id = ?", productID).
		Update("stock_quantity", gorm.Expr("stock_quantity + ?", quantity))

	if result.Error != nil {
		return fmt.Errorf("failed to update stock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}

	return nil
}
