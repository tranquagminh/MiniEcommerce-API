package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"product-service/internal/infrastructure/postgres"
)

type OrderHandler struct {
	repo *postgres.OrderRepository
}

func NewOrderHandler(repo *postgres.OrderRepository) *OrderHandler {
	return &OrderHandler{repo: repo}
}

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	UserID           uint               `json:"user_id"`
	ShippingFullName string             `json:"shipping_full_name"`
	ShippingPhone    string             `json:"shipping_phone"`
	ShippingProvince string             `json:"shipping_province"`
	ShippingDistrict string             `json:"shipping_district"`
	ShippingWard     string             `json:"shipping_ward"`
	ShippingAddress  string             `json:"shipping_address"`
	PaymentMethod    string             `json:"payment_method"`
	CouponCode       string             `json:"coupon_code,omitempty"`
	Note             string             `json:"note,omitempty"`
	Items            []CreateOrderItem  `json:"items"`
}

type CreateOrderItem struct {
	ProductID      uint    `json:"product_id"`
	VariantID      *uint   `json:"variant_id,omitempty"`
	ProductName    string  `json:"product_name"`
	ProductImage   string  `json:"product_image,omitempty"`
	VariantOptions string  `json:"variant_options,omitempty"`
	SKU            string  `json:"sku,omitempty"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
}

// CreateOrder handles POST /orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Generate order number
	orderNumber, err := h.repo.GenerateOrderNumber(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate order number"})
		return
	}

	// Calculate totals
	var subtotal float64
	items := make([]postgres.OrderItemModel, len(req.Items))
	for i, item := range req.Items {
		totalPrice := float64(item.Quantity) * item.UnitPrice
		subtotal += totalPrice
		items[i] = postgres.OrderItemModel{
			ProductID:      item.ProductID,
			VariantID:      item.VariantID,
			ProductName:    item.ProductName,
			ProductImage:   item.ProductImage,
			VariantOptions: item.VariantOptions,
			SKU:            item.SKU,
			Quantity:       item.Quantity,
			UnitPrice:      item.UnitPrice,
			TotalPrice:     totalPrice,
		}
	}

	// TODO: Calculate shipping fee based on address
	shippingFee := 30000.0
	if subtotal >= 500000 {
		shippingFee = 0 // Free shipping
	}

	// TODO: Apply coupon discount
	discountAmount := 0.0

	total := subtotal + shippingFee - discountAmount

	order := &postgres.OrderModel{
		UserID:           req.UserID,
		OrderNumber:      orderNumber,
		ShippingFullName: req.ShippingFullName,
		ShippingPhone:    req.ShippingPhone,
		ShippingProvince: req.ShippingProvince,
		ShippingDistrict: req.ShippingDistrict,
		ShippingWard:     req.ShippingWard,
		ShippingAddress:  req.ShippingAddress,
		Status:           postgres.OrderStatusPending,
		PaymentMethod:    req.PaymentMethod,
		PaymentStatus:    postgres.PaymentStatusPending,
		Subtotal:         subtotal,
		ShippingFee:      shippingFee,
		DiscountAmount:   discountAmount,
		CouponCode:       req.CouponCode,
		Total:            total,
		Note:             req.Note,
		Items:            items,
	}

	if err := h.repo.Create(r.Context(), order); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create order"})
		return
	}

	respondJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/orders/")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid order ID"})
		return
	}

	order, err := h.repo.GetByID(r.Context(), uint(id))
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Order not found"})
		return
	}

	respondJSON(w, http.StatusOK, order)
}

// ListOrders handles GET /orders
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := postgres.OrderFilter{
		Status:        query.Get("status"),
		PaymentStatus: query.Get("payment_status"),
		Search:        query.Get("search"),
	}

	if userID := query.Get("user_id"); userID != "" {
		if id, err := strconv.ParseUint(userID, 10, 32); err == nil {
			uid := uint(id)
			filter.UserID = &uid
		}
	}

	if fromDate := query.Get("from_date"); fromDate != "" {
		if t, err := time.Parse("2006-01-02", fromDate); err == nil {
			filter.FromDate = &t
		}
	}

	if toDate := query.Get("to_date"); toDate != "" {
		if t, err := time.Parse("2006-01-02", toDate); err == nil {
			filter.ToDate = &t
		}
	}

	orders, total, err := h.repo.List(r.Context(), filter, page, limit)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list orders"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        orders,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// UpdateOrderStatus handles PUT /orders/{id}/status
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/orders/")
	path = strings.TrimSuffix(path, "/status")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid order ID"})
		return
	}

	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		postgres.OrderStatusPending:    true,
		postgres.OrderStatusConfirmed:  true,
		postgres.OrderStatusProcessing: true,
		postgres.OrderStatusShipped:    true,
		postgres.OrderStatusDelivered:  true,
		postgres.OrderStatusCancelled:  true,
	}
	if !validStatuses[req.Status] {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid status"})
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), uint(id), req.Status); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update status"})
		return
	}

	// If cancelled, update with reason
	if req.Status == postgres.OrderStatusCancelled && req.Reason != "" {
		order, _ := h.repo.GetByID(r.Context(), uint(id))
		if order != nil {
			order.CancelledReason = req.Reason
			h.repo.Update(r.Context(), order)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Status updated successfully"})
}

// GetOrderStats handles GET /orders/stats
func (h *OrderHandler) GetOrderStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.repo.GetStats(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get stats"})
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetUserOrders handles GET /users/{user_id}/orders
func (h *OrderHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	path = strings.TrimSuffix(path, "/orders")
	userID, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	orders, total, err := h.repo.GetByUserID(r.Context(), uint(userID), page, limit)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get orders"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        orders,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}
