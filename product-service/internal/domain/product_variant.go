package domain

import (
	"encoding/json"
	"time"
)

type ProductVariant struct {
	ID        uint
	ProductID uint
	SKU       string
	Barcode   string

	// Options stored as JSON
	// Example: {"color": "Red", "size": "L"}
	Options json.RawMessage `gorm:"type:jsonb"`

	// Pricing & inventory
	Price          float64
	CompareAtPrice float64
	CostPerItem    float64
	StockQuantity  int

	// Physical
	Weight float64

	// Image
	ImageURL string

	// Status
	IsAvailable bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (v *ProductVariant) IsInStock() bool {
	return v.IsAvailable && v.StockQuantity > 0
}
