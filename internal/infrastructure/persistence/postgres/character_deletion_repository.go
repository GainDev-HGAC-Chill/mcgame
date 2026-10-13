package postgres

import (
	"context"

	"github.com/google/uuid"
)

type CharacterDeletionRepository struct {
	db *Database
}

func NewCharacterDeletionRepository(db *Database) *CharacterDeletionRepository {
	return &CharacterDeletionRepository{db: db}
}

func (r *CharacterDeletionRepository) DeleteNow(ctx context.Context, accountID uuid.UUID, charID int64) (bool, error) {
	var deleted bool
	err := r.db.pool.QueryRow(ctx, "select player.delete_character_immediately($1, $2)", accountID, charID).Scan(&deleted)
	return deleted, err
}
