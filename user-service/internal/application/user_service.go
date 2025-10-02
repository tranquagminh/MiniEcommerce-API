package application

import (
	"context"
	"fmt"
	"strings"
	"time"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/postgres"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uint) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id uint) error
	ExistsEmail(ctx context.Context, email string) (bool, error)
	List(ctx context.Context, offset, limit int) ([]*domain.User, int64, error)
}

type UserService struct {
	repo      *postgres.UserRepository
	txManager *postgres.TransactionManager
}

func NewUserService(repo *postgres.UserRepository, txManager *postgres.TransactionManager) *UserService {
	return &UserService{
		repo:      repo,
		txManager: txManager,
	}
}

func (s *UserService) Register(ctx context.Context, user *domain.User) error {
	// Trim and validate
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Username = strings.TrimSpace(user.Username)
	password := strings.TrimSpace(user.Password)

	if password == "" {
		return fmt.Errorf("password is required")
	}

	// Check if email exists
	exists, err := s.repo.ExistsEmail(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return fmt.Errorf("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	// Use transaction for complex operations
	err = s.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		// Create user
		userRepo := s.repo.WithTx(tx)
		if err := userRepo.Create(ctx, user); err != nil {
			return err
		}

		// Could add more operations here (e.g., create profile, send event, etc.)
		// Example: create default settings, audit log, etc.

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	return nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login time
	now := time.Now()
	s.repo.UpdateFields(ctx, user.ID, map[string]interface{}{
		"last_login": &now,
	})

	user.LastLogin = &now
	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) error {
	return s.repo.Update(ctx, user)
}

func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	return s.repo.SoftDelete(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}
