package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"user-service/internal/application"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/auth"
	"user-service/internal/infrastructure/security"
	"user-service/internal/interfaces/http/middleware"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
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

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone" validate:"omitempty,e164"` // E.164 format
	Gender    string `json:"gender" validate:"omitempty,oneof=male female other"`
	Birthday  string `json:"birthday" validate:"omitempty"` // Format: YYYY-MM-DD
}

// NEW: Change Password Request ✅
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

type UserHandler struct {
	service    *application.UserService
	jwtManager *auth.JWTManager
	sanitizer  *security.Sanitizer
}

func NewUserHandler(s *application.UserService, jwt *auth.JWTManager) *UserHandler {
	return &UserHandler{
		service:    s,
		jwtManager: jwt,
		sanitizer:  security.NewSanitizer(),
	}
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

	// Sanitize inputs
	sanitizedUsername := h.sanitizer.SanitizeUsername(req.Username)
	sanitizedEmail := h.sanitizer.SanitizeEmail(req.Email)

	u := domain.User{
		Username: sanitizedUsername,
		Email:    sanitizedEmail,
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

	// Record metrics
	middleware.RecordUserRegistration()

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
		middleware.RecordFailedLogin()
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	// Record successful login
	middleware.RecordUserLogin()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Login successful",
		"user":    user,
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

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserID(r)
	if userID == 0 {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var updateReq UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validate.Struct(updateReq); err != nil {
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

	ctx := r.Context()

	// Get current user
	user, err := h.service.GetUser(ctx, uint(userID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update fields with sanitization
	if updateReq.FirstName != "" {
		user.FirstName = h.sanitizer.SanitizeString(updateReq.FirstName)
	}
	if updateReq.LastName != "" {
		user.LastName = h.sanitizer.SanitizeString(updateReq.LastName)
	}
	if updateReq.Username != "" {
		user.Username = h.sanitizer.SanitizeUsername(updateReq.Username)
	}
	if updateReq.Phone != "" {
		user.Phone = h.sanitizer.SanitizePhone(updateReq.Phone)
	}
	if updateReq.Gender != "" {
		user.Gender = h.sanitizer.SanitizeString(updateReq.Gender)
	}
	if updateReq.Birthday != "" {
		// Parse birthday string to time.Time in UTC
		birthday, err := time.Parse("2006-01-02", updateReq.Birthday)
		if err != nil {
			http.Error(w, "Invalid birthday format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		// Ensure birthday is in UTC and set to start of day
		birthdayUTC := time.Date(birthday.Year(), birthday.Month(), birthday.Day(), 0, 0, 0, 0, time.UTC)
		user.Birthday = &birthdayUTC
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
		"message": "Profile updated successfully",
		"user":    user,
	})
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserID(r)
	if userID == 0 {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
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

	ctx := r.Context()

	// Get current user
	user, err := h.service.GetUser(ctx, uint(userID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword))
	if err != nil {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := h.service.UpdateUser(ctx, user); err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	// Record password change
	middleware.RecordPasswordChange()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Password changed successfully",
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
	if pageSize <= 0 {
		pageSize = 10 // default
	}
	if pageSize > 100 {
		pageSize = 100
	}

	if page <= 0 {
		page = 1
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
