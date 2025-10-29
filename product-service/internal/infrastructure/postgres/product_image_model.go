package postgres

import (
	"product-service/internal/domain"
	"time"
)

type ProductImageModel struct {
	ID           uint      `gorm:"primaryKey"`
	ProductID    uint      `gorm:"not null;index" json:"product_id"`
	ImageURL     string    `gorm:"size:500;not null" json:"image_url"`
	AltText      string    `gorm:"size:200" json:"alt_text,omitempty"`
	DisplayOrder int       `gorm:"default:0" json:"display_order"`
	IsPrimary    bool      `gorm:"default:false" json:"is_primary"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ProductImageModel) TableName() string {
	return "product_images"
}

func (m *ProductImageModel) ToDomain() *domain.ProductImage {
	return &domain.ProductImage{
		ID:           m.ID,
		ProductID:    m.ProductID,
		ImageURL:     m.ImageURL,
		AltText:      m.AltText,
		DisplayOrder: m.DisplayOrder,
		IsPrimary:    m.IsPrimary,
		CreatedAt:    m.CreatedAt,
	}
}

func (m *ProductImageModel) FromDomain(img *domain.ProductImage) {
	m.ID = img.ID
	m.ProductID = img.ProductID
	m.ImageURL = img.ImageURL
	m.AltText = img.AltText
	m.DisplayOrder = img.DisplayOrder
	m.IsPrimary = img.IsPrimary
	m.CreatedAt = img.CreatedAt
}
