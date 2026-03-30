package postgres

import (
	"time"

	"gorm.io/gorm"
)

// ReviewModel represents a product review in the database
type ReviewModel struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null;index" json:"user_id"`
	ProductID uint `gorm:"not null;index" json:"product_id"`
	OrderID   *uint `gorm:"index" json:"order_id,omitempty"`

	Rating  int    `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Title   string `gorm:"size:200" json:"title,omitempty"`
	Content string `gorm:"type:text;not null" json:"content"`

	// Images as JSON array of URLs
	Images string `gorm:"type:jsonb" json:"images,omitempty"` // ["url1", "url2"]

	// Moderation: pending, approved, rejected
	Status         string `gorm:"size:20;default:pending;index" json:"status"`
	RejectedReason string `gorm:"type:text" json:"rejected_reason,omitempty"`

	// Helpful votes
	HelpfulCount int `gorm:"default:0" json:"helpful_count"`

	// Admin response
	AdminReply   string     `gorm:"type:text" json:"admin_reply,omitempty"`
	AdminReplyAt *time.Time `json:"admin_reply_at,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Product *ProductModel `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (ReviewModel) TableName() string {
	return "reviews"
}

// Review status constants
const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
)
