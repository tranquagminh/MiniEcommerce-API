package application

import (
	"context"
	"fmt"
	"product-service/internal/domain"
	"strings"

	"github.com/gosimple/slug"
)

type ProductService struct {
	repo      ProductRepository
	txManager TransactionManager
	cache     ProductCache
}

func NewProductService(repo ProductRepository, txManager TransactionManager, cache ProductCache) *ProductService {
	return &ProductService{
		repo:      repo,
		txManager: txManager,
		cache:     cache,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, product *domain.Product) error {
	// Validate
	product.Name = strings.TrimSpace(product.Name)
	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}

	// Generate slug if not provided
	if product.Slug == "" {
		product.Slug = slug.Make(product.Name)
	} else {
		product.Slug = slug.Make(product.Slug)
	}

	// Check if slug exists
	exists, err := s.repo.ExistsSlug(ctx, product.Slug, 0)
	if err != nil {
		return fmt.Errorf("failed to check slug: %w", err)
	}
	if exists {
		return fmt.Errorf("slug already exists")
	}

	// Check SKU if provided
	if product.SKU != "" {
		exists, err := s.repo.ExistsSKU(ctx, product.SKU, 0)
		if err != nil {
			return fmt.Errorf("failed to check SKU: %w", err)
		}
		if exists {
			return fmt.Errorf("SKU already exists")
		}
	}

	// Set default status
	if product.Status == "" {
		product.Status = "draft"
	}

	// Create product
	if err := s.repo.Create(ctx, product); err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

func (s *ProductService) GetProduct(ctx context.Context, id uint) (*domain.Product, error) {
	// Try cache first
	if s.cache != nil {
		product, err := s.cache.Get(ctx, id)
		if err == nil {
			return product, nil
		}
	}

	// Get from database
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		_ = s.cache.Set(ctx, product)
		_ = s.cache.SetBySlug(ctx, product.Slug, product)
	}

	return product, nil
}

func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	// Try cache first
	if s.cache != nil {
		product, err := s.cache.GetBySlug(ctx, slug)
		if err == nil {
			return product, nil
		}
	}

	// Get from database
	product, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		_ = s.cache.Set(ctx, product)
		_ = s.cache.SetBySlug(ctx, product.Slug, product)
	}

	return product, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, product *domain.Product) error {
	// Validate
	product.Name = strings.TrimSpace(product.Name)
	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}

	// Check if slug changed and exists
	if product.Slug != "" {
		product.Slug = slug.Make(product.Slug)
		exists, err := s.repo.ExistsSlug(ctx, product.Slug, product.ID)
		if err != nil {
			return fmt.Errorf("failed to check slug: %w", err)
		}
		if exists {
			return fmt.Errorf("slug already exists")
		}
	}

	// Check SKU if changed
	if product.SKU != "" {
		exists, err := s.repo.ExistsSKU(ctx, product.SKU, product.ID)
		if err != nil {
			return fmt.Errorf("failed to check SKU: %w", err)
		}
		if exists {
			return fmt.Errorf("SKU already exists")
		}
	}

	// Update product
	if err := s.repo.Update(ctx, product); err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.Delete(ctx, product.ID)
		_ = s.cache.DeleteBySlug(ctx, product.Slug)
	}

	return nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, id uint) error {
	// Get product first to get slug for cache invalidation
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete product
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.Delete(ctx, id)
		_ = s.cache.DeleteBySlug(ctx, product.Slug)
	}

	return nil
}

func (s *ProductService) ListProducts(ctx context.Context, filters *ProductFilters) ([]*domain.Product, int64, error) {
	// Validate pagination
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}
	if filters.PageSize > 100 {
		filters.PageSize = 100
	}

	return s.repo.List(ctx, filters)
}

func (s *ProductService) AddProductImage(ctx context.Context, productID uint, image *domain.ProductImage) error {
	if image.ImageURL == "" {
		return fmt.Errorf("image_url is required")
	}
	return s.repo.AddImage(ctx, productID, image)
}

func (s *ProductService) RemoveProductImage(ctx context.Context, imageID uint) error {
	return s.repo.RemoveImage(ctx, imageID)
}

func (s *ProductService) UpdateStock(ctx context.Context, productID uint, quantity int) error {
	if err := s.repo.UpdateStock(ctx, productID, quantity); err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.Delete(ctx, productID)
	}

	return nil
}

func (s *ProductService) UpdateStatus(ctx context.Context, productID uint, status string) error {
	// Validate status
	validStatuses := map[string]bool{
		"draft":    true,
		"active":   true,
		"archived": true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	fields := map[string]interface{}{
		"status": status,
	}

	if err := s.repo.UpdateFields(ctx, productID, fields); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.Delete(ctx, productID)
	}

	return nil
}
