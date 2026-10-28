// Send Combine service keeps durable purchase commands behind an application boundary.
package activity

import (
	"context"
	"errors"

	domainactivity "mcgame-server/internal/domain/activity"
	pkerrors "mcgame-server/pkg/errors"
)

var (
	ErrSendCombineExpired      = errors.New("send combine activity expired")
	ErrSendCombineLimitReached = errors.New("send combine purchase limit reached")
)

type SendCombineService struct {
	repository domainactivity.SendCombineRepository
}

func NewSendCombineService(repository domainactivity.SendCombineRepository) *SendCombineService {
	return &SendCombineService{repository: repository}
}

func (s *SendCombineService) Purchase(ctx context.Context, command domainactivity.SendCombinePurchaseCommand) (*domainactivity.SendCombinePurchaseResult, error) {
	if s == nil || s.repository == nil || command.CharacterID <= 0 || command.ActivityID == "" || command.EventEndMs <= 0 || command.Limit <= 0 || command.Cost <= 0 || len(command.Awards) == 0 {
		return nil, pkerrors.ErrInvalidInput
	}
	for _, award := range command.Awards {
		if award.TemplateID <= 0 || award.ItemType <= 0 || award.Count <= 0 {
			return nil, pkerrors.ErrInvalidInput
		}
	}
	return s.repository.PurchaseSendCombine(ctx, command)
}

func (s *SendCombineService) BuyCounts(ctx context.Context, characterID int64, eventEndMs int64) (map[string]int, error) {
	if s == nil || s.repository == nil || characterID <= 0 || eventEndMs <= 0 {
		return nil, pkerrors.ErrInvalidInput
	}
	return s.repository.ListSendCombineBuyCounts(ctx, characterID, eventEndMs)
}
