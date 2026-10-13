// Each successful per-quest kill write stores its event key with the objective.
// A retry after a partial repository failure cannot replay previously persisted kills.
package quest

import (
	"context"
	"slices"

	"mcgame-server/internal/domain/quest"
	pkgerrors "mcgame-server/pkg/errors"
)

func (s *Service) OnMonsterKilledWithCredit(ctx context.Context, charID int64, monsterID int, eventID string) ([]*quest.QuestProgress, error) {
	if eventID == "" || len(eventID) > 256 || charID <= 0 || monsterID <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	lock := &s.mutationLocks[uint64(charID)%uint64(len(s.mutationLocks))]
	lock.Lock()
	defer lock.Unlock()
	active, err := s.questRepo.FindActiveByCharacter(ctx, charID)
	if err != nil {
		return nil, err
	}
	var updated []*quest.QuestProgress
	for _, qp := range active {
		if err := s.reconcileCatalogObjectives(qp); err != nil {
			return updated, err
		}
		if err := s.syncSpecialRequirements(ctx, qp); err != nil {
			return updated, err
		}
		if qp.CompletionBlocked {
			continue
		}
		alreadyApplied := false
		for _, obj := range qp.Objectives {
			if obj.Type == quest.ObjectiveKillMonster && slices.Contains(obj.KillCreditEvents, eventID) {
				alreadyApplied = true
				break
			}
		}
		if alreadyApplied {
			continue
		}
		for i := range qp.Objectives {
			obj := &qp.Objectives[i]
			if obj.Type != quest.ObjectiveKillMonster || obj.Target != monsterID || obj.IsComplete() {
				continue
			}
			obj.AddProgress(1)
			obj.KillCreditEvents = append(obj.KillCreditEvents, eventID)
			if err := s.questRepo.Update(ctx, qp); err != nil {
				return updated, err
			}
			updated = append(updated, qp)
			break
		}
	}
	return updated, nil
}
