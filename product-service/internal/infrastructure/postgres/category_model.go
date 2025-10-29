package postgres

import (
	"product-service/internal/domain"
	"time"

	"gorm.io/gorm"
)

type CategoryModel struct {
	ID           uint           `gorm:"primaryKey"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	Slug         string         `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Description  string         `gorm:"type:text" json:"description,omitempty"`
	ParentID     *uint          `gorm:"index" json:"parent_id,omitempty"`
	ImageURL     string         `gorm:"size:500" json:"image_url,omitempty"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	DisplayOrder int            `gorm:"default:0" json:"display_order"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Parent   *CategoryModel  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []CategoryModel `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (CategoryModel) TableName() string {
	return "categories"
}

func (m *CategoryModel) ToDomain() *domain.Category {
	category := &domain.Category{
		ID:           m.ID,
		Name:         m.Name,
		Slug:         m.Slug,
		Description:  m.Description,
		ParentID:     m.ParentID,
		ImageURL:     m.ImageURL,
		IsActive:     m.IsActive,
		DisplayOrder: m.DisplayOrder,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}

	// Convert parent if loaded
	if m.Parent != nil {
		category.Parent = m.Parent.ToDomain()
	}

	// Convert children if loaded
	if len(m.Children) > 0 {
		category.Children = make([]domain.Category, len(m.Children))
		for i, child := range m.Children {
			category.Children[i] = *child.ToDomain()
		}
	}

	return category
}

func (m *CategoryModel) FromDomain(c *domain.Category) {
	m.ID = c.ID
	m.Name = c.Name
	m.Slug = c.Slug
	m.Description = c.Description
	m.ParentID = c.ParentID
	m.ImageURL = c.ImageURL
	m.IsActive = c.IsActive
	m.DisplayOrder = c.DisplayOrder
	m.CreatedAt = c.CreatedAt
	m.UpdatedAt = c.UpdatedAt
	m.DeletedAt = c.DeletedAt
}
