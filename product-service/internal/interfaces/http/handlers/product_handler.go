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

var validate = validator.New()

type ProductHandler struct {
	service *application.ProductService
}

func NewProductHandler(service *application.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

type CreateProductRequest struct {
	Name              string  `json:"name" validate:"required,min=3,max=200"`
	Slug              string  `json:"slug,omitempty"`
	Description       string  `json:"description"`
	ShortDescription  string  `json:"short_description" validate:"max=500"`
	CategoryID        uint    `json:"category_id" validate:"required"`
	Price             float64 `json:"price" validate:"required,gt=0"`
	CompareAtPrice    float64 `json:"compare_at_price,omitempty"`
	CostPerItem       float64 `json:"cost_per_item,omitempty"`
	SKU               string  `json:"sku,omitempty"`
	Barcode           string  `json:"barcode,omitempty"`
	TrackInventory    bool    `json:"track_inventory"`
	StockQuantity     int     `json:"stock_quantity"`
	LowStockThreshold int     `json:"low_stock_threshold"`
	Weight            float64 `json:"weight,omitempty"`
	Length            float64 `json:"length,omitempty"`
	Width             float64 `json:"width,omitempty"`
	Height            float64 `json:"height,omitempty"`
	Status            string  `json:"status" validate:"omitempty,oneof=draft active archived"`
	IsFeatured        bool    `json:"is_featured"`
	MetaTitle         string  `json:"meta_title,omitempty"`
	MetaDescription   string  `json:"meta_description,omitempty"`
	MetaKeywords      string  `json:"meta_keywords,omitempty"`
}

type UpdateProductRequest struct {
	Name              string  `json:"name" validate:"omitempty,min=3,max=200"`
	Slug              string  `json:"slug,omitempty"`
	Description       string  `json:"description,omitempty"`
	ShortDescription  string  `json:"short_description,omitempty"`
	CategoryID        uint    `json:"category_id,omitempty"`
	Price             float64 `json:"price,omitempty"`
	CompareAtPrice    float64 `json:"compare_at_price,omitempty"`
	CostPerItem       float64 `json:"cost_per_item,omitempty"`
	SKU               string  `json:"sku,omitempty"`
	Barcode           string  `json:"barcode,omitempty"`
	TrackInventory    *bool   `json:"track_inventory,omitempty"`
	StockQuantity     *int    `json:"stock_quantity,omitempty"`
	LowStockThreshold *int    `json:"low_stock_threshold,omitempty"`
	Weight            float64 `json:"weight,omitempty"`
	Length            float64 `json:"length,omitempty"`
	Width             float64 `json:"width,omitempty"`
	Height            float64 `json:"height,omitempty"`
	Status            string  `json:"status,omitempty"`
	IsFeatured        *bool   `json:"is_featured,omitempty"`
	MetaTitle         string  `json:"meta_title,omitempty"`
	MetaDescription   string  `json:"meta_description,omitempty"`
	MetaKeywords      string  `json:"meta_keywords,omitempty"`
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateProductRequest
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

	// Create product
	product := &domain.Product{
		Name:              req.Name,
		Slug:              req.Slug,
		Description:       req.Description,
		ShortDescription:  req.ShortDescription,
		CategoryID:        req.CategoryID,
		Price:             req.Price,
		CompareAtPrice:    req.CompareAtPrice,
		CostPerItem:       req.CostPerItem,
		SKU:               req.SKU,
		Barcode:           req.Barcode,
		TrackInventory:    req.TrackInventory,
		StockQuantity:     req.StockQuantity,
		LowStockThreshold: req.LowStockThreshold,
		Weight:            req.Weight,
		Length:            req.Length,
		Width:             req.Width,
		Height:            req.Height,
		Status:            req.Status,
		IsFeatured:        req.IsFeatured,
		MetaTitle:         req.MetaTitle,
		MetaDescription:   req.MetaDescription,
		MetaKeywords:      req.MetaKeywords,
	}

	ctx := r.Context()
	if err := h.service.CreateProduct(ctx, product); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Product created successfully",
		"product": product,
	})
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from path
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	product, err := h.service.GetProduct(ctx, uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from path
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get existing product
	product, err := h.service.GetProduct(ctx, uint(id))
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Slug != "" {
		product.Slug = req.Slug
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.ShortDescription != "" {
		product.ShortDescription = req.ShortDescription
	}
	if req.CategoryID > 0 {
		product.CategoryID = req.CategoryID
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.CompareAtPrice > 0 {
		product.CompareAtPrice = req.CompareAtPrice
	}
	if req.CostPerItem > 0 {
		product.CostPerItem = req.CostPerItem
	}
	if req.SKU != "" {
		product.SKU = req.SKU
	}
	if req.Barcode != "" {
		product.Barcode = req.Barcode
	}
	if req.TrackInventory != nil {
		product.TrackInventory = *req.TrackInventory
	}
	if req.StockQuantity != nil {
		product.StockQuantity = *req.StockQuantity
	}
	if req.LowStockThreshold != nil {
		product.LowStockThreshold = *req.LowStockThreshold
	}
	if req.Status != "" {
		product.Status = req.Status
	}
	if req.IsFeatured != nil {
		product.IsFeatured = *req.IsFeatured
	}

	// Update product
	if err := h.service.UpdateProduct(ctx, product); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Product updated successfully",
		"product": product,
	})
}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from path
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.service.DeleteProduct(ctx, uint(id)); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Product deleted successfully",
	})
}

func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query params
	query := r.URL.Query()

	filters := &application.ProductFilters{
		Page:     1,
		PageSize: 20,
	}

	// Parse filters
	if page := query.Get("page"); page != "" {
		fmt.Sscanf(page, "%d", &filters.Page)
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		fmt.Sscanf(pageSize, "%d", &filters.PageSize)
	}
	if categoryID := query.Get("category_id"); categoryID != "" {
		var cid uint
		fmt.Sscanf(categoryID, "%d", &cid)
		filters.CategoryID = cid
	}
	if status := query.Get("status"); status != "" {
		filters.Status = status
	}
	if featured := query.Get("is_featured"); featured != "" {
		isFeatured := featured == "true"
		filters.IsFeatured = &isFeatured
	}
	if inStock := query.Get("in_stock"); inStock == "true" {
		filters.InStock = true
	}
	if minPrice := query.Get("min_price"); minPrice != "" {
		fmt.Sscanf(minPrice, "%f", &filters.MinPrice)
	}
	if maxPrice := query.Get("max_price"); maxPrice != "" {
		fmt.Sscanf(maxPrice, "%f", &filters.MaxPrice)
	}
	if search := query.Get("q"); search != "" {
		filters.Search = search
	}
	if sort := query.Get("sort"); sort != "" {
		filters.Sort = sort
	}

	ctx := r.Context()
	products, total, err := h.service.ListProducts(ctx, filters)
	if err != nil {
		http.Error(w, "Failed to list products", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"products":    products,
		"total":       total,
		"page":        filters.Page,
		"page_size":   filters.PageSize,
		"total_pages": (total + int64(filters.PageSize) - 1) / int64(filters.PageSize),
	})
}

func (h *ProductHandler) AddProductImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path: /products/{id}/images
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	productID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ImageURL string `json:"image_url"`
		AltText  string `json:"alt_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ImageURL == "" {
		http.Error(w, "image_url is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	image := &domain.ProductImage{
		ImageURL: req.ImageURL,
		AltText:  req.AltText,
	}
	if err := h.service.AddProductImage(ctx, uint(productID), image); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Image added successfully",
		"image":   image,
	})
}

func (h *ProductHandler) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path: /products/{id}/images/{imageId}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	imageID, err := strconv.ParseUint(parts[3], 10, 32)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.service.RemoveProductImage(ctx, uint(imageID)); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Image removed successfully",
	})
}

func formatValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}
