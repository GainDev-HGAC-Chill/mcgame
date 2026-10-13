package combat

import (
	"errors"
	"strconv"

	"mcgame-server/internal/domain/combat"
	pkgerrors "mcgame-server/pkg/errors"
)

var ErrQuestKillCreditInflight = errors.New("quest kill credit is being processed; retry battle cleanup")

type questKillClaimState uint8

const (
	questKillClaimInflight questKillClaimState = iota + 1
	questKillClaimCompleted
)

func (s *Service) GetCharacterBattleForCleanup(battleID string, characterID int64) (*combat.Battle, error) {
	battle, err := s.GetBattleForCleanup(battleID)
	if err != nil {
		return nil, err
	}
	if !isPlayerCharacterParticipant(battle, characterID) {
		return nil, pkgerrors.ErrUnauthorized
	}
	return battle, nil
}

func (s *Service) ClaimQuestKillCredit(battleID string, characterID int64) (*combat.Battle, bool, error) {
	s.endedBattleMu.Lock()
	defer s.endedBattleMu.Unlock()

	entry := s.endedBattles[battleID]
	if entry == nil || !isPlayerCharacterParticipant(entry.battle, characterID) {
		return nil, false, nil
	}
	battle := entry.battle
	if !battle.IsEnded() || battle.Result == nil || battle.Result.WinnerSide != combat.SidePlayer {
		return nil, false, nil
	}
	if _, pending := entry.pendingCleanups[strconv.FormatInt(characterID, 10)]; !pending {
		return nil, false, nil
	}
	switch entry.questKillClaims[characterID] {
	case questKillClaimInflight:
		return nil, false, ErrQuestKillCreditInflight
	case questKillClaimCompleted:
		return nil, false, nil
	}
	entry.questKillClaims[characterID] = questKillClaimInflight
	return battle, true, nil
}

func (s *Service) CompleteQuestKillCredit(battleID string, characterID int64) {
	s.endedBattleMu.Lock()
	defer s.endedBattleMu.Unlock()
	entry := s.endedBattles[battleID]
	if entry != nil && entry.questKillClaims[characterID] == questKillClaimInflight {
		entry.questKillClaims[characterID] = questKillClaimCompleted
	}
}

func (s *Service) ReleaseQuestKillCredit(battleID string, characterID int64) {
	s.endedBattleMu.Lock()
	defer s.endedBattleMu.Unlock()
	entry := s.endedBattles[battleID]
	if entry != nil && entry.questKillClaims[characterID] == questKillClaimInflight {
		delete(entry.questKillClaims, characterID)
	}
}

func isPlayerCharacterParticipant(battle *combat.Battle, characterID int64) bool {
	if battle == nil || characterID <= 0 {
		return false
	}
	participant := battle.GetParticipant(strconv.FormatInt(characterID, 10))
	return participant != nil && participant.EntityID == characterID &&
		participant.EntityType == combat.ParticipantTypeCharacter && participant.Side == combat.SidePlayer
}
