// Open-sourced by BaoLT

package fairy

import (
	"context"
	"strconv"
)

func (s *Service) BattleSkillRotation(ctx context.Context, charID int64) ([]int, error) {
	if s == nil || s.featureRepo == nil {
		return nil, nil
	}
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	guarding := guardingFairy(state)
	if guarding == nil || len(guarding.Config) == 0 {
		return nil, nil
	}
	config, err := s.validateSkillConfig(guarding, guarding.Config)
	if err != nil {
		return nil, err
	}
	selected, _ := exactInt(config[skillConfigSel])
	if selected < 0 {
		return nil, nil
	}
	group, ok := config[strconv.Itoa(selected)].(map[string]interface{})
	if !ok {
		return nil, ErrSkillFailed
	}
	rotation := make([]int, 5)
	for index := range rotation {
		skillID, valid := exactInt(group["c"+strconv.Itoa(index)])
		if !valid {
			return nil, ErrSkillFailed
		}
		rotation[index] = skillID
	}
	return rotation, nil
}
