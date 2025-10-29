package application

import (
	"context"
	"fmt"
	"product-service/internal/domain"
	"strings"

	"github.com/gosimple/slug"
)

type CategoryService struct {
	repo      CategoryRepository
	txManager TransactionManager
	cache     CategoryCache
}

func NewCategoryService(repo CategoryRepository, txManager TransactionManager, cache CategoryCache) *CategoryService {
	return &CategoryService{
		repo:      repo,
		txManager: txManager,
		cache:     cache,
	}
}

func (s *CategoryService) CreateCategory(ctx context.Context, category *domain.Category) error {
	// Validate
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return fmt.Errorf("category name is required")
	}

	// Generate slug if not provided
	if category.Slug == "" {
		category.Slug = slug.Make(category.Name)
	} else {
		category.Slug = slug.Make(category.Slug)
	}

	// Check if slug exists
	exists, err := s.repo.ExistsSlug(ctx, category.Slug, 0)
	if err != nil {
		return fmt.Errorf("failed to check slug: %w", err)
	}
	if exists {
		return fmt.Errorf("slug already exists")
	}

	// Validate parent exists if provided
	if category.ParentID != nil {
		_, err := s.repo.GetByID(ctx, *category.ParentID)
		if err != nil {
			return fmt.Errorf("parent category not found")
		}
	}

	// Create category
	if err := s.repo.Create(ctx, category); err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	// Invalidate list cache
	if s.cache != nil {
		_ = s.cache.DeleteList(ctx)
	}

	return nil
}

func (s *CategoryService) GetCategory(ctx context.Context, id uint) (*domain.Category, error) {
	// Try cache first
	if s.cache != nil {
		category, err := s.cache.Get(ctx, id)
		if err == nil {
			return category, nil
		}
	}

	// Get from database
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		_ = s.cache.Set(ctx, category)
	}

	return category, nil
}

func (s *CategoryService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, category *domain.Category) error {
	// Validate
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return fmt.Errorf("category name is required")
	}

	// Check if slug changed and exists
	if category.Slug != "" {
		category.Slug = slug.Make(category.Slug)
		exists, err := s.repo.ExistsSlug(ctx, category.Slug, category.ID)
		if err != nil {
			return fmt.Errorf("failed to check slug: %w", err)
		}
		if exists {
			return fmt.Errorf("slug already exists")
		}
	}

	// Validate parent exists if provided
	if category.ParentID != nil {
		// Can't be parent of itself
		if *category.ParentID == category.ID {
			return fmt.Errorf("category cannot be parent of itself")
		}

		_, err := s.repo.GetByID(ctx, *category.ParentID)
		if err != nil {
			return fmt.Errorf("parent category not found")
		}
	}

	// Update category
	if err := s.repo.Update(ctx, category); err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.Delete(ctx, category.ID)
		_ = s.cache.DeleteList(ctx)
	}

	return nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id uint) error {
	// Check if category has children
	children, err := s.repo.GetChildren(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check children: %w", err)
	}
	if len(children) > 0 {
		return fmt.Errorf("cannot delete category with children")
	}

	// Delete category
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.Delete(ctx, id)
		_ = s.cache.DeleteList(ctx)
	}

	return nil
}

func (s *CategoryService) ListCategories(ctx context.Context, activeOnly bool) ([]*domain.Category, error) {
	// Try cache first (only for active categories)
	if s.cache != nil && activeOnly {
		categories, err := s.cache.GetList(ctx)
		if err == nil {
			return categories, nil
		}
	}

	// Get from database
	categories, err := s.repo.List(ctx, activeOnly)
	if err != nil {
		return nil, err
	}

	// Update cache (only for active categories)
	if s.cache != nil && activeOnly {
		_ = s.cache.SetList(ctx, categories)
	}

	return categories, nil
}

func (s *CategoryService) GetRootCategories(ctx context.Context, activeOnly bool) ([]*domain.Category, error) {
	return s.repo.GetRootCategories(ctx, activeOnly)
}

func (s *CategoryService) GetChildren(ctx context.Context, parentID uint) ([]*domain.Category, error) {
	return s.repo.GetChildren(ctx, parentID)
}
