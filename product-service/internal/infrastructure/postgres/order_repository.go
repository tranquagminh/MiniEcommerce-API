package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create creates a new order with items
func (r *OrderRepository) Create(ctx context.Context, order *OrderModel) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// GetByID retrieves an order by ID with items
func (r *OrderRepository) GetByID(ctx context.Context, id uint) (*OrderModel, error) {
	var order OrderModel
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOrderNumber retrieves an order by order number
func (r *OrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*OrderModel, error) {
	var order OrderModel
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("order_number = ?", orderNumber).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByUserID retrieves all orders for a user
func (r *OrderRepository) GetByUserID(ctx context.Context, userID uint, page, limit int) ([]OrderModel, int64, error) {
	var orders []OrderModel
	var total int64

	query := r.db.WithContext(ctx).Model(&OrderModel{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Preload("Items").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// OrderFilter for filtering orders
type OrderFilter struct {
	Status        string
	PaymentStatus string
	UserID        *uint
	FromDate      *time.Time
	ToDate        *time.Time
	Search        string // search by order number
}

// List retrieves orders with filtering and pagination
func (r *OrderRepository) List(ctx context.Context, filter OrderFilter, page, limit int) ([]OrderModel, int64, error) {
	var orders []OrderModel
	var total int64

	query := r.db.WithContext(ctx).Model(&OrderModel{})

	// Apply filters
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.PaymentStatus != "" {
		query = query.Where("payment_status = ?", filter.PaymentStatus)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.FromDate != nil {
		query = query.Where("created_at >= ?", filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("created_at <= ?", filter.ToDate)
	}
	if filter.Search != "" {
		query = query.Where("order_number ILIKE ?", "%"+filter.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Preload("Items").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// Update updates an order
func (r *OrderRepository) Update(ctx context.Context, order *OrderModel) error {
	return r.db.WithContext(ctx).Save(order).Error
}

// UpdateStatus updates order status
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// Set timestamp based on status
	switch status {
	case OrderStatusShipped:
		updates["shipped_at"] = time.Now()
	case OrderStatusDelivered:
		updates["delivered_at"] = time.Now()
	case OrderStatusCancelled:
		updates["cancelled_at"] = time.Now()
	}

	return r.db.WithContext(ctx).
		Model(&OrderModel{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdatePaymentStatus updates payment status
func (r *OrderRepository) UpdatePaymentStatus(ctx context.Context, id uint, paymentStatus string) error {
	updates := map[string]interface{}{
		"payment_status": paymentStatus,
		"updated_at":     time.Now(),
	}

	if paymentStatus == PaymentStatusPaid {
		updates["paid_at"] = time.Now()
	}

	return r.db.WithContext(ctx).
		Model(&OrderModel{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetStats returns order statistics
func (r *OrderRepository) GetStats(ctx context.Context) (*OrderStats, error) {
	var stats OrderStats

	// Total orders
	r.db.WithContext(ctx).Model(&OrderModel{}).Count(&stats.TotalOrders)

	// Orders by status
	r.db.WithContext(ctx).Model(&OrderModel{}).Where("status = ?", OrderStatusPending).Count(&stats.PendingOrders)
	r.db.WithContext(ctx).Model(&OrderModel{}).Where("status = ?", OrderStatusProcessing).Count(&stats.ProcessingOrders)
	r.db.WithContext(ctx).Model(&OrderModel{}).Where("status = ?", OrderStatusShipped).Count(&stats.ShippedOrders)
	r.db.WithContext(ctx).Model(&OrderModel{}).Where("status = ?", OrderStatusDelivered).Count(&stats.DeliveredOrders)
	r.db.WithContext(ctx).Model(&OrderModel{}).Where("status = ?", OrderStatusCancelled).Count(&stats.CancelledOrders)

	// Total revenue (from delivered orders)
	r.db.WithContext(ctx).Model(&OrderModel{}).
		Where("status = ? AND payment_status = ?", OrderStatusDelivered, PaymentStatusPaid).
		Select("COALESCE(SUM(total), 0)").
		Scan(&stats.TotalRevenue)

	// Today's orders
	today := time.Now().Truncate(24 * time.Hour)
	r.db.WithContext(ctx).Model(&OrderModel{}).
		Where("created_at >= ?", today).
		Count(&stats.TodayOrders)

	return &stats, nil
}

// GenerateOrderNumber generates a unique order number
func (r *OrderRepository) GenerateOrderNumber(ctx context.Context) (string, error) {
	now := time.Now()
	prefix := fmt.Sprintf("ORD-%s", now.Format("20060102"))

	var count int64
	r.db.WithContext(ctx).Model(&OrderModel{}).
		Where("order_number LIKE ?", prefix+"%").
		Count(&count)

	return fmt.Sprintf("%s-%04d", prefix, count+1), nil
}

// OrderStats holds order statistics
type OrderStats struct {
	TotalOrders      int64   `json:"total_orders"`
	PendingOrders    int64   `json:"pending_orders"`
	ProcessingOrders int64   `json:"processing_orders"`
	ShippedOrders    int64   `json:"shipped_orders"`
	DeliveredOrders  int64   `json:"delivered_orders"`
	CancelledOrders  int64   `json:"cancelled_orders"`
	TotalRevenue     float64 `json:"total_revenue"`
	TodayOrders      int64   `json:"today_orders"`
}
