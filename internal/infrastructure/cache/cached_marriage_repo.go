// Cached marriage repository coordinates database-owned commands with player write-behind state.
package cache

import (
	"context"
	"time"

	domainmarriage "mcgame-server/internal/domain/marriage"
	pkgerrors "mcgame-server/pkg/errors"
)

type CachedMarriageRepository struct {
	cache    *PlayerCache
	delegate domainmarriage.Repository
}

func NewCachedMarriageRepository(cache *PlayerCache, delegate domainmarriage.Repository) *CachedMarriageRepository {
	return &CachedMarriageRepository{cache: cache, delegate: delegate}
}

func (r *CachedMarriageRepository) FindByPartner(ctx context.Context, characterID int64) (*domainmarriage.MarriageRecord, error) {
	return r.delegate.FindByPartner(ctx, characterID)
}

func (r *CachedMarriageRepository) Create(ctx context.Context, record *domainmarriage.MarriageRecord) error {
	return r.delegate.Create(ctx, record)
}

func (r *CachedMarriageRepository) ListRanks(ctx context.Context, offset, limit int) ([]*domainmarriage.CoupleRank, error) {
	return r.delegate.ListRanks(ctx, offset, limit)
}

func (r *CachedMarriageRepository) ListWeddingBookings(ctx context.Context, from, to time.Time) ([]*domainmarriage.WeddingBooking, error) {
	return r.delegate.ListWeddingBookings(ctx, from, to)
}

func (r *CachedMarriageRepository) BookWedding(ctx context.Context, command domainmarriage.BookWeddingCommand) (*domainmarriage.BookWeddingResult, error) {
	var result *domainmarriage.BookWeddingResult
	err := r.cache.runPlayerCommand(ctx, command.CharacterID, func() error {
		var commandErr error
		result, commandErr = r.delegate.BookWedding(ctx, command)
		return commandErr
	}, func(data *PlayerData) {
		if result != nil {
			data.syncCommittedMarriageMoney(result.Money)
		}
	})
	return result, err
}

func (r *CachedMarriageRepository) CancelWedding(ctx context.Context, characterID int64) error {
	return r.delegate.CancelWedding(ctx, characterID)
}

func (r *CachedMarriageRepository) FindWeddingByGroom(ctx context.Context, groomID int64) (*domainmarriage.WeddingBooking, error) {
	return r.delegate.FindWeddingByGroom(ctx, groomID)
}

func (r *CachedMarriageRepository) DivorceWithCard(ctx context.Context, characterID int64) (*domainmarriage.DivorceResult, error) {
	var result *domainmarriage.DivorceResult
	err := r.cache.runPlayerCommand(ctx, characterID, func() error {
		var commandErr error
		result, commandErr = r.delegate.DivorceWithCard(ctx, characterID)
		return commandErr
	}, func(data *PlayerData) {
		if result != nil {
			data.syncCommittedItemCount(result.ItemID, result.Remaining)
		}
	})
	return result, err
}

func (r *CachedMarriageRepository) GetMarriageAchievementState(ctx context.Context, characterID int64) (int, int, int, error) {
	reader, ok := r.delegate.(interface {
		GetMarriageAchievementState(context.Context, int64) (int, int, int, error)
	})
	if !ok {
		return 0, 0, 0, pkgerrors.ErrSystemError
	}
	return reader.GetMarriageAchievementState(ctx, characterID)
}

var _ domainmarriage.Repository = (*CachedMarriageRepository)(nil)
