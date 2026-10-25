// Open-sourced by BaoLT

// Awakening skill variants authorize the transformed exSid selected by the Flash client.
package awaken

import "context"

func (s *Service) IsAuthorizedSkillVariant(ctx context.Context, charID int64, requestedSkillID int, learnedSkillIDs []int) (bool, error) {
	if s == nil || s.gameData == nil || s.featureRepo == nil || requestedSkillID <= 0 {
		return false, nil
	}
	if s.gameData.GetSkill(requestedSkillID) == nil {
		return false, nil
	}

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return false, err
	}
	for _, learnedSkillID := range learnedSkillIDs {
		base := s.gameData.GetSkill(learnedSkillID)
		if base == nil {
			continue
		}
		rank := state.PointDict[base.CodeName]
		if awakeningVariantID(base.ExSid1, base.ExSid2, base.ExSid3, rank) == requestedSkillID {
			return true, nil
		}
	}
	return false, nil
}

func awakeningVariantID(exSid1, exSid2, exSid3 float64, rank int) int {
	switch rank {
	case 1:
		return int(exSid1)
	case 2:
		return int(exSid2)
	case 3:
		return int(exSid3)
	default:
		return 0
	}
}
