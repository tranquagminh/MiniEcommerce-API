package domain

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `json:"id"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	Password  string         `json:"-"`
	FirstName string         `json:"first_name,omitempty"`
	LastName  string         `json:"last_name,omitempty"`
	Phone     string         `json:"phone,omitempty"`
	Gender    string         `json:"gender,omitempty"`
	Birthday  *time.Time     `json:"birthday,omitempty"`
	Role      string         `json:"role"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	IsActive  bool           `json:"is_active"`
	LastLogin *time.Time     `json:"last_login,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *User) IsCustomer() bool {
	return u.Role == "" || u.Role == "customer"
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt.Valid
}

func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return ""
	}
	if u.FirstName == "" {
		return strings.TrimSpace(u.LastName)
	}
	if u.LastName == "" {
		return strings.TrimSpace(u.FirstName)
	}
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}
