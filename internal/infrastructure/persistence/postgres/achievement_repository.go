// Open-sourced by BaoLT

// PostgreSQL achievement repository persists per-character progress rows.
// All queries dispatch through schema-qualified player.* functions.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AchievementRepository struct {
	db *Database
}

func (r *AchievementRepository) ListRequirementProgress(ctx context.Context, characterID int64) ([]*domainachievement.RequirementProgress, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_character_achievement_requirements($1)", characterID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to query achievement requirements")
	}
	defer rows.Close()

	result := make([]*domainachievement.RequirementProgress, 0)
	for rows.Next() {
		progress := &domainachievement.RequirementProgress{}
		if err := rows.Scan(
			&progress.CharacterID,
			&progress.RequirementID,
			&progress.AchievementID,
			&progress.Progress,
			&progress.Target,
			&progress.Done,
			&progress.UpdatedAt,
		); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan achievement requirement")
		}
		result = append(result, progress)
	}
	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate achievement requirements")
	}
	return result, nil
}

func (r *AchievementRepository) RecordEvent(ctx context.Context, characterID int64, eventKey string, updates []domainachievement.RequirementUpdate, occurredAt time.Time) (domainachievement.ProgressUpdate, error) {
	result := domainachievement.ProgressUpdate{
		Requirements: []*domainachievement.RequirementProgress{},
		Completed:    []domainachievement.CompletionEntry{},
	}
	encoded, err := json.Marshal(updates)
	if err != nil {
		return result, pkgerrors.Wrap(err, "failed to encode achievement updates")
	}
	var raw []byte
	if err := r.db.pool.QueryRow(ctx,
		"select player.record_character_achievement_event($1, $2, $3::jsonb, $4)",
		characterID,
		eventKey,
		encoded,
		occurredAt,
	).Scan(&raw); err != nil {
		return result, pkgerrors.Wrap(err, "failed to record achievement event")
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, pkgerrors.Wrap(err, "failed to decode achievement event result")
	}
	return result, nil
}

func (r *AchievementRepository) ReserveClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID, reservedAt time.Time) (bool, error) {
	var reserved bool
	if err := r.db.pool.QueryRow(ctx,
		"select player.reserve_character_achievement_claim($1, $2, $3, $4)",
		characterID,
		achievementID,
		token,
		reservedAt,
	).Scan(&reserved); err != nil {
		return false, pkgerrors.Wrap(err, "failed to reserve achievement claim")
	}
	return reserved, nil
}

func (r *AchievementRepository) FinalizeClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID, claimedAt time.Time) (*domainachievement.Progress, error) {
	row := r.db.pool.QueryRow(ctx,
		"select * from player.finalize_character_achievement_claim($1, $2, $3, $4)",
		characterID,
		achievementID,
		token,
		claimedAt,
	)
	return scanAchievementProgress(row)
}

func (r *AchievementRepository) ReleaseClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID) error {
	var released bool
	if err := r.db.pool.QueryRow(ctx,
		"select player.release_character_achievement_claim($1, $2, $3)",
		characterID,
		achievementID,
		token,
	).Scan(&released); err != nil {
		return pkgerrors.Wrap(err, "failed to release achievement claim")
	}
	if !released {
		return fmt.Errorf("achievement claim reservation not found")
	}
	return nil
}

func NewAchievementRepository(db *Database) *AchievementRepository {
	return &AchievementRepository{db: db}
}

func (r *AchievementRepository) GetProgressByCharacter(ctx context.Context, characterID int64) ([]*domainachievement.Progress, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_character_achievements($1)", characterID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to query character achievements")
	}
	defer rows.Close()

	progressRows := make([]*domainachievement.Progress, 0)
	for rows.Next() {
		progress, scanErr := scanAchievementProgress(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		progressRows = append(progressRows, progress)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate character achievements")
	}

	return progressRows, nil
}

func (r *AchievementRepository) UpsertProgress(ctx context.Context, progress *domainachievement.Progress) (*domainachievement.Progress, error) {
	if progress == nil {
		return nil, pkgerrors.ErrInvalidInput
	}

	row := r.db.pool.QueryRow(ctx,
		"select * from player.upsert_character_achievement($1, $2, $3, $4, $5, $6, $7, $8)",
		progress.CharacterID,
		progress.AchievementID,
		progress.Progress,
		progress.Target,
		progress.IsCompleted,
		progress.IsClaimed,
		progress.CompletedAt,
		progress.ClaimedAt,
	)

	return scanAchievementProgress(row)
}

func (r *AchievementRepository) MarkCompleted(ctx context.Context, characterID, achievementID int64, completedAt time.Time) (*domainachievement.Progress, error) {
	row := r.db.pool.QueryRow(ctx,
		"select * from player.mark_character_achievement_completed($1, $2, $3)",
		characterID,
		achievementID,
		completedAt,
	)

	return scanAchievementProgress(row)
}

func (r *AchievementRepository) MarkClaimed(ctx context.Context, characterID, achievementID int64, claimedAt time.Time) (*domainachievement.Progress, error) {
	row := r.db.pool.QueryRow(ctx,
		"select * from player.mark_character_achievement_claimed($1, $2, $3)",
		characterID,
		achievementID,
		claimedAt,
	)

	return scanAchievementProgress(row)
}

func (r *AchievementRepository) ListClaimedCounts(ctx context.Context, characterID int64) (map[int64]int, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_claimed_character_achievements($1)", characterID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to list claimed achievements")
	}
	defer rows.Close()

	claimed := make(map[int64]int)
	for rows.Next() {
		var achievementID int64
		var count int
		if err := rows.Scan(&achievementID, &count); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan claimed achievement")
		}
		claimed[achievementID] = count
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate claimed achievements")
	}

	return claimed, nil
}

type achievementProgressScanner interface {
	Scan(dest ...interface{}) error
}

func scanAchievementProgress(scanner achievementProgressScanner) (*domainachievement.Progress, error) {
	progress := &domainachievement.Progress{}
	var completedAt *time.Time
	var claimedAt *time.Time

	err := scanner.Scan(
		&progress.ID,
		&progress.CharacterID,
		&progress.AchievementID,
		&progress.Progress,
		&progress.Target,
		&progress.IsCompleted,
		&progress.IsClaimed,
		&completedAt,
		&claimedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("achievement progress not found")
		}
		return nil, pkgerrors.Wrap(err, "failed to scan achievement progress")
	}

	progress.CompletedAt = completedAt
	progress.ClaimedAt = claimedAt
	return progress, nil
}
