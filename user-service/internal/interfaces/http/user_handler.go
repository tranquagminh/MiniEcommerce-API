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
	Username string `json:"username" validate:"required, min=3, max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required, min=6"`
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
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if err := validate.Struct(req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	u := domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.service.Register(&u); err != nil {
		if strings.Contains((err.Error()), "duplicate") {
			http.Error(w, "Email already in use", http.StatusConflict)
			return
		}
		http.Error(w, "Could not register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User created"})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		http.Error(w, "email/password required", http.StatusBadRequest)
		return
	}

	user, err := h.service.Login(email, password)
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
		"users":      users,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
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
	json.
}
