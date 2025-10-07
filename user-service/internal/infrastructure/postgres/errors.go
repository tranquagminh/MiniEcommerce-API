package postgres

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateUser  = errors.New("user already exists")
	ErrOptimisticLock = errors.New("record was modified by another process")
	ErrEmailExists    = errors.New("email already exists")
)

func IsDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 is unique_violation error code in PostgreSQL
		return pgErr.Code == "23505"
	}

	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "UNIQUE constraint")
}
