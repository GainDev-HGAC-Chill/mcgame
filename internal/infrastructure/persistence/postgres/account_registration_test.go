package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	pkgerrors "mcgame-server/pkg/errors"
)

func TestAccountCreateResult(t *testing.T) {
	duplicate := fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "23505", ConstraintName: "accounts_username_key"})
	otherConstraint := &pgconn.PgError{Code: "23505", ConstraintName: "accounts_pkey"}
	otherCode := &pgconn.PgError{Code: "23503", ConstraintName: "accounts_username_key"}
	other := errors.New("unavailable")
	for _, tc := range []struct {
		name        string
		created     bool
		err, target error
	}{
		{"created", true, nil, nil},
		{"false", false, nil, pkgerrors.ErrSystemError},
		{"username race", false, duplicate, pkgerrors.ErrAlreadyExists},
		{"other constraint", false, otherConstraint, otherConstraint},
		{"other code", false, otherCode, otherCode},
		{"other failure", false, other, other},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := accountCreateResult(tc.created, tc.err)
			if !errors.Is(err, tc.target) {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.target != pkgerrors.ErrAlreadyExists && errors.Is(err, pkgerrors.ErrAlreadyExists) {
				t.Fatal("unrelated failure swallowed")
			}
		})
	}
}
