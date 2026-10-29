package marriage

import (
	"context"
	"errors"
	"testing"
	"time"

	domainmarriage "mcgame-server/internal/domain/marriage"
)

type weddingTestRepo struct {
	marriage  *domainmarriage.MarriageRecord
	bookings  []*domainmarriage.WeddingBooking
	command   domainmarriage.BookWeddingCommand
	bookErr   error
	cancelErr error
	entry     *domainmarriage.WeddingBooking
	divorce   *domainmarriage.DivorceResult
}

func (r *weddingTestRepo) FindByPartner(context.Context, int64) (*domainmarriage.MarriageRecord, error) {
	if r.marriage == nil {
		return nil, ErrNotMarried
	}
	return r.marriage, nil
}

func (r *weddingTestRepo) Create(context.Context, *domainmarriage.MarriageRecord) error { return nil }

func (r *weddingTestRepo) ListRanks(context.Context, int, int) ([]*domainmarriage.CoupleRank, error) {
	return nil, nil
}

func (r *weddingTestRepo) ListWeddingBookings(context.Context, time.Time, time.Time) ([]*domainmarriage.WeddingBooking, error) {
	return r.bookings, nil
}

func (r *weddingTestRepo) BookWedding(_ context.Context, command domainmarriage.BookWeddingCommand) (*domainmarriage.BookWeddingResult, error) {
	r.command = command
	if r.bookErr != nil {
		return nil, r.bookErr
	}
	return &domainmarriage.BookWeddingResult{Money: 1}, nil
}

func (r *weddingTestRepo) CancelWedding(context.Context, int64) error { return r.cancelErr }

func (r *weddingTestRepo) FindWeddingByGroom(context.Context, int64) (*domainmarriage.WeddingBooking, error) {
	if r.entry == nil {
		return nil, ErrWeddingNotBooked
	}
	return r.entry, nil
}

func (r *weddingTestRepo) DivorceWithCard(context.Context, int64) (*domainmarriage.DivorceResult, error) {
	return r.divorce, nil
}

type weddingInvitationCounter struct {
	count      int
	templateID int
}

func (c *weddingInvitationCounter) CountCarriedItemsByTemplateID(_ context.Context, _ int64, templateID int) (int, error) {
	c.templateID = templateID
	return c.count, nil
}

func TestBookWeddingHallUsesPersistedTierAndCurrentFlashFee(t *testing.T) {
	now := time.Date(2026, time.October, 29, 10, 0, 0, 0, weddingLocation())
	repo := &weddingTestRepo{
		marriage: &domainmarriage.MarriageRecord{RingType: 2},
		bookings: []*domainmarriage.WeddingBooking{{Partner1ID: 1, Partner1Name: "A", Partner2ID: 2, Partner2Name: "B", WeddingDate: now.Add(15 * time.Minute), Line: 3, Flag: 1}},
	}
	service := NewService(nil, repo, nil, nil)
	rows, err := service.BookWeddingHall(context.Background(), 1, 3, 10, 15, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if repo.command.Cost != 3990000 || repo.command.Venue != 63 || repo.command.Line != 3 || repo.command.Flag != 1 {
		t.Fatalf("command = %#v", repo.command)
	}
	if len(rows) != 1 || rows[0]["cid"] != int64(1) || rows[0]["pid"] != int64(2) || rows[0]["date"] != now.Add(15*time.Minute).UnixMilli() {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestBookWeddingHallRejectsTooSoonAndInvalidFlag(t *testing.T) {
	now := time.Date(2026, time.October, 29, 10, 0, 0, 0, weddingLocation())
	repo := &weddingTestRepo{marriage: &domainmarriage.MarriageRecord{RingType: 1}}
	service := NewService(nil, repo, nil, nil)
	if _, err := service.BookWeddingHall(context.Background(), 1, 1, 10, 9, 0, now); err == nil {
		t.Fatal("expected lead-time rejection")
	}
	if _, err := service.BookWeddingHall(context.Background(), 1, 1, 10, 15, 2, now); err == nil {
		t.Fatal("expected flag rejection")
	}
}

func TestEnterWeddingHallEnforcesWindowLineAndInvitation(t *testing.T) {
	start := time.Date(2026, time.October, 29, 11, 0, 0, 0, weddingLocation())
	repo := &weddingTestRepo{entry: &domainmarriage.WeddingBooking{Partner1ID: 1, Partner2ID: 2, RingType: 3, WeddingDate: start, Line: 4, Flag: 1}}
	counter := &weddingInvitationCounter{}
	service := NewService(nil, repo, nil, nil)
	service.SetInvitationCounter(counter)

	if _, err := service.EnterWeddingHall(context.Background(), 9, 1, 3, start); !errors.Is(err, ErrWeddingWrongLine) {
		t.Fatalf("wrong-line error = %v", err)
	}
	if _, err := service.EnterWeddingHall(context.Background(), 9, 1, 4, start.Add(-11*time.Minute)); !errors.Is(err, ErrWeddingNotOpen) {
		t.Fatalf("early error = %v", err)
	}
	if _, err := service.EnterWeddingHall(context.Background(), 9, 1, 4, start); !errors.Is(err, ErrWeddingInvitation) {
		t.Fatalf("invitation error = %v", err)
	}
	counter.count = 1
	entry, err := service.EnterWeddingHall(context.Background(), 9, 1, 4, start.Add(60*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if entry.MapID != 65 || counter.templateID != 1693 {
		t.Fatalf("entry = %#v, invitation = %d", entry, counter.templateID)
	}
	if _, err := service.EnterWeddingHall(context.Background(), 9, 1, 4, start.Add(60*time.Minute+time.Nanosecond)); !errors.Is(err, ErrWeddingNotOpen) {
		t.Fatalf("late error = %v", err)
	}
}

func TestEnterWeddingHallAllowsSpouseWithoutInvitation(t *testing.T) {
	start := time.Now()
	repo := &weddingTestRepo{entry: &domainmarriage.WeddingBooking{Partner1ID: 1, Partner2ID: 2, RingType: 1, WeddingDate: start, Line: 1, Flag: 1}}
	service := NewService(nil, repo, nil, nil)
	entry, err := service.EnterWeddingHall(context.Background(), 2, 1, 1, start)
	if err != nil || entry.MapID != 64 {
		t.Fatalf("entry = %#v, error = %v", entry, err)
	}
}
