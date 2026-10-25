// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
	"time"

	domaindailyact "mcgame-server/internal/domain/dailyact"
)

type dailyActStateRepositoryFixture struct {
	record *domaindailyact.DailyActRecord
}

func (r dailyActStateRepositoryFixture) Get(ctx context.Context, characterID int64, day time.Time) (*domaindailyact.DailyActRecord, error) {
	return r.record, nil
}

func TestDailyActStateReaderUsesCurrentDayVitality(t *testing.T) {
	now := time.Date(2026, time.October, 26, 15, 30, 0, 0, time.FixedZone("test", 7*60*60))
	reader := NewDailyActStateReader(dailyActStateRepositoryFixture{record: &domaindailyact.DailyActRecord{
		CharacterID:    99,
		Day:            time.Date(2026, time.October, 26, 0, 0, 0, 0, now.Location()),
		VitalityPoints: 720,
	}})
	reader.now = func() time.Time { return now }
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 1 || !proofs[0].Done || proofs[0].Progress != 720 || proofs[0].Target != 720 {
		t.Fatalf("proofs = %#v, want completed 720/720", proofs)
	}
}

func TestDailyActStateReaderRejectsWrongDay(t *testing.T) {
	now := time.Date(2026, time.October, 26, 15, 30, 0, 0, time.Local)
	reader := NewDailyActStateReader(dailyActStateRepositoryFixture{record: &domaindailyact.DailyActRecord{
		CharacterID: 99,
		Day:         now.AddDate(0, 0, -1),
	}})
	reader.now = func() time.Time { return now }
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want wrong-day error")
	}
}
