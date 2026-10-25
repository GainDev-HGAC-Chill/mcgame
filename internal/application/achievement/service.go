// Open-sourced by BaoLT

// Achievement service loads client-facing progress snapshots and registers login hooks.
package achievement

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	appevents "mcgame-server/internal/application/events"
	domainachievement "mcgame-server/internal/domain/achievement"
	"mcgame-server/internal/gamedata"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	repo            domainachievement.Repository
	gameDataManager *gamedata.Manager
	requirements    *RequirementRegistry
	stateReaders    []StateProofReader
	stateOwners     map[int64]string
	logger          *zap.Logger
}

var (
	ErrAchievementNotCompleted  = errors.New("thành tựu chưa hoàn thành")
	ErrAchievementClaimed       = errors.New("phần thưởng thành tựu đã nhận")
	ErrAchievementClaimBusy     = errors.New("phần thưởng thành tựu đang được xử lý")
	ErrAchievementRewardUnknown = errors.New("phần thưởng thành tựu chưa được cấu hình")
)

func NewService(repo domainachievement.Repository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		repo:        repo,
		stateOwners: make(map[int64]string),
		logger:      logger,
	}
}

func (s *Service) SetGameDataManager(manager *gamedata.Manager) {
	s.gameDataManager = manager
	if manager == nil {
		s.requirements = nil
		return
	}
	registry, _, _, err := BuildRequirementRegistry(manager)
	if err != nil {
		s.logger.Error("Failed to build achievement requirement registry", zap.Error(err))
		return
	}
	s.requirements = registry
}

func (s *Service) SetEventBus(bus *appevents.Bus) {
	if bus == nil {
		return
	}

	bus.Register(s)
}

func (s *Service) LoadSnapshot(ctx context.Context, charID int64) (domainachievement.Snapshot, error) {
	snapshot := domainachievement.Snapshot{
		AchieveLog:          map[string]interface{}{},
		AchieveReqLog:       map[string]interface{}{},
		TakeAchieveAwardLog: map[string]interface{}{},
	}

	if s.repo == nil {
		return snapshot, nil
	}

	rows, err := s.repo.GetProgressByCharacter(ctx, charID)
	if err != nil {
		return snapshot, err
	}

	for _, progress := range rows {
		if progress == nil {
			continue
		}

		key := strconv.FormatInt(progress.AchievementID, 10)

		if progress.IsCompleted {
			snapshot.AchieveLog[key] = s.resolveCompletionTimestamp(progress)
			snapshot.AchievementPoints += s.resolveAchievementPoints(progress.AchievementID)
		}

		if progress.IsClaimed {
			snapshot.TakeAchieveAwardLog[key] = 1
		}

	}

	requirements, err := s.repo.ListRequirementProgress(ctx, charID)
	if err != nil {
		return snapshot, err
	}
	for _, progress := range requirements {
		if progress == nil {
			continue
		}
		done := 0
		if progress.Done {
			done = 1
		}
		snapshot.AchieveReqLog[strconv.FormatInt(progress.RequirementID, 10)] = map[string]interface{}{
			"done":     done,
			"progress": progress.Progress,
		}
	}

	return snapshot, nil
}

func (s *Service) ReserveClaim(ctx context.Context, charID, achievementID int64, token uuid.UUID) error {
	if s.repo == nil {
		return fmt.Errorf("achievement repository not configured")
	}
	if _, ok := RewardForAchievement(achievementID); !ok {
		return ErrAchievementRewardUnknown
	}
	reserved, err := s.repo.ReserveClaim(ctx, charID, achievementID, token, time.Now().UTC())
	if err != nil {
		return err
	}
	if !reserved {
		rows, loadErr := s.repo.GetProgressByCharacter(ctx, charID)
		if loadErr != nil {
			return ErrAchievementClaimBusy
		}
		for _, row := range rows {
			if row == nil || row.AchievementID != achievementID {
				continue
			}
			if row.IsClaimed {
				return ErrAchievementClaimed
			}
			if !row.IsCompleted {
				return ErrAchievementNotCompleted
			}
			return ErrAchievementClaimBusy
		}
		return ErrAchievementNotCompleted
	}
	return nil
}

func (s *Service) FinalizeClaim(ctx context.Context, charID, achievementID int64, token uuid.UUID) error {
	updated, err := s.repo.FinalizeClaim(ctx, charID, achievementID, token, time.Now().UTC())
	if err != nil {
		return err
	}
	if updated == nil || !updated.IsClaimed {
		return fmt.Errorf("achievement claim was not finalized")
	}
	return nil
}

func (s *Service) ReleaseClaim(ctx context.Context, charID, achievementID int64, token uuid.UUID) error {
	return s.repo.ReleaseClaim(ctx, charID, achievementID, token)
}

func (s *Service) OnLogin(ctx context.Context, charID int64) error {
	return nil
}

func (s *Service) OnLevelUp(ctx context.Context, charID int64, oldLevel, newLevel int) error {
	_, err := s.RecordLevel(ctx, charID, newLevel, time.Now().UTC())
	return err
}

func (s *Service) resolveCompletionTimestamp(progress *domainachievement.Progress) int64 {
	if progress != nil && progress.CompletedAt != nil {
		timestamp := progress.CompletedAt.UnixMilli()
		if timestamp > 999999 {
			return timestamp
		}
	}

	return 1000000
}

func (s *Service) resolveAchievementPoints(achievementID int64) int {
	if s.gameDataManager == nil {
		return 0
	}

	template := s.gameDataManager.GetAchievement(int(achievementID))
	if template == nil {
		return 0
	}

	return int(template.Award)
}
