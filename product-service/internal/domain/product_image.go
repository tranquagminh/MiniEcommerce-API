package domain

import (
	"time"
)

type ProductImage struct {
	ID           uint
	ProductID    uint
	ImageURL     string
	AltText      string
	DisplayOrder int
	IsPrimary    bool
	CreatedAt    time.Time
}
