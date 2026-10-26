// Open-sourced by BaoLT

// Achievement service loads client-facing progress snapshots and registers login hooks.
package achievement

import (
	"context"
	"encoding/json"
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

func (s *Service) PrepareClaim(ctx context.Context, charID, achievementID int64, token uuid.UUID) (*domainachievement.RewardGrant, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("achievement repository not configured")
	}
	reward, ok := RewardForAchievement(achievementID)
	if !ok {
		return nil, ErrAchievementRewardUnknown
	}
	grant, err := s.repo.PrepareRewardGrant(ctx, charID, achievementID, int(reward.Kind), reward.TemplateID, reward.Quantity, token, time.Now().UTC())
	if err != nil {
		rows, loadErr := s.repo.GetProgressByCharacter(ctx, charID)
		if loadErr != nil {
			return nil, err
		}
		for _, row := range rows {
			if row == nil || row.AchievementID != achievementID {
				continue
			}
			if row.IsClaimed {
				return nil, ErrAchievementClaimed
			}
			if !row.IsCompleted {
				return nil, ErrAchievementNotCompleted
			}
			return nil, err
		}
		return nil, ErrAchievementNotCompleted
	}
	if grant == nil {
		return nil, ErrAchievementClaimBusy
	}
	if grant.Status == domainachievement.RewardGrantGranting && grant.ClaimToken != token {
		return nil, ErrAchievementClaimBusy
	}
	return grant, nil
}

func (s *Service) MarkClaimGranted(ctx context.Context, charID, achievementID int64, token uuid.UUID, receipt domainachievement.RewardReceipt) error {
	if err := ValidateRewardReceipt(achievementID, receipt); err != nil {
		return err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	grant, err := s.repo.MarkRewardGrantGranted(ctx, charID, achievementID, token, raw, time.Now().UTC())
	if err != nil {
		return err
	}
	if grant == nil || grant.Status != domainachievement.RewardGrantGranted {
		return fmt.Errorf("achievement reward grant receipt was not persisted")
	}
	return nil
}

func (s *Service) MarkClaimFailed(ctx context.Context, charID, achievementID int64, token uuid.UUID, cause error) error {
	reason := "reward grant failed"
	if cause != nil {
		reason = cause.Error()
	}
	return s.repo.MarkRewardGrantFailed(ctx, charID, achievementID, token, reason, time.Now().UTC())
}

func (s *Service) FinalizeClaim(ctx context.Context, charID, achievementID int64) error {
	updated, err := s.repo.FinalizeRewardGrant(ctx, charID, achievementID, time.Now().UTC())
	if err != nil {
		return err
	}
	if updated == nil || !updated.IsClaimed {
		return fmt.Errorf("achievement claim was not finalized")
	}
	return nil
}

func (s *Service) OnLogin(ctx context.Context, charID int64) error {
	return s.RecoverGrantedRewards(ctx, charID)
}

func (s *Service) RecoverGrantedRewards(ctx context.Context, charID int64) error {
	if s.repo == nil {
		return nil
	}
	grants, err := s.repo.ListGrantedRewardGrants(ctx, charID, 100)
	if err != nil {
		return err
	}
	for _, grant := range grants {
		reward, ok := RewardForAchievement(grant.AchievementID)
		if !ok || grant.RewardKind != int(reward.Kind) || grant.TemplateID != reward.TemplateID || grant.Quantity != reward.Quantity {
			return fmt.Errorf("achievement reward grant %d definition is invalid", grant.AchievementID)
		}
		var receipt domainachievement.RewardReceipt
		if err := json.Unmarshal(grant.Receipt, &receipt); err != nil {
			return fmt.Errorf("decode recovered achievement reward %d receipt: %w", grant.AchievementID, err)
		}
		if err := ValidateRewardReceipt(grant.AchievementID, receipt); err != nil {
			return fmt.Errorf("validate recovered achievement reward %d receipt: %w", grant.AchievementID, err)
		}
		if _, err := s.repo.FinalizeRewardGrant(ctx, charID, grant.AchievementID, time.Now().UTC()); err != nil {
			return fmt.Errorf("finalize recovered achievement reward %d: %w", grant.AchievementID, err)
		}
	}
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
