package postgres

import (
	"time"
	"user-service/internal/domain"

	"gorm.io/gorm"
)

type UserModel struct {
	ID        uint           `gorm:"primaryKey"`
	Username  string         `gorm:"size:100;not null" json:"username"`
	Email     string         `gorm:"size:100;not null;uniqueIndex" json:"email"`
	Password  string         `gorm:"not null" json:"-"` // json:"-" to never expose
	FirstName string         `gorm:"size:100" json:"first_name,omitempty"`
	LastName  string         `gorm:"size:100" json:"last_name,omitempty"`
	LastLogin *time.Time     `json:"last_login,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (UserModel) TableName() string {
	return "users"
}

func (m *UserModel) ToDomain() *domain.User {
	var deletedAt gorm.DeletedAt
	if m.DeletedAt.Valid {
		deletedAt = m.DeletedAt
	}

	return &domain.User{
		ID:        m.ID,
		Username:  m.Username,
		Email:     m.Email,
		Password:  m.Password,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		LastLogin: m.LastLogin,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: deletedAt,
	}

}

// FromDomain coverts domain entity to Gorm model
func (m *UserModel) FromDomain(user *domain.User) {
	m.ID = user.ID
	m.Username = user.Username
	m.Email = user.Email
	m.Password = user.Password
	m.FirstName = user.FirstName
	m.LastName = user.LastName
	m.LastLogin = user.LastLogin
	m.CreatedAt = user.CreatedAt
	m.UpdatedAt = user.UpdatedAt
	m.DeletedAt = user.DeletedAt
}
