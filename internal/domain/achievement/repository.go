// Open-sourced by BaoLT

// Achievement repository ports expose per-character achievement persistence.
package achievement

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	GetProgressByCharacter(ctx context.Context, characterID int64) ([]*Progress, error)
	UpsertProgress(ctx context.Context, progress *Progress) (*Progress, error)
	MarkCompleted(ctx context.Context, characterID, achievementID int64, completedAt time.Time) (*Progress, error)
	MarkClaimed(ctx context.Context, characterID, achievementID int64, claimedAt time.Time) (*Progress, error)
	ListClaimedCounts(ctx context.Context, characterID int64) (map[int64]int, error)
	ListRequirementProgress(ctx context.Context, characterID int64) ([]*RequirementProgress, error)
	RecordEvent(ctx context.Context, characterID int64, eventKey string, updates []RequirementUpdate, occurredAt time.Time) (ProgressUpdate, error)
	RecordLoginDay(ctx context.Context, characterID int64, updates []RequirementUpdate, occurredAt time.Time) (ProgressUpdate, error)
	ListPendingFacts(ctx context.Context, characterID int64, limit int) ([]Fact, error)
	MarkFactProcessed(ctx context.Context, factID string, processedAt time.Time) (bool, error)
	PrepareRewardGrant(ctx context.Context, characterID, achievementID int64, rewardKind, templateID, quantity int, token uuid.UUID, preparedAt time.Time) (*RewardGrant, error)
	MarkRewardGrantGranted(ctx context.Context, characterID, achievementID int64, token uuid.UUID, receipt json.RawMessage, grantedAt time.Time) (*RewardGrant, error)
	MarkRewardGrantFailed(ctx context.Context, characterID, achievementID int64, token uuid.UUID, reason string, failedAt time.Time) error
	FinalizeRewardGrant(ctx context.Context, characterID, achievementID int64, finalizedAt time.Time) (*Progress, error)
	ListGrantedRewardGrants(ctx context.Context, characterID int64, limit int) ([]RewardGrant, error)
}
