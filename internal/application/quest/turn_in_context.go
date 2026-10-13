// NPC identity and map location are validated from server data, not finishQuest's optional n field.
package quest

import (
	"context"
	"math"

	pkgerrors "mcgame-server/pkg/errors"
)

const questNPCInteractionRadius = 300

func (s *Service) atQuestNPC(ctx context.Context, charID int64, npcID int) bool {
	if s.gameDataManager == nil || s.charRepo == nil || npcID <= 0 {
		return false
	}
	npc := s.gameDataManager.GetNPC(npcID)
	if npc == nil || npc.PosMapID <= 0 {
		return false
	}
	char, err := s.GetCharacter(ctx, charID)
	if err != nil || char == nil || char.ID != charID || char.MapID != int(npc.PosMapID) {
		return false
	}
	dx := float64(char.PosX) - npc.PosX
	dy := float64(char.PosY) - npc.PosY
	return math.Hypot(dx, dy) <= questNPCInteractionRadius
}

func (s *Service) ValidateQuestTurnIn(ctx context.Context, charID int64, questID int) error {
	if s.gameDataManager == nil {
		return pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetQuest(questID)
	if tpl == nil {
		return pkgerrors.ErrInvalidInput
	}
	if tpl.FinishNPC > 0 && !s.atQuestNPC(ctx, charID, int(tpl.FinishNPC)) {
		return pkgerrors.ErrInvalidInput
	}
	return nil
}
