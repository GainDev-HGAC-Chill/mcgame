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
	encoded, result, err := encodeAchievementUpdates(updates)
	if err != nil {
		return result, err
	}
	return r.scanEventResult(ctx, "select player.record_character_achievement_event($1, $2, $3::jsonb, $4)", characterID, eventKey, encoded, occurredAt)
}

func (r *AchievementRepository) RecordLoginDay(ctx context.Context, characterID int64, updates []domainachievement.RequirementUpdate, occurredAt time.Time) (domainachievement.ProgressUpdate, error) {
	encoded, result, err := encodeAchievementUpdates(updates)
	if err != nil {
		return result, err
	}
	return r.scanEventResult(ctx, "select player.record_character_achievement_login($1, $2::jsonb, $3)", characterID, encoded, occurredAt)
}

func (r *AchievementRepository) ListPendingFacts(ctx context.Context, characterID int64, limit int) ([]domainachievement.Fact, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_pending_character_achievement_facts($1, $2)", characterID, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to list pending achievement facts")
	}
	defer rows.Close()
	facts := make([]domainachievement.Fact, 0)
	for rows.Next() {
		var fact domainachievement.Fact
		if err := rows.Scan(&fact.FactID, &fact.CharacterID, &fact.Kind, &fact.Payload, &fact.OccurredAt); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan achievement fact")
		}
		facts = append(facts, fact)
	}
	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate achievement facts")
	}
	return facts, nil
}

func (r *AchievementRepository) MarkFactProcessed(ctx context.Context, factID string, processedAt time.Time) (bool, error) {
	var processed bool
	if err := r.db.pool.QueryRow(ctx, "select player.mark_character_achievement_fact_processed($1, $2)", factID, processedAt).Scan(&processed); err != nil {
		return false, pkgerrors.Wrap(err, "failed to mark achievement fact processed")
	}
	return processed, nil
}

func encodeAchievementUpdates(updates []domainachievement.RequirementUpdate) ([]byte, domainachievement.ProgressUpdate, error) {
	result := domainachievement.ProgressUpdate{
		Requirements: []*domainachievement.RequirementProgress{},
		Completed:    []domainachievement.CompletionEntry{},
	}
	encoded, err := json.Marshal(updates)
	if err != nil {
		return nil, result, pkgerrors.Wrap(err, "failed to encode achievement updates")
	}
	return encoded, result, nil
}

func (r *AchievementRepository) scanEventResult(ctx context.Context, query string, args ...interface{}) (domainachievement.ProgressUpdate, error) {
	result := domainachievement.ProgressUpdate{
		Requirements: []*domainachievement.RequirementProgress{},
		Completed:    []domainachievement.CompletionEntry{},
	}
	var raw []byte
	if err := r.db.pool.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return result, pkgerrors.Wrap(err, "failed to record achievement event")
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, pkgerrors.Wrap(err, "failed to decode achievement event result")
	}
	return result, nil
}

func (r *AchievementRepository) PrepareRewardGrant(ctx context.Context, characterID, achievementID int64, rewardKind, templateID, quantity int, token uuid.UUID, preparedAt time.Time) (*domainachievement.RewardGrant, error) {
	row := r.db.pool.QueryRow(ctx,
		"select * from player.prepare_character_achievement_reward_grant($1, $2, $3, $4, $5, $6, $7)",
		characterID,
		achievementID,
		rewardKind,
		templateID,
		quantity,
		token,
		preparedAt,
	)
	return scanAchievementRewardGrant(row)
}

func (r *AchievementRepository) MarkRewardGrantGranted(ctx context.Context, characterID, achievementID int64, token uuid.UUID, receipt json.RawMessage, grantedAt time.Time) (*domainachievement.RewardGrant, error) {
	row := r.db.pool.QueryRow(ctx,
		"select * from player.mark_character_achievement_reward_granted($1, $2, $3, $4::jsonb, $5)",
		characterID,
		achievementID,
		token,
		receipt,
		grantedAt,
	)
	return scanAchievementRewardGrant(row)
}

func (r *AchievementRepository) MarkRewardGrantFailed(ctx context.Context, characterID, achievementID int64, token uuid.UUID, reason string, failedAt time.Time) error {
	var marked bool
	if err := r.db.pool.QueryRow(ctx,
		"select player.mark_character_achievement_reward_failed($1, $2, $3, $4, $5)",
		characterID,
		achievementID,
		token,
		reason,
		failedAt,
	).Scan(&marked); err != nil {
		return pkgerrors.Wrap(err, "failed to mark achievement reward grant failed")
	}
	if !marked {
		return fmt.Errorf("achievement reward grant lease not found")
	}
	return nil
}

func (r *AchievementRepository) FinalizeRewardGrant(ctx context.Context, characterID, achievementID int64, finalizedAt time.Time) (*domainachievement.Progress, error) {
	row := r.db.pool.QueryRow(ctx,
		"select * from player.finalize_character_achievement_reward_grant($1, $2, $3)",
		characterID,
		achievementID,
		finalizedAt,
	)
	return scanAchievementProgress(row)
}

func (r *AchievementRepository) ListGrantedRewardGrants(ctx context.Context, characterID int64, limit int) ([]domainachievement.RewardGrant, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_granted_character_achievement_rewards($1, $2)", characterID, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to list granted achievement rewards")
	}
	defer rows.Close()
	grants := make([]domainachievement.RewardGrant, 0)
	for rows.Next() {
		grant, scanErr := scanAchievementRewardGrant(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		grants = append(grants, *grant)
	}
	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate granted achievement rewards")
	}
	return grants, nil
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

func scanAchievementRewardGrant(scanner achievementProgressScanner) (*domainachievement.RewardGrant, error) {
	grant := &domainachievement.RewardGrant{}
	var receipt []byte
	if err := scanner.Scan(
		&grant.CharacterID,
		&grant.AchievementID,
		&grant.ClaimToken,
		&grant.RewardKind,
		&grant.TemplateID,
		&grant.Quantity,
		&grant.Status,
		&receipt,
		&grant.LastError,
		&grant.LeaseExpires,
		&grant.CreatedAt,
		&grant.UpdatedAt,
		&grant.GrantedAt,
		&grant.FinalizedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("achievement reward grant not available")
		}
		return nil, pkgerrors.Wrap(err, "failed to scan achievement reward grant")
	}
	grant.Receipt = receipt
	return grant, nil
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
