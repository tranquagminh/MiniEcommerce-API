package postgres

import (
	"time"

	"gorm.io/gorm"
)

// ProductQAModel represents a Q&A entry for a product
type ProductQAModel struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null;index" json:"user_id"`
	ProductID uint `gorm:"not null;index" json:"product_id"`

	Question string `gorm:"type:text;not null" json:"question"`
	Answer   string `gorm:"type:text" json:"answer,omitempty"`

	// Status: pending, answered
	Status string `gorm:"size:20;default:pending;index" json:"status"`

	AnsweredBy *uint      `json:"answered_by,omitempty"` // Admin user ID
	AnsweredAt *time.Time `json:"answered_at,omitempty"`

	// Helpful votes
	HelpfulCount int `gorm:"default:0" json:"helpful_count"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Product *ProductModel `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (ProductQAModel) TableName() string {
	return "product_qa"
}

// QA status constants
const (
	QAStatusPending  = "pending"
	QAStatusAnswered = "answered"
)
