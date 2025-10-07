package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"user-service/internal/application"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/auth"
	"user-service/internal/interfaces/http/middleware"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserHandler struct {
	service    *application.UserService
	jwtManager *auth.JWTManager
}

func NewUserHandler(s *application.UserService, jwt *auth.JWTManager) *UserHandler {
	return &UserHandler{service: s, jwtManager: jwt}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// FIX: Remove spaces in validation tags
	// validate:"required,min=3,max=50" NOT "required, min=3, max=50"
	if err := validate.Struct(req); err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			http.Error(w, "Validation failed", http.StatusBadRequest)
			return
		}

		// Tạo map chứa lỗi cho từng field
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

	u := domain.User{
		Username: strings.TrimSpace(req.Username),
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
		Password: req.Password,
	}

	ctx := r.Context() // FIX: Add context
	if err := h.service.Register(ctx, &u); err != nil {
		if strings.Contains(err.Error(), "already registered") {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Could not register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User registered successfully",
		"user": UserResponse{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
		},
	})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Login successful",
		"user":    UserResponse{ID: user.ID, Username: user.Username, Email: user.Email},
		"token":   token,
	})
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == 0 {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	user, err := h.service.GetUser(ctx, uint(userID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Don't send password
	user.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserID(r)
	if userID == 0 {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var updateReq struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get current user
	user, err := h.service.GetUser(ctx, uint(userID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update fields
	if updateReq.FirstName != "" {
		user.FirstName = updateReq.FirstName
	}
	if updateReq.LastName != "" {
		user.LastName = updateReq.LastName
	}
	if updateReq.Username != "" {
		user.Username = updateReq.Username
	}

	// Save updates
	if err := h.service.UpdateUser(ctx, user); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// Return updated user (without password)
	user.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User updated successfully",
		"user":    user,
	})
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query params
	page := 1
	pageSize := 10

	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}

	if ps := r.URL.Query().Get("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	// Limit page size
	if pageSize > 100 {
		pageSize = 100
	}

	ctx := r.Context()
	users, total, err := h.service.ListUsers(ctx, page, pageSize)
	if err != nil {
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	// Remove passwords
	for _, user := range users {
		user.Password = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users":       users,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserID(r)
	if userID == 0 {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	if err := h.service.DeleteUser(ctx, uint(userID)); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User delete successfully",
		"user_id": userID,
	})
}

func formatValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}
