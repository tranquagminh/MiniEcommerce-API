package application

import (
	"context"
	"product-service/internal/domain"

	"gorm.io/gorm"
)

// ProductRepository interface
type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id uint) (*domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters *ProductFilters) ([]*domain.Product, int64, error)
	ExistsSKU(ctx context.Context, sku string, excludeID uint) (bool, error)
	ExistsSlug(ctx context.Context, slug string, excludeID uint) (bool, error)
	UpdateStock(ctx context.Context, productID uint, quantity int) error
	WithTx(tx *gorm.DB) ProductRepository
}

// CategoryRepository interface
type CategoryRepository interface {
	Create(ctx context.Context, category *domain.Category) error
	GetByID(ctx context.Context, id uint) (*domain.Category, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Category, error)
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, activeOnly bool) ([]*domain.Category, error)
	GetRootCategories(ctx context.Context, activeOnly bool) ([]*domain.Category, error)
	GetChildren(ctx context.Context, parentID uint) ([]*domain.Category, error)
	ExistsSlug(ctx context.Context, slug string, excludeID uint) (bool, error)
	WithTx(tx *gorm.DB) CategoryRepository
}

// TransactionManager interface
type TransactionManager interface {
	ExecuteInTx(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// ProductCache interface
type ProductCache interface {
	Set(ctx context.Context, product *domain.Product) error
	Get(ctx context.Context, productID uint) (*domain.Product, error)
	Delete(ctx context.Context, productID uint) error
	SetBySlug(ctx context.Context, slug string, product *domain.Product) error
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	DeleteBySlug(ctx context.Context, slug string) error
}

// CategoryCache interface
type CategoryCache interface {
	Set(ctx context.Context, category *domain.Category) error
	Get(ctx context.Context, categoryID uint) (*domain.Category, error)
	Delete(ctx context.Context, categoryID uint) error
	SetList(ctx context.Context, categories []*domain.Category) error
	GetList(ctx context.Context) ([]*domain.Category, error)
	DeleteList(ctx context.Context) error
}

// ProductFilters for listing products
type ProductFilters struct {
	CategoryID uint
	Status     string // draft, active, archived
	IsFeatured *bool
	InStock    bool
	MinPrice   float64
	MaxPrice   float64
	Search     string
	Sort       string // price_asc, price_desc, name, newest
	Page       int
	PageSize   int
}
