// Wedding booking, hall-entry, and divorce use cases.
package marriage

import (
	"context"
	"errors"
	"time"

	domainmarriage "mcgame-server/internal/domain/marriage"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	weddingOpenFlag       = 0
	weddingInvitationFlag = 1
)

var (
	ErrWeddingAlreadyBooked = errors.New("wedding already booked")
	ErrWeddingSlotTaken     = errors.New("wedding slot taken")
	ErrWeddingNotBooked     = errors.New("wedding not booked")
	ErrWeddingNotOpen       = errors.New("wedding hall not open")
	ErrWeddingWrongLine     = errors.New("wedding booked on another line")
	ErrWeddingInvitation    = errors.New("wedding invitation required")
	ErrNotMarried           = errors.New("character is not married")
	ErrDivorceCardRequired  = errors.New("divorce card required")
)

type weddingTier struct {
	Fee          int64
	InvitationID int
	MapID        int
}

var weddingTiers = map[int]weddingTier{
	1: {Fee: 2990000, InvitationID: 1691, MapID: 64},
	2: {Fee: 3990000, InvitationID: 1692, MapID: 63},
	3: {Fee: 5000000, InvitationID: 1693, MapID: 65},
}

type InvitationCounter interface {
	CountCarriedItemsByTemplateID(ctx context.Context, characterID int64, templateID int) (int, error)
}

type WeddingEntry struct {
	MapID int
}

func (s *Service) SetInvitationCounter(counter InvitationCounter) {
	s.invitationCounter = counter
}

func (s *Service) BookWeddingHall(ctx context.Context, characterID int64, line, hour, minute, flag int, now time.Time) ([]map[string]interface{}, error) {
	if line <= 0 || hour < 0 || hour > 23 || minute < 0 || minute > 59 || (flag != weddingOpenFlag && flag != weddingInvitationFlag) {
		return nil, pkgerrors.ErrInvalidArgs
	}
	marriage, err := s.repo.FindByPartner(ctx, characterID)
	if err != nil {
		return nil, err
	}
	tier, ok := weddingTiers[marriage.RingType]
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	location := weddingLocation()
	localNow := now.In(location)
	weddingDate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, location)
	if weddingDate.Before(localNow.Add(10 * time.Minute)) {
		return nil, pkgerrors.ErrInvalidInput
	}
	_, err = s.repo.BookWedding(ctx, domainmarriage.BookWeddingCommand{
		CharacterID: characterID,
		WeddingDate: weddingDate,
		Line:        line,
		Flag:        flag,
		Venue:       tier.MapID,
		Cost:        tier.Fee,
	})
	if err != nil {
		return nil, err
	}
	return s.WeddingBookList(ctx, now)
}

func (s *Service) CancelWeddingHall(ctx context.Context, characterID int64, now time.Time) ([]map[string]interface{}, error) {
	if err := s.repo.CancelWedding(ctx, characterID); err != nil {
		return nil, err
	}
	return s.WeddingBookList(ctx, now)
}

func (s *Service) WeddingBookList(ctx context.Context, now time.Time) ([]map[string]interface{}, error) {
	location := weddingLocation()
	localNow := now.In(location)
	from := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	to := from.AddDate(0, 0, 1)
	bookings, err := s.repo.ListWeddingBookings(ctx, from, to)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, map[string]interface{}{
			"cid":   booking.Partner1ID,
			"pid":   booking.Partner2ID,
			"cname": booking.Partner1Name,
			"pname": booking.Partner2Name,
			"date":  booking.WeddingDate.UnixMilli(),
			"line":  booking.Line,
			"flag":  booking.Flag,
		})
	}
	return result, nil
}

func (s *Service) EnterWeddingHall(ctx context.Context, characterID, groomID int64, currentLine int, now time.Time) (*WeddingEntry, error) {
	booking, err := s.repo.FindWeddingByGroom(ctx, groomID)
	if err != nil {
		return nil, err
	}
	if booking.Line != currentLine {
		return nil, ErrWeddingWrongLine
	}
	if now.Before(booking.WeddingDate.Add(-10*time.Minute)) || now.After(booking.WeddingDate.Add(60*time.Minute)) {
		return nil, ErrWeddingNotOpen
	}
	if characterID != booking.Partner1ID && characterID != booking.Partner2ID && booking.Flag == weddingInvitationFlag {
		if s.invitationCounter == nil {
			return nil, ErrWeddingInvitation
		}
		tier, ok := weddingTiers[booking.RingType]
		if !ok {
			return nil, pkgerrors.ErrInvalidInput
		}
		count, countErr := s.invitationCounter.CountCarriedItemsByTemplateID(ctx, characterID, tier.InvitationID)
		if countErr != nil {
			return nil, countErr
		}
		if count <= 0 {
			return nil, ErrWeddingInvitation
		}
	}
	tier, ok := weddingTiers[booking.RingType]
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	return &WeddingEntry{MapID: tier.MapID}, nil
}

func (s *Service) DivorceWithCard(ctx context.Context, characterID int64) (*domainmarriage.DivorceResult, error) {
	return s.repo.DivorceWithCard(ctx, characterID)
}

func weddingLocation() *time.Location {
	return time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
}
