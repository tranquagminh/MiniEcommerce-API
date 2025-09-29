package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/postgres"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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
	// Validate password
	user.Password = strings.TrimSpace(user.Password)
	if user.Password == "" {
		return errors.New("password is required")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	// Normalize email
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))

	// Use transaction for complex operations
	err = s.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		// Create user with transaction
		repoWithTx := s.repo.WithTx(tx)
		if err := repoWithTx.Create(ctx, user); err != nil {
			if errors.Is(err, postgres.ErrorDuplicateUser) {
				return fmt.Errorf("email already registered")
			}
			return err
		}

		// Future: Add more operations here
		// e.g., create user profile, send welcome email, etc.
		
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	// Get user by email
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrorUserNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	now := time.Now()
	_ = s.repo.UpdateFields(ctx, user.ID, map[string]interface{}{
		"last_login": &now,
	})

	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) error {
	return s.repo.Update(ctx, user)
}

func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}