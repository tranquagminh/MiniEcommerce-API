package domain

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID               uint      `json:"id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Description      string    `json:"description"`
	ShortDescription string    `json:"short_description"`
	CategoryID       uint      `json:"category_id"`
	Category         *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Price            float64   `json:"price"`
	CompareAtPrice   float64   `json:"compare_at_price"` // Original price (for discount display)
	CostPerItem      float64   `json:"cost_per_item"`    // Cost for profit calculation

	// Inventory
	SKU               string `json:"sku"`
	Barcode           string `json:"barcode"`
	TrackInventory    bool   `json:"track_inventory"`
	StockQuantity     int    `json:"stock_quantity"`
	LowStockThreshold int    `json:"low_stock_threshold"`

	// Physical attributes
	Weight float64 `json:"weight"` // in kg
	Length float64 `json:"length"` // in cm
	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	// Status
	Status     string `json:"status"` // draft, active, archived
	IsFeatured bool   `json:"is_featured"`

	// SEO
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	MetaKeywords    string `json:"meta_keywords"`

	// Relationships
	Images   []ProductImage   `json:"images" gorm:"foreignKey:ProductID"`
	Variants []ProductVariant `json:"variants" gorm:"foreignKey:ProductID"`
	Tags     []Tag            `json:"tags" gorm:"many2many:product_tags"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (p *Product) IsAvailable() bool {
	return p.Status == "active" && (!p.TrackInventory || p.StockQuantity > 0)
}

func (p *Product) IsLowStock() bool {
	return p.TrackInventory && p.StockQuantity <= p.LowStockThreshold
}

func (p *Product) DiscountPercentage() float64 {
	if p.CompareAtPrice > 0 && p.Price < p.CompareAtPrice {
		return ((p.CompareAtPrice - p.Price) / p.CompareAtPrice) * 100
	}
	return 0
}
