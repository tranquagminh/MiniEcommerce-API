package application

import (
	"context"
	"fmt"
	"strings"
	"time"
	"user-service/internal/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserCache interface {
	Set(ctx context.Context, user *domain.User) error
	Get(ctx context.Context, userID uint) (*domain.User, error)
	Delete(ctx context.Context, userID uint) error
	SetByEmail(ctx context.Context, email string, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	DeleteByEmail(ctx context.Context, email string) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uint) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id uint) error
	ExistsEmail(ctx context.Context, email string) (bool, error)
	List(ctx context.Context, offset, limit int) ([]*domain.User, int64, error)
	WithTx(tx *gorm.DB) UserRepository
}

type TransactionManager interface {
	ExecuteInTx(ctx context.Context, fn func(tx *gorm.DB) error) error
}

type UserService struct {
	repo      UserRepository
	txManager TransactionManager
	cache     UserCache
}

func NewUserService(repo UserRepository, txManager TransactionManager, cache UserCache) *UserService {
	return &UserService{
		repo:      repo,
		txManager: txManager,
		cache:     cache,
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
	if err := s.repo.UpdateFields(ctx, user.ID, map[string]interface{}{
		"last_login": &now,
	}); err != nil {
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	user.LastLogin = &now
	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	// Try cache first
	if s.cache != nil {
		user, err := s.cache.Get(ctx, id)
		if err == nil {
			return user, nil
		}
		// If error, continue to database
	}

	// Get from database
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		_ = s.cache.Set(ctx, user)
	}

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) error {
	err := s.repo.Update(ctx, user)
	if err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.Delete(ctx, user.ID)
		_ = s.cache.DeleteByEmail(ctx, user.Email)
	}

	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	return s.repo.SoftDelete(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}
