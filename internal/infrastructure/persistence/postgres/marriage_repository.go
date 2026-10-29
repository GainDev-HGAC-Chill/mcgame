// Open-sourced by BaoLT

// PostgreSQL implementation of persisted marriage data.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	appmarriage "mcgame-server/internal/application/marriage"
	domainmarriage "mcgame-server/internal/domain/marriage"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MarriageRepository struct {
	db *Database
}

func NewMarriageRepository(db *Database) *MarriageRepository {
	return &MarriageRepository{db: db}
}

func (r *MarriageRepository) GetMarriageAchievementState(ctx context.Context, characterID int64) (int, int, int, error) {
	var normalWeddings, marriageCount, currentMarriedDays int
	err := r.db.pool.QueryRow(ctx, "select * from player.get_achievement_marriage_state($1)", characterID).Scan(
		&normalWeddings,
		&marriageCount,
		&currentMarriedDays,
	)
	if err != nil {
		return 0, 0, 0, pkgerrors.Wrap(err, "failed to get achievement marriage state")
	}
	return normalWeddings, marriageCount, currentMarriedDays, nil
}

func (r *MarriageRepository) FindByPartner(ctx context.Context, characterID int64) (*domainmarriage.MarriageRecord, error) {
	record := &domainmarriage.MarriageRecord{}
	err := r.db.pool.QueryRow(ctx, "select * from player.find_marriage_by_partner($1)", characterID).Scan(
		&record.ID,
		&record.Partner1ID,
		&record.Partner2ID,
		&record.RingType,
		&record.Intimacy,
		&record.MarriedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pkgerrors.ErrNotFound
		}
		return nil, pkgerrors.Wrap(err, "failed to find marriage by partner")
	}

	return record, nil
}

func (r *MarriageRepository) Create(ctx context.Context, record *domainmarriage.MarriageRecord) error {
	err := r.db.pool.QueryRow(
		ctx,
		"select * from player.create_marriage($1, $2, $3, $4, $5)",
		record.Partner1ID,
		record.Partner2ID,
		record.RingType,
		record.Intimacy,
		time.Now(),
	).Scan(&record.ID, &record.MarriedAt)
	if err != nil {
		return mapWeddingCommandError(err)
	}

	return nil
}

func (r *MarriageRepository) ListRanks(ctx context.Context, offset, limit int) ([]*domainmarriage.CoupleRank, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_marriage_ranks($1, $2)", offset, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to list couple ranks")
	}
	defer rows.Close()

	result := make([]*domainmarriage.CoupleRank, 0)
	for rows.Next() {
		rank := &domainmarriage.CoupleRank{}
		if err := rows.Scan(
			&rank.ID,
			&rank.Partner1ID,
			&rank.Partner1Name,
			&rank.Partner2ID,
			&rank.Partner2Name,
			&rank.RingType,
			&rank.Intimacy,
			&rank.MarriedAt,
		); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan couple rank")
		}
		result = append(result, rank)
	}

	return result, nil
}

func (r *MarriageRepository) ListWeddingBookings(ctx context.Context, from, to time.Time) ([]*domainmarriage.WeddingBooking, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_wedding_bookings($1, $2)", from, to)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to list wedding bookings")
	}
	defer rows.Close()
	result := make([]*domainmarriage.WeddingBooking, 0)
	for rows.Next() {
		booking := &domainmarriage.WeddingBooking{}
		if err := rows.Scan(&booking.MarriageID, &booking.Partner1ID, &booking.Partner1Name, &booking.Partner2ID, &booking.Partner2Name, &booking.RingType, &booking.WeddingDate, &booking.Line, &booking.Flag); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan wedding booking")
		}
		result = append(result, booking)
	}
	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to read wedding bookings")
	}
	return result, nil
}

func (r *MarriageRepository) BookWedding(ctx context.Context, command domainmarriage.BookWeddingCommand) (*domainmarriage.BookWeddingResult, error) {
	result := &domainmarriage.BookWeddingResult{}
	err := r.db.pool.QueryRow(ctx, "select * from player.book_wedding_hall($1, $2, $3, $4, $5, $6)", command.CharacterID, command.WeddingDate, command.Line, command.Flag, command.Venue, command.Cost).Scan(&result.Money)
	if err != nil {
		return nil, mapWeddingCommandError(err)
	}
	return result, nil
}

func (r *MarriageRepository) CancelWedding(ctx context.Context, characterID int64) error {
	var cancelled bool
	if err := r.db.pool.QueryRow(ctx, "select player.cancel_wedding_hall($1)", characterID).Scan(&cancelled); err != nil {
		return mapWeddingCommandError(err)
	}
	if !cancelled {
		return appmarriage.ErrWeddingNotBooked
	}
	return nil
}

func (r *MarriageRepository) FindWeddingByGroom(ctx context.Context, groomID int64) (*domainmarriage.WeddingBooking, error) {
	booking := &domainmarriage.WeddingBooking{}
	err := r.db.pool.QueryRow(ctx, "select * from player.get_wedding_booking_by_groom($1)", groomID).Scan(
		&booking.MarriageID, &booking.Partner1ID, &booking.Partner1Name, &booking.Partner2ID, &booking.Partner2Name,
		&booking.RingType, &booking.WeddingDate, &booking.Line, &booking.Flag,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appmarriage.ErrWeddingNotBooked
		}
		return nil, pkgerrors.Wrap(err, "failed to find wedding booking")
	}
	return booking, nil
}

func (r *MarriageRepository) DivorceWithCard(ctx context.Context, characterID int64) (*domainmarriage.DivorceResult, error) {
	result := &domainmarriage.DivorceResult{}
	err := r.db.pool.QueryRow(ctx, "select * from player.divorce_with_card($1)", characterID).Scan(
		&result.MarriageID, &result.Partner1ID, &result.Partner2ID, &result.ItemID, &result.Remaining,
	)
	if err != nil {
		return nil, mapWeddingCommandError(err)
	}
	return result, nil
}

func mapWeddingCommandError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Message {
		case "NOT_MARRIED":
			return appmarriage.ErrNotMarried
		case "DIVORCE_CARD_REQUIRED":
			return appmarriage.ErrDivorceCardRequired
		case "WEDDING_ALREADY_BOOKED":
			return appmarriage.ErrWeddingAlreadyBooked
		case "WEDDING_SLOT_TAKEN":
			return appmarriage.ErrWeddingSlotTaken
		case "WEDDING_NOT_BOOKED":
			return appmarriage.ErrWeddingNotBooked
		case "INSUFFICIENT_FUNDS":
			return pkgerrors.ErrInsufficientFunds
		case "INVALID_WEDDING_COMMAND":
			return pkgerrors.ErrInvalidInput
		case "MARRIAGE_ACTIVE":
			return pkgerrors.ErrAlreadyExists
		case "CHARACTER_NOT_FOUND":
			return pkgerrors.ErrCharacterNotFound
		case "INVALID_MARRIAGE_COMMAND":
			return pkgerrors.ErrInvalidInput
		}
	}
	return fmt.Errorf("wedding command failed: %w", err)
}
