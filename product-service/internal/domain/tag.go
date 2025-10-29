package domain

import "time"

type Tag struct {
	ID        uint
	Name      string
	Slug      string
	CreatedAt time.Time
}
