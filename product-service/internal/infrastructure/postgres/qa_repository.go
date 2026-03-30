package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type QARepository struct {
	db *gorm.DB
}

func NewQARepository(db *gorm.DB) *QARepository {
	return &QARepository{db: db}
}

// Create creates a new Q&A entry
func (r *QARepository) Create(ctx context.Context, qa *ProductQAModel) error {
	return r.db.WithContext(ctx).Create(qa).Error
}

// GetByID retrieves a Q&A by ID
func (r *QARepository) GetByID(ctx context.Context, id uint) (*ProductQAModel, error) {
	var qa ProductQAModel
	err := r.db.WithContext(ctx).
		Preload("Product").
		First(&qa, id).Error
	if err != nil {
		return nil, err
	}
	return &qa, nil
}

// GetByProductID retrieves all Q&A for a product (answered only for public)
func (r *QARepository) GetByProductID(ctx context.Context, productID uint, includeUnanswered bool, page, limit int) ([]ProductQAModel, int64, error) {
	var qas []ProductQAModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ProductQAModel{}).Where("product_id = ?", productID)

	if !includeUnanswered {
		query = query.Where("status = ?", QAStatusAnswered)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&qas).Error

	return qas, total, err
}

// QAFilter for filtering Q&A
type QAFilter struct {
	Status    string
	ProductID *uint
	UserID    *uint
}

// List retrieves Q&A with filtering (for admin)
func (r *QARepository) List(ctx context.Context, filter QAFilter, page, limit int) ([]ProductQAModel, int64, error) {
	var qas []ProductQAModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ProductQAModel{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ProductID != nil {
		query = query.Where("product_id = ?", *filter.ProductID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Preload("Product").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&qas).Error

	return qas, total, err
}

// Answer adds an answer to a question
func (r *QARepository) Answer(ctx context.Context, id uint, answer string, answeredBy uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&ProductQAModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"answer":      answer,
			"answered_by": answeredBy,
			"answered_at": now,
			"status":      QAStatusAnswered,
			"updated_at":  now,
		}).Error
}

// Update updates a Q&A
func (r *QARepository) Update(ctx context.Context, qa *ProductQAModel) error {
	return r.db.WithContext(ctx).Save(qa).Error
}

// Delete soft deletes a Q&A
func (r *QARepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&ProductQAModel{}, id).Error
}

// GetPendingCount returns count of pending questions
func (r *QARepository) GetPendingCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ProductQAModel{}).
		Where("status = ?", QAStatusPending).
		Count(&count).Error
	return count, err
}

// IncrementHelpful increments the helpful count
func (r *QARepository) IncrementHelpful(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&ProductQAModel{}).
		Where("id = ?", id).
		UpdateColumn("helpful_count", gorm.Expr("helpful_count + 1")).Error
}
