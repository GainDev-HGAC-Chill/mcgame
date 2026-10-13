package quest

import (
	"context"

	"go.uber.org/zap"
)

type rewardNotificationKey struct{}

type rewardNotificationScope struct {
	service     *Service
	characterID int64
	items       map[int]int
}

func (s *Service) deferRewardItemNotifications(ctx context.Context, characterID int64) (context.Context, func()) {
	scope := &rewardNotificationScope{service: s, characterID: characterID, items: make(map[int]int)}
	return context.WithValue(ctx, rewardNotificationKey{}, scope), func() {
		for templateID, amount := range scope.items {
			if _, err := s.OnItemCollected(ctx, characterID, templateID, amount); err != nil {
				s.logger.Warn("Failed to refresh quests after reward items", zap.Int64("character_id", characterID), zap.Int("template_id", templateID), zap.Error(err))
			}
		}
	}
}
