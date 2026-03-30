package domain

import (
	"time"
)

type ProductImage struct {
	ID           uint      `json:"id"`
	ProductID    uint      `json:"product_id"`
	ImageURL     string    `json:"image_url"`
	AltText      string    `json:"alt_text"`
	DisplayOrder int       `json:"display_order"`
	IsPrimary    bool      `json:"is_primary"`
	CreatedAt    time.Time `json:"created_at"`
}
