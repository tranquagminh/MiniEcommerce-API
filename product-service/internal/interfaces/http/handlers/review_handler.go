package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"product-service/internal/infrastructure/postgres"
)

type ReviewHandler struct {
	repo *postgres.ReviewRepository
}

func NewReviewHandler(repo *postgres.ReviewRepository) *ReviewHandler {
	return &ReviewHandler{repo: repo}
}

// ReviewResponse is the DTO for review responses with flattened fields
type ReviewResponse struct {
	ID             uint    `json:"id"`
	UserID         uint    `json:"user_id"`
	ProductID      uint    `json:"product_id"`
	ProductName    string  `json:"product_name,omitempty"`
	OrderID        *uint   `json:"order_id,omitempty"`
	Rating         int     `json:"rating"`
	Title          string  `json:"title,omitempty"`
	Content        string  `json:"content"`
	Images         string  `json:"images,omitempty"`
	Status         string  `json:"status"`
	RejectedReason string  `json:"rejected_reason,omitempty"`
	HelpfulCount   int     `json:"helpful_count"`
	AdminReply     string  `json:"admin_reply,omitempty"`
	AdminReplyAt   *string `json:"admin_reply_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func toReviewResponse(review *postgres.ReviewModel) ReviewResponse {
	resp := ReviewResponse{
		ID:             review.ID,
		UserID:         review.UserID,
		ProductID:      review.ProductID,
		OrderID:        review.OrderID,
		Rating:         review.Rating,
		Title:          review.Title,
		Content:        review.Content,
		Images:         review.Images,
		Status:         review.Status,
		RejectedReason: review.RejectedReason,
		HelpfulCount:   review.HelpfulCount,
		AdminReply:     review.AdminReply,
		CreatedAt:      review.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      review.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if review.AdminReplyAt != nil {
		formatted := review.AdminReplyAt.Format("2006-01-02T15:04:05Z07:00")
		resp.AdminReplyAt = &formatted
	}
	if review.Product != nil {
		resp.ProductName = review.Product.Name
	}
	return resp
}

func toReviewResponses(reviews []postgres.ReviewModel) []ReviewResponse {
	responses := make([]ReviewResponse, len(reviews))
	for i, review := range reviews {
		responses[i] = toReviewResponse(&review)
	}
	return responses
}

// CreateReviewRequest represents the request body for creating a review
type CreateReviewRequest struct {
	UserID    uint   `json:"user_id"`
	ProductID uint   `json:"product_id"`
	OrderID   *uint  `json:"order_id,omitempty"`
	Rating    int    `json:"rating"`
	Title     string `json:"title,omitempty"`
	Content   string `json:"content"`
	Images    string `json:"images,omitempty"` // JSON array of URLs
}

// CreateReview handles POST /reviews
func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate rating
	if req.Rating < 1 || req.Rating > 5 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Rating must be between 1 and 5"})
		return
	}

	// Check if user already reviewed this product
	hasReviewed, err := h.repo.HasUserReviewed(r.Context(), req.UserID, req.ProductID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check existing review"})
		return
	}
	if hasReviewed {
		respondJSON(w, http.StatusConflict, map[string]string{"error": "You have already reviewed this product"})
		return
	}

	review := &postgres.ReviewModel{
		UserID:    req.UserID,
		ProductID: req.ProductID,
		OrderID:   req.OrderID,
		Rating:    req.Rating,
		Title:     req.Title,
		Content:   req.Content,
		Images:    req.Images,
		Status:    postgres.ReviewStatusPending,
	}

	if err := h.repo.Create(r.Context(), review); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create review"})
		return
	}

	respondJSON(w, http.StatusCreated, review)
}

// GetReview handles GET /reviews/{id}
func (h *ReviewHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/reviews/")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid review ID"})
		return
	}

	review, err := h.repo.GetByID(r.Context(), uint(id))
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Review not found"})
		return
	}

	respondJSON(w, http.StatusOK, review)
}

// ListReviews handles GET /reviews (admin)
func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	filter := postgres.ReviewFilter{
		Status: query.Get("status"),
	}

	if productID := query.Get("product_id"); productID != "" {
		if id, err := strconv.ParseUint(productID, 10, 32); err == nil {
			pid := uint(id)
			filter.ProductID = &pid
		}
	}

	if rating := query.Get("rating"); rating != "" {
		if r, err := strconv.Atoi(rating); err == nil {
			filter.Rating = &r
		}
	}

	reviews, total, err := h.repo.List(r.Context(), filter, page, limit)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list reviews"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        toReviewResponses(reviews),
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetProductReviews handles GET /products/{id}/reviews (public)
func (h *ReviewHandler) GetProductReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/products/")
	path = strings.TrimSuffix(path, "/reviews")
	productID, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	reviews, total, err := h.repo.GetByProductID(r.Context(), uint(productID), page, limit)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get reviews"})
		return
	}

	// Get rating stats
	stats, _ := h.repo.GetProductRatingStats(r.Context(), uint(productID))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        reviews,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
		"stats":       stats,
	})
}

// UpdateReviewStatus handles PUT /reviews/{id}/status
func (h *ReviewHandler) UpdateReviewStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/reviews/")
	path = strings.TrimSuffix(path, "/status")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid review ID"})
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

	validStatuses := map[string]bool{
		postgres.ReviewStatusApproved: true,
		postgres.ReviewStatusRejected: true,
	}
	if !validStatuses[req.Status] {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid status"})
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), uint(id), req.Status, req.Reason); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update status"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Status updated successfully"})
}

// AddAdminReply handles POST /reviews/{id}/reply
func (h *ReviewHandler) AddAdminReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/reviews/")
	path = strings.TrimSuffix(path, "/reply")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid review ID"})
		return
	}

	var req struct {
		Reply string `json:"reply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Reply == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Reply cannot be empty"})
		return
	}

	if err := h.repo.AddAdminReply(r.Context(), uint(id), req.Reply); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add reply"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Reply added successfully"})
}

// DeleteReview handles DELETE /reviews/{id}
func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/reviews/")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid review ID"})
		return
	}

	if err := h.repo.Delete(r.Context(), uint(id)); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete review"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Review deleted successfully"})
}
