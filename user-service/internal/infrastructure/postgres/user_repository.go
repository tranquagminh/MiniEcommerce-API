package postgres

import (
	"context"
	"errors"
	"fmt"
	"user-service/internal/application"
	"user-service/internal/domain"

	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

var _ application.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) WithTx(tx *gorm.DB) application.UserRepository {
	return &UserRepository{db: tx}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		if IsDuplicateError(result.Error) {
			return ErrDuplicateUser
		}
		return fmt.Errorf("failed to create user: %w", result.Error)
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	err := r.db.WithContext(ctx).Save(user)
	if err.Error != nil {
		return fmt.Errorf("failed to update user: %w", err.Error)
	}
	return nil
}

func (r *UserRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", id).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("failed to update fields: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&domain.User{}, id)

	if result.Error != nil {
		return fmt.Errorf("failed to soft delete: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) HardDelete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).
		Unscoped(). //Bypass soft delete
		Delete(&domain.User{}, id)

	if result.Error != nil {
		return fmt.Errorf("failed to hard delete: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Restore - restore soft deleted record
func (r *UserRepository) Restore(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Unscoped().
		Where("id = ?", id).
		Update("deleted_at", nil)

	if result.Error != nil {
		return fmt.Errorf("failed to restore: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, int64, error) {
	var users []*domain.User
	var total int64

	// Count total
	if err := r.db.Model(&domain.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count user: %w", err)
	}

	// Get paginated date
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&users).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func (r *UserRepository) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("email = ?", email).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check mail exists: %w", err)
	}

	return count > 0, nil
}
