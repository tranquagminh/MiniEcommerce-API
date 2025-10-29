package postgres

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrVariantNotFound  = errors.New("variant not found")
	ErrTagNotFound      = errors.New("tag not found")
	ErrDuplicateProduct = errors.New("product already exists")
	ErrDuplicateSKU     = errors.New("SKU already exists")
	ErrDuplicateSlug    = errors.New("slug already exists")
)

func IsDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "UNIQUE constraint")
}
