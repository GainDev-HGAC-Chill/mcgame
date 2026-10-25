// Open-sourced by BaoLT

// Daily-activity state reader proves the current day's persisted vitality milestone.
package achievement

import (
	"context"
	"fmt"
	"time"

	domaindailyact "mcgame-server/internal/domain/dailyact"
)

type DailyActStateRepository interface {
	Get(ctx context.Context, characterID int64, day time.Time) (*domaindailyact.DailyActRecord, error)
}

type DailyActStateReader struct {
	repo DailyActStateRepository
	now  func() time.Time
}

func NewDailyActStateReader(repo DailyActStateRepository) *DailyActStateReader {
	return &DailyActStateReader{repo: repo, now: time.Now}
}

func (r *DailyActStateReader) Name() string {
	return "daily-act"
}

func (r *DailyActStateReader) RequirementIDs() []int64 {
	return []int64{421}
}

func (r *DailyActStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.now == nil {
		return nil, fmt.Errorf("daily-act state dependencies not configured")
	}
	now := r.now()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	record, err := r.repo.Get(ctx, characterID, day)
	if err != nil {
		return nil, err
	}
	if record == nil || record.CharacterID != characterID || !sameCalendarDay(record.Day, day) {
		return nil, fmt.Errorf("invalid daily-act state for character %d", characterID)
	}
	if record.VitalityPoints < 0 {
		return nil, fmt.Errorf("negative daily-act vitality for character %d", characterID)
	}
	return []StateProof{stateProof(421, record.VitalityPoints, 720)}, nil
}

func sameCalendarDay(left, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}
