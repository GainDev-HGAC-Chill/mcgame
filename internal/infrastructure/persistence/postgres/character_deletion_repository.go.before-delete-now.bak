package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CharacterDeletionRepository struct {
	db *Database
}

func NewCharacterDeletionRepository(db *Database) *CharacterDeletionRepository {
	return &CharacterDeletionRepository{db: db}
}

func (r *CharacterDeletionRepository) ScheduleDeletion(ctx context.Context, accountID uuid.UUID, charID int64, at time.Time) (bool, error) {
	var applied bool
	err := r.db.pool.QueryRow(ctx, "select player.schedule_character_deletion($1, $2, $3)", accountID, charID, at).Scan(&applied)
	return applied, err
}

func (r *CharacterDeletionRepository) RestoreDeletion(ctx context.Context, accountID uuid.UUID, charID int64) (bool, error) {
	var restored bool
	err := r.db.pool.QueryRow(ctx, "select player.restore_character_deletion($1, $2)", accountID, charID).Scan(&restored)
	return restored, err
}

func (r *CharacterDeletionRepository) CharacterDeletionTimes(ctx context.Context, accountID uuid.UUID) (map[int64]int64, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_character_deletion_times($1)", accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int64]int64)
	for rows.Next() {
		var id, deleteTime int64
		if err := rows.Scan(&id, &deleteTime); err != nil {
			return nil, err
		}
		result[id] = deleteTime
	}
	return result, rows.Err()
}

func (r *CharacterDeletionRepository) IsDeletionPending(ctx context.Context, accountID uuid.UUID, charID int64) (bool, error) {
	var pending bool
	err := r.db.pool.QueryRow(ctx, "select player.character_deletion_pending($1, $2)", accountID, charID).Scan(&pending)
	return pending, err
}

func (r *CharacterDeletionRepository) PurgeOneExpired(ctx context.Context, accountID uuid.UUID, charID int64) (bool, error) {
	var purged bool
	err := r.db.pool.QueryRow(ctx, "select player.purge_expired_character($1, $2)", accountID, charID).Scan(&purged)
	return purged, err
}
