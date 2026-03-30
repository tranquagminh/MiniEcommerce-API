package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// Create creates a new review
func (r *ReviewRepository) Create(ctx context.Context, review *ReviewModel) error {
	return r.db.WithContext(ctx).Create(review).Error
}

// GetByID retrieves a review by ID
func (r *ReviewRepository) GetByID(ctx context.Context, id uint) (*ReviewModel, error) {
	var review ReviewModel
	err := r.db.WithContext(ctx).
		Preload("Product").
		First(&review, id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// GetByProductID retrieves all approved reviews for a product
func (r *ReviewRepository) GetByProductID(ctx context.Context, productID uint, page, limit int) ([]ReviewModel, int64, error) {
	var reviews []ReviewModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ReviewModel{}).
		Where("product_id = ? AND status = ?", productID, ReviewStatusApproved)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	return reviews, total, err
}

// GetByUserID retrieves all reviews by a user
func (r *ReviewRepository) GetByUserID(ctx context.Context, userID uint, page, limit int) ([]ReviewModel, int64, error) {
	var reviews []ReviewModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ReviewModel{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Preload("Product").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	return reviews, total, err
}

// ReviewFilter for filtering reviews
type ReviewFilter struct {
	Status    string
	ProductID *uint
	UserID    *uint
	Rating    *int
}

// List retrieves reviews with filtering (for admin)
func (r *ReviewRepository) List(ctx context.Context, filter ReviewFilter, page, limit int) ([]ReviewModel, int64, error) {
	var reviews []ReviewModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ReviewModel{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ProductID != nil {
		query = query.Where("product_id = ?", *filter.ProductID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Rating != nil {
		query = query.Where("rating = ?", *filter.Rating)
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
		Find(&reviews).Error

	return reviews, total, err
}

// Update updates a review
func (r *ReviewRepository) Update(ctx context.Context, review *ReviewModel) error {
	return r.db.WithContext(ctx).Save(review).Error
}

// UpdateStatus updates review status (approve/reject)
func (r *ReviewRepository) UpdateStatus(ctx context.Context, id uint, status string, reason string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == ReviewStatusRejected && reason != "" {
		updates["rejected_reason"] = reason
	}

	return r.db.WithContext(ctx).
		Model(&ReviewModel{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// AddAdminReply adds admin reply to a review
func (r *ReviewRepository) AddAdminReply(ctx context.Context, id uint, reply string) error {
	return r.db.WithContext(ctx).
		Model(&ReviewModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"admin_reply":    reply,
			"admin_reply_at": time.Now(),
			"updated_at":     time.Now(),
		}).Error
}

// Delete soft deletes a review
func (r *ReviewRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&ReviewModel{}, id).Error
}

// GetProductRatingStats returns rating statistics for a product
func (r *ReviewRepository) GetProductRatingStats(ctx context.Context, productID uint) (*ProductRatingStats, error) {
	var stats ProductRatingStats

	// Total reviews
	r.db.WithContext(ctx).Model(&ReviewModel{}).
		Where("product_id = ? AND status = ?", productID, ReviewStatusApproved).
		Count(&stats.TotalReviews)

	// Average rating
	r.db.WithContext(ctx).Model(&ReviewModel{}).
		Where("product_id = ? AND status = ?", productID, ReviewStatusApproved).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&stats.AverageRating)

	// Rating distribution
	for i := 1; i <= 5; i++ {
		var count int64
		r.db.WithContext(ctx).Model(&ReviewModel{}).
			Where("product_id = ? AND status = ? AND rating = ?", productID, ReviewStatusApproved, i).
			Count(&count)
		stats.RatingDistribution[i] = count
	}

	return &stats, nil
}

// HasUserReviewed checks if a user has already reviewed a product
func (r *ReviewRepository) HasUserReviewed(ctx context.Context, userID, productID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ReviewModel{}).
		Where("user_id = ? AND product_id = ?", userID, productID).
		Count(&count).Error
	return count > 0, err
}

// IncrementHelpful increments the helpful count
func (r *ReviewRepository) IncrementHelpful(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&ReviewModel{}).
		Where("id = ?", id).
		UpdateColumn("helpful_count", gorm.Expr("helpful_count + 1")).Error
}

// ProductRatingStats holds rating statistics
type ProductRatingStats struct {
	TotalReviews       int64          `json:"total_reviews"`
	AverageRating      float64        `json:"average_rating"`
	RatingDistribution map[int]int64  `json:"rating_distribution"`
}

func init() {
	// Initialize the map
}

func NewProductRatingStats() *ProductRatingStats {
	return &ProductRatingStats{
		RatingDistribution: make(map[int]int64),
	}
}
