package domain

import "time"

type Address struct {
	ID            uint
	UserID        uint
	FullName      string
	Phone         string
	Province      string
	District      string
	Ward          string
	StreetAddress string
	IsDefault     bool
	AddressType   string // home, office, other
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (a *Address) GetFullAddress() string {
	return a.StreetAddress + ", " + a.Ward + ", " + a.District + ", " + a.Province
}
