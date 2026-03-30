package domain

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID           uint       `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	Description  string     `json:"description"`
	ParentID     *uint      `json:"parent_id"`
	Parent       *Category  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children     []Category `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	ImageURL     string     `json:"image_url"`
	IsActive     bool       `json:"is_active"`
	DisplayOrder int        `json:"display_order"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (c *Category) IsRootCategory() bool {
	return c.ParentID == nil
}

func (c *Category) HasChildren() bool {
	return len(c.Children) > 0
}
