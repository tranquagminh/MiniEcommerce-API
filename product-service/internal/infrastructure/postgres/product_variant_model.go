package postgres

import (
	"encoding/json"
	"product-service/internal/domain"
	"time"
)

type ProductVariantModel struct {
	ID             uint            `gorm:"primaryKey"`
	ProductID      uint            `gorm:"not null;index" json:"product_id"`
	SKU            string          `gorm:"size:100;uniqueIndex;not null" json:"sku"`
	Barcode        string          `gorm:"size:100" json:"barcode,omitempty"`
	Options        json.RawMessage `gorm:"type:jsonb;not null" json:"options"`
	Price          float64         `gorm:"type:decimal(10,2);not null" json:"price"`
	CompareAtPrice float64         `gorm:"type:decimal(10,2)" json:"compare_at_price,omitempty"`
	CostPerItem    float64         `gorm:"type:decimal(10,2)" json:"cost_per_item,omitempty"`
	StockQuantity  int             `gorm:"default:0" json:"stock_quantity"`
	Weight         float64         `gorm:"type:decimal(8,2)" json:"weight,omitempty"`
	ImageURL       string          `gorm:"size:500" json:"image_url,omitempty"`
	IsAvailable    bool            `gorm:"default:true" json:"is_available"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (ProductVariantModel) TableName() string {
	return "product_variants"
}

func (m *ProductVariantModel) ToDomain() *domain.ProductVariant {
	return &domain.ProductVariant{
		ID:             m.ID,
		ProductID:      m.ProductID,
		SKU:            m.SKU,
		Barcode:        m.Barcode,
		Options:        m.Options,
		Price:          m.Price,
		CompareAtPrice: m.CompareAtPrice,
		CostPerItem:    m.CostPerItem,
		StockQuantity:  m.StockQuantity,
		Weight:         m.Weight,
		ImageURL:       m.ImageURL,
		IsAvailable:    m.IsAvailable,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func (m *ProductVariantModel) FromDomain(v *domain.ProductVariant) {
	m.ID = v.ID
	m.ProductID = v.ProductID
	m.SKU = v.SKU
	m.Barcode = v.Barcode
	m.Options = v.Options
	m.Price = v.Price
	m.CompareAtPrice = v.CompareAtPrice
	m.CostPerItem = v.CostPerItem
	m.StockQuantity = v.StockQuantity
	m.Weight = v.Weight
	m.ImageURL = v.ImageURL
	m.IsAvailable = v.IsAvailable
	m.CreatedAt = v.CreatedAt
	m.UpdatedAt = v.UpdatedAt
}
