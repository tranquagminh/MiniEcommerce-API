package postgres

import (
	"time"
	"user-service/internal/domain"
)

type AddressModel struct {
	ID            uint      `gorm:"primaryKey"`
	UserID        uint      `gorm:"not null;index" json:"user_id"`
	FullName      string    `gorm:"size:100;not null" json:"full_name"`
	Phone         string    `gorm:"size:20;not null" json:"phone"`
	Province      string    `gorm:"size:100;not null" json:"province"`
	District      string    `gorm:"size:100;not null" json:"district"`
	Ward          string    `gorm:"size:100;not null" json:"ward"`
	StreetAddress string    `gorm:"size:255;not null" json:"street_address"`
	IsDefault     bool      `gorm:"default:false" json:"is_default"`
	AddressType   string    `gorm:"size:20;default:home" json:"address_type"` // home, office, other
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (AddressModel) TableName() string {
	return "addresses"
}

func (m *AddressModel) ToDomain() *domain.Address {
	return &domain.Address{
		ID:            m.ID,
		UserID:        m.UserID,
		FullName:      m.FullName,
		Phone:         m.Phone,
		Province:      m.Province,
		District:      m.District,
		Ward:          m.Ward,
		StreetAddress: m.StreetAddress,
		IsDefault:     m.IsDefault,
		AddressType:   m.AddressType,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func (m *AddressModel) FromDomain(addr *domain.Address) {
	m.ID = addr.ID
	m.UserID = addr.UserID
	m.FullName = addr.FullName
	m.Phone = addr.Phone
	m.Province = addr.Province
	m.District = addr.District
	m.Ward = addr.Ward
	m.StreetAddress = addr.StreetAddress
	m.IsDefault = addr.IsDefault
	m.AddressType = addr.AddressType
	m.CreatedAt = addr.CreatedAt
	m.UpdatedAt = addr.UpdatedAt
}
