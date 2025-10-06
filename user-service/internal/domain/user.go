package domain

import (
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
	LastLogin *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt.Valid
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}
