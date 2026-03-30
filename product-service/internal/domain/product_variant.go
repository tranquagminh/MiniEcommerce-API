package domain

import (
	"encoding/json"
	"time"
)

type ProductVariant struct {
	ID        uint   `json:"id"`
	ProductID uint   `json:"product_id"`
	SKU       string `json:"sku"`
	Barcode   string `json:"barcode"`

	// Options stored as JSON
	// Example: {"color": "Red", "size": "L"}
	Options json.RawMessage `json:"options" gorm:"type:jsonb"`

	// Pricing & inventory
	Price          float64 `json:"price"`
	CompareAtPrice float64 `json:"compare_at_price"`
	CostPerItem    float64 `json:"cost_per_item"`
	StockQuantity  int     `json:"stock_quantity"`

	// Physical
	Weight float64 `json:"weight"`

	// Image
	ImageURL string `json:"image_url"`

	// Status
	IsAvailable bool `json:"is_available"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (v *ProductVariant) IsInStock() bool {
	return v.IsAvailable && v.StockQuantity > 0
}
