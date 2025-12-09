package domain

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint
	Username  string
	Email     string
	Password  string
	FirstName string
	LastName  string
	Phone     string
	Gender    string
	Birthday  *time.Time
	LastLogin *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
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
