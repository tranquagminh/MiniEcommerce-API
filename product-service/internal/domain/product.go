package domain

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID               uint
	Name             string
	Slug             string
	Description      string
	ShortDescription string
	CategoryID       uint
	Category         *Category `gorm:"foreignKey:CategoryID"`
	Price            float64
	CompareAtPrice   float64 // Original price (for discount display)
	CostPerItem      float64 // Cost for profit calculation

	// Inventory
	SKU               string
	Barcode           string
	TrackInventory    bool
	StockQuantity     int
	LowStockThreshold int

	// Physical attributes
	Weight float64 // in kg
	Length float64 // in cm
	Width  float64
	Height float64

	// Status
	Status     string // draft, active, archived
	IsFeatured bool

	// SEO
	MetaTitle       string
	MetaDescription string
	MetaKeywords    string

	// Relationships
	Images   []ProductImage   `gorm:"foreignKey:ProductID"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID"`
	Tags     []Tag            `gorm:"many2many:product_tags"`

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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
