// Open-sourced by BaoLT

// Achievement repository ports expose per-character achievement persistence.
package achievement

import (
	"context"
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
	ReserveClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID, reservedAt time.Time) (bool, error)
	FinalizeClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID, claimedAt time.Time) (*Progress, error)
	ReleaseClaim(ctx context.Context, characterID, achievementID int64, token uuid.UUID) error
}
