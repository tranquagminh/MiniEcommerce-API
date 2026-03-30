package postgres

import (
	"time"

	"gorm.io/gorm"
)

// OrderModel represents an order in the database
type OrderModel struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	UserID      uint   `gorm:"not null;index" json:"user_id"`
	OrderNumber string `gorm:"size:50;not null;uniqueIndex" json:"order_number"`

	// Shipping Info (snapshot at order time)
	ShippingFullName string `gorm:"size:100;not null" json:"shipping_full_name"`
	ShippingPhone    string `gorm:"size:20;not null" json:"shipping_phone"`
	ShippingProvince string `gorm:"size:100;not null" json:"shipping_province"`
	ShippingDistrict string `gorm:"size:100;not null" json:"shipping_district"`
	ShippingWard     string `gorm:"size:100;not null" json:"shipping_ward"`
	ShippingAddress  string `gorm:"size:255;not null" json:"shipping_address"`

	// Order Status: pending, confirmed, processing, shipped, delivered, cancelled, refunded
	Status string `gorm:"size:20;default:pending;index" json:"status"`

	// Payment
	PaymentMethod string     `gorm:"size:20;not null" json:"payment_method"`        // cod, bank, card
	PaymentStatus string     `gorm:"size:20;default:pending" json:"payment_status"` // pending, paid, failed, refunded
	PaidAt        *time.Time `json:"paid_at,omitempty"`

	// Pricing
	Subtotal       float64 `gorm:"type:decimal(15,2);not null" json:"subtotal"`
	ShippingFee    float64 `gorm:"type:decimal(10,2);default:0" json:"shipping_fee"`
	DiscountAmount float64 `gorm:"type:decimal(10,2);default:0" json:"discount_amount"`
	CouponCode     string  `gorm:"size:50" json:"coupon_code,omitempty"`
	Total          float64 `gorm:"type:decimal(15,2);not null" json:"total"`

	// Additional
	Note            string     `gorm:"type:text" json:"note,omitempty"`
	CancelledReason string     `gorm:"type:text" json:"cancelled_reason,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	ShippedAt       *time.Time `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time `json:"delivered_at,omitempty"`

	// Relationships
	Items []OrderItemModel `gorm:"foreignKey:OrderID" json:"items,omitempty"`

	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (OrderModel) TableName() string {
	return "orders"
}

// OrderItemModel represents an item in an order
type OrderItemModel struct {
	ID        uint  `gorm:"primaryKey" json:"id"`
	OrderID   uint  `gorm:"not null;index" json:"order_id"`
	ProductID uint  `gorm:"not null;index" json:"product_id"`
	VariantID *uint `json:"variant_id,omitempty"`

	// Snapshot at order time
	ProductName    string `gorm:"size:200;not null" json:"product_name"`
	ProductImage   string `gorm:"size:500" json:"product_image,omitempty"`
	VariantOptions string `gorm:"type:text" json:"variant_options,omitempty"`
	SKU            string `gorm:"size:100" json:"sku,omitempty"`

	Quantity   int     `gorm:"not null" json:"quantity"`
	UnitPrice  float64 `gorm:"type:decimal(10,2);not null" json:"unit_price"`
	TotalPrice float64 `gorm:"type:decimal(10,2);not null" json:"total_price"`

	CreatedAt time.Time `json:"created_at"`
}

func (OrderItemModel) TableName() string {
	return "order_items"
}

// Order status constants
const (
	OrderStatusPending    = "pending"
	OrderStatusConfirmed  = "confirmed"
	OrderStatusProcessing = "processing"
	OrderStatusShipped    = "shipped"
	OrderStatusDelivered  = "delivered"
	OrderStatusCancelled  = "cancelled"
	OrderStatusRefunded   = "refunded"
)

// Payment status constants
const (
	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"
)

// Payment method constants
const (
	PaymentMethodCOD  = "cod"
	PaymentMethodBank = "bank"
	PaymentMethodCard = "card"
)
