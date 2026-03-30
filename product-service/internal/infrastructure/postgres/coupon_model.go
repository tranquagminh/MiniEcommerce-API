package postgres

import (
	"time"

	"gorm.io/gorm"
)

// CouponModel represents a discount coupon
type CouponModel struct {
	ID          uint   `gorm:"primaryKey"`
	Code        string `gorm:"size:50;not null;uniqueIndex" json:"code"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// Discount Type: percentage, fixed
	DiscountType  string  `gorm:"size:20;not null" json:"discount_type"`
	DiscountValue float64 `gorm:"type:decimal(10,2);not null" json:"discount_value"`
	MaxDiscount   *float64 `gorm:"type:decimal(10,2)" json:"max_discount,omitempty"` // Max discount for percentage type

	// Conditions
	MinOrderValue  float64 `gorm:"type:decimal(10,2);default:0" json:"min_order_value"`
	MaxUses        *int    `json:"max_uses,omitempty"`         // NULL = unlimited
	MaxUsesPerUser int     `gorm:"default:1" json:"max_uses_per_user"`
	UsedCount      int     `gorm:"default:0" json:"used_count"`

	// Validity
	StartsAt  time.Time `gorm:"not null" json:"starts_at"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	IsActive  bool      `gorm:"default:true;index" json:"is_active"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CouponModel) TableName() string {
	return "coupons"
}

// CouponUsageModel tracks coupon usage
type CouponUsageModel struct {
	ID             uint      `gorm:"primaryKey"`
	CouponID       uint      `gorm:"not null;index" json:"coupon_id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	OrderID        uint      `gorm:"not null" json:"order_id"`
	DiscountAmount float64   `gorm:"type:decimal(10,2);not null" json:"discount_amount"`
	UsedAt         time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"used_at"`

	// Relationships
	Coupon *CouponModel `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
}

func (CouponUsageModel) TableName() string {
	return "coupon_usage"
}

// Discount type constants
const (
	DiscountTypePercentage = "percentage"
	DiscountTypeFixed      = "fixed"
)

// IsValid checks if the coupon is currently valid
func (c *CouponModel) IsValid() bool {
	now := time.Now()
	if !c.IsActive {
		return false
	}
	if now.Before(c.StartsAt) || now.After(c.ExpiresAt) {
		return false
	}
	if c.MaxUses != nil && c.UsedCount >= *c.MaxUses {
		return false
	}
	return true
}

// CalculateDiscount calculates the discount for a given order amount
func (c *CouponModel) CalculateDiscount(orderAmount float64) float64 {
	if orderAmount < c.MinOrderValue {
		return 0
	}

	var discount float64
	if c.DiscountType == DiscountTypePercentage {
		discount = orderAmount * (c.DiscountValue / 100)
		if c.MaxDiscount != nil && discount > *c.MaxDiscount {
			discount = *c.MaxDiscount
		}
	} else {
		discount = c.DiscountValue
	}

	// Discount cannot exceed order amount
	if discount > orderAmount {
		discount = orderAmount
	}

	return discount
}
