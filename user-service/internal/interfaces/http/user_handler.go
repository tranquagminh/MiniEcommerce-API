package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"user-service/internal/application"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/auth"

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
