package postgres

import (
	"product-service/internal/domain"
	"time"

	"gorm.io/gorm"
)

type ProductModel struct {
	ID                uint           `gorm:"primaryKey"`
	Name              string         `gorm:"size:200;not null" json:"name"`
	Slug              string         `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	Description       string         `gorm:"type:text" json:"description,omitempty"`
	ShortDescription  string         `gorm:"size:500" json:"short_description,omitempty"`
	CategoryID        uint           `gorm:"not null" json:"category_id"`
	
	// Pricing
	Price             float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	CompareAtPrice    float64        `gorm:"type:decimal(10,2)" json:"compare_at_price,omitempty"`
	CostPerItem       float64        `gorm:"type:decimal(10,2)" json:"cost_per_item,omitempty"`
	
	// Inventory
	SKU               string         `gorm:"size:100;uniqueIndex" json:"sku,omitempty"`
	Barcode           string         `gorm:"size:100" json:"barcode,omitempty"`
	TrackInventory    bool           `gorm:"default:true" json:"track_inventory"`
	StockQuantity     int            `gorm:"default:0" json:"stock_quantity"`
	LowStockThreshold int            `gorm:"default:10" json:"low_stock_threshold"`
	
	// Physical attributes
	Weight            float64        `gorm:"type:decimal(8,2)" json:"weight,omitempty"` // kg
	Length            float64        `gorm:"type:decimal(8,2)" json:"length,omitempty"` // cm
	Width             float64        `gorm:"type:decimal(8,2)" json:"width,omitempty"`
	Height            float64        `gorm:"type:decimal(8,2)" json:"height,omitempty"`
	
	// Status
	Status            string         `gorm:"size:20;default:'draft'" json:"status"` // draft, active, archived
	IsFeatured        bool           `gorm:"default:false" json:"is_featured"`
	
	// SEO
	MetaTitle         string         `gorm:"size:200" json:"meta_title,omitempty"`
	MetaDescription   string         `gorm:"type:text" json:"meta_description,omitempty"`
	MetaKeywords      string         `gorm:"type:text" json:"meta_keywords,omitempty"`
	
	// Timestamps
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships (not stored, just for loading)
	Category          *CategoryModel       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Images            []ProductImageModel  `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	Variants          []ProductVariantModel `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
	Tags              []TagModel           `gorm:"many2many:product_tags" json:"tags,omitempty"`
}

func (ProductModel) TableName() string {
	return "products"
}

func (m *ProductModel) ToDomain() *domain.Product {
	product := &domain.Product{
		ID:                m.ID,
		Name:              m.Name,
		Slug:              m.Slug,
		Description:       m.Description,
		ShortDescription:  m.ShortDescription,
		CategoryID:        m.CategoryID,
		Price:             m.Price,
		CompareAtPrice:    m.CompareAtPrice,
		CostPerItem:       m.CostPerItem,
		SKU:               m.SKU,
		Barcode:           m.Barcode,
		TrackInventory:    m.TrackInventory,
		StockQuantity:     m.StockQuantity,
		LowStockThreshold: m.LowStockThreshold,
		Weight:            m.Weight,
		Length:            m.Length,
		Width:             m.Width,
		Height:            m.Height,
		Status:            m.Status,
		IsFeatured:        m.IsFeatured,
		MetaTitle:         m.MetaTitle,
		MetaDescription:   m.MetaDescription,
		MetaKeywords:      m.MetaKeywords,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		DeletedAt:         m.DeletedAt,
	}
	
	// Convert category if loaded
	if m.Category != nil {
		product.Category = m.Category.ToDomain()
	}
	
	// Convert images if loaded
	if len(m.Images) > 0 {
		product.Images = make([]domain.ProductImage, len(m.Images))
		for i, img := range m.Images {
			product.Images[i] = *img.ToDomain()
		}
	}
	
	// Convert variants if loaded
	if len(m.Variants) > 0 {
		product.Variants = make([]domain.ProductVariant, len(m.Variants))
		for i, v := range m.Variants {
			product.Variants[i] = *v.ToDomain()
		}
	}
	
	// Convert tags if loaded
	if len(m.Tags) > 0 {
		product.Tags = make([]domain.Tag, len(m.Tags))
		for i, t := range m.Tags {
			product.Tags[i] = *t.ToDomain()
		}
	}
	
	return product
}

func (m *ProductModel) FromDomain(p *domain.Product) {
	m.ID = p.ID
	m.Name = p.Name
	m.Slug = p.Slug
	m.Description = p.Description
	m.ShortDescription = p.ShortDescription
	m.CategoryID = p.CategoryID
	m.Price = p.Price
	m.CompareAtPrice = p.CompareAtPrice
	m.CostPerItem = p.CostPerItem
	m.SKU = p.SKU
	m.Barcode = p.Barcode
	m.TrackInventory = p.TrackInventory
	m.StockQuantity = p.StockQuantity
	m.LowStockThreshold = p.LowStockThreshold
	m.Weight = p.Weight
	m.Length = p.Length
	m.Width = p.Width
	m.Height = p.Height
	m.Status = p.Status
	m.IsFeatured = p.IsFeatured
	m.MetaTitle = p.MetaTitle
	m.MetaDescription = p.MetaDescription
	m.MetaKeywords = p.MetaKeywords
	m.CreatedAt = p.CreatedAt
	m.UpdatedAt = p.UpdatedAt
	m.DeletedAt = p.DeletedAt
}