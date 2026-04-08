package dbErr

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"go-rest-template/internal/services/errors"
)

var constraintToField = map[string]string{
	"idx_users_username_unique": "username",
	"idx_users_email_unique":    "email",
}

func DuplicateFieldError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if field, ok := constraintToField[pgErr.ConstraintName]; ok {
			return svcErr.DuplicateFieldError{Field: field}
		}
	}
	return nil
}
