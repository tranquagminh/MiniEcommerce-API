package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"product-service/internal/application"
	"product-service/internal/domain"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

type CategoryHandler struct {
	service *application.CategoryService
}

func NewCategoryHandler(service *application.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

type CreateCategoryRequest struct {
	Name         string `json:"name" validate:"required,min=2,max=100"`
	Slug         string `json:"slug,omitempty"`
	Description  string `json:"description"`
	ParentID     *uint  `json:"parent_id,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	IsActive     bool   `json:"is_active"`
	DisplayOrder int    `json:"display_order"`
}

type UpdateCategoryRequest struct {
	Name         string `json:"name,omitempty"`
	Slug         string `json:"slug,omitempty"`
	Description  string `json:"description,omitempty"`
	ParentID     *uint  `json:"parent_id,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
	DisplayOrder *int   `json:"display_order,omitempty"`
}

func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if err := validate.Struct(req); err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			http.Error(w, "Validation failed", http.StatusBadRequest)
			return
		}

		errorMessages := make(map[string]string)
		for _, e := range validationErrors {
			errorMessages[strings.ToLower(e.Field())] = formatValidationError(e)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Validation failed",
			"fields": errorMessages,
		})
		return
	}

	// Create category
	category := &domain.Category{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		ParentID:     req.ParentID,
		ImageURL:     req.ImageURL,
		IsActive:     req.IsActive,
		DisplayOrder: req.DisplayOrder,
	}

	ctx := r.Context()
	if err := h.service.CreateCategory(ctx, category); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Category created successfully",
		"category": category,
	})
}

func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from path
	idStr := r.URL.Path[len("/categories/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	category, err := h.service.GetCategory(ctx, uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(category)
}

func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from path
	idStr := r.URL.Path[len("/categories/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get existing category
	category, err := h.service.GetCategory(ctx, uint(id))
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != "" {
		category.Name = req.Name
	}
	if req.Slug != "" {
		category.Slug = req.Slug
	}
	if req.Description != "" {
		category.Description = req.Description
	}
	if req.ParentID != nil {
		category.ParentID = req.ParentID
	}
	if req.ImageURL != "" {
		category.ImageURL = req.ImageURL
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	if req.DisplayOrder != nil {
		category.DisplayOrder = *req.DisplayOrder
	}

	// Update category
	if err := h.service.UpdateCategory(ctx, category); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Category updated successfully",
		"category": category,
	})
}

func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from path
	idStr := r.URL.Path[len("/categories/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.service.DeleteCategory(ctx, uint(id)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Category deleted successfully",
	})
}

func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query params
	query := r.URL.Query()
	activeOnly := query.Get("active_only") == "true"

	ctx := r.Context()
	categories, err := h.service.ListCategories(ctx, activeOnly)
	if err != nil {
		http.Error(w, "Failed to list categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": categories,
		"total":      len(categories),
	})
}

func (h *CategoryHandler) GetRootCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query params
	query := r.URL.Query()
	activeOnly := query.Get("active_only") == "true"

	ctx := r.Context()
	categories, err := h.service.GetRootCategories(ctx, activeOnly)
	if err != nil {
		http.Error(w, "Failed to get root categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": categories,
		"total":      len(categories),
	})
}

func (h *CategoryHandler) GetCategoryProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	idStr := pathParts[2]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Parse query params for pagination
	query := r.URL.Query()

	filters := &application.ProductFilters{
		CategoryID: uint(id),
		Page:       1,
		PageSize:   20,
		Status:     "active", // Only active products
	}

	if page := query.Get("page"); page != "" {
		fmt.Sscanf(page, "%d", &filters.Page)
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		fmt.Sscanf(pageSize, "%d", &filters.PageSize)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Use /products endpoint with category_id filter",
		"example": fmt.Sprintf("/products?category_id=%d&page=1&page_size=20", id),
	})
}
