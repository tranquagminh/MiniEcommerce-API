package domain

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID           uint
	Name         string
	Slug         string
	Description  string
	ParentID     *uint
	Parent       *Category   `gorm:"foreignKey:ParentID"`
	Children     []Category  `gorm:"foreignKey:ParentID"`
	ImageURL     string
	IsActive     bool
	DisplayOrder int
	
	// Timestamps
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (c *Category) IsRootCategory() bool {
	return c.ParentID == nil
}

func (c *Category) HasChildren() bool {
	return len(c.Children) > 0
}