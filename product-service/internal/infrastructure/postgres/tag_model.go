package postgres

import (
	"product-service/internal/domain"
	"time"
)

type TagModel struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:50;not null;uniqueIndex" json:"name"`
	Slug      string    `gorm:"size:50;not null;uniqueIndex" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

func (TagModel) TableName() string {
	return "tags"
}

func (m *TagModel) ToDomain() *domain.Tag {
	return &domain.Tag{
		ID:        m.ID,
		Name:      m.Name,
		Slug:      m.Slug,
		CreatedAt: m.CreatedAt,
	}
}

func (m *TagModel) FromDomain(t *domain.Tag) {
	m.ID = t.ID
	m.Name = t.Name
	m.Slug = t.Slug
	m.CreatedAt = t.CreatedAt
}
