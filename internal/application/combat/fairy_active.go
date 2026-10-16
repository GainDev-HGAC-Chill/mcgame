// Open-sourced by BaoLT

package combat

import (
	"context"
	"fmt"
	"math"

	domaincombat "mcgame-server/internal/domain/combat"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type fairyCombatData interface {
	GetSkill(id int) *models.SkillTemplate
	GetBuff(id int) *models.BuffTemplate
}

type fairyActiveSpec struct {
	baseSkillID int
	baseBuffID  int
	duration    int
	stat        domaincombat.CombatStat
	propID      int
	values      [3]int
	finalReduce bool
}

var fairyActiveFamilyBases = []int{
	6227, 6230, 6236, 6239, 6242, 6245, 6248, 6251,
	6254, 6257, 6260, 6263, 6266, 6269, 6272, 6275,
}

var supportedFairyActiveSpecs = map[int]fairyActiveSpec{
	6230: {baseSkillID: 6230, baseBuffID: 2724, duration: 2, stat: domaincombat.StatMagicAttack, propID: 5, values: [3]int{5, 8, 13}},
	6236: {baseSkillID: 6236, baseBuffID: 2732, duration: 2, stat: domaincombat.StatAttack, propID: 4, values: [3]int{5, 8, 13}},
	6239: {baseSkillID: 6239, baseBuffID: 2735, duration: 2, stat: domaincombat.StatMagicDefense, propID: 7, values: [3]int{10, 20, 35}},
	6242: {baseSkillID: 6242, baseBuffID: 2738, duration: 2, stat: domaincombat.StatDefense, propID: 6, values: [3]int{10, 20, 35}},
	6260: {baseSkillID: 6260, baseBuffID: 2756, duration: 3, stat: domaincombat.StatSpeed, propID: 11, values: [3]int{7, 10, 15}},
	6269: {baseSkillID: 6269, baseBuffID: 2765, duration: 2, stat: domaincombat.StatIncomingDamage, values: [3]int{5, 8, 13}, finalReduce: true},
}

func (s *Service) attachFairyBattleRotation(ctx context.Context, battle *domaincombat.Battle, participant *domaincombat.Participant, charID int64) {
	if s == nil || s.fairyProvider == nil || battle == nil || participant == nil {
		return
	}
	skills, err := s.fairyProvider.BattleSkillRotation(ctx, charID)
	if err != nil {
		s.logger.Warn("fairy battle profile rejected", zap.Int64("character_id", charID), zap.Error(err))
		return
	}
	if len(skills) == 0 {
		return
	}
	if len(skills) != 5 {
		s.logger.Warn("fairy battle profile rejected", zap.Int64("character_id", charID), zap.Int("slot_count", len(skills)))
		return
	}
	gameData := s.getGameDataManager()
	rotation, err := compileFairyRotation(skills, gameData)
	if err != nil {
		s.logger.Warn("fairy battle profile rejected", zap.Int64("character_id", charID), zap.Error(err))
		return
	}
	battle.SetFairyRotation(participant.ID, rotation)
}

func compileFairyRotation(skills []int, data fairyCombatData) (*domaincombat.FairyRotation, error) {
	if len(skills) != 5 || data == nil {
		return nil, fmt.Errorf("invalid fairy rotation")
	}
	rotation := &domaincombat.FairyRotation{}
	for slot, skillID := range skills {
		if skillID == -1 {
			continue
		}
		base, level, canonical := fairyActiveFamily(skillID)
		if !canonical {
			return nil, fmt.Errorf("invalid fairy active skill %d", skillID)
		}
		spec, supported := supportedFairyActiveSpecs[base]
		if !supported {
			skill := data.GetSkill(skillID)
			if skill == nil || int(skill.ID) != skillID || !wholeNumberEquals(skill.Kind, 1) || !wholeNumberEquals(skill.Level, level) {
				return nil, fmt.Errorf("fairy skill catalog drift for %d", skillID)
			}
			rotation.Slots[slot] = &domaincombat.FairyActiveEffect{SkillID: skillID, Family: fmt.Sprintf("fairy:%d", base)}
			continue
		}
		effect, err := compileSupportedFairyEffect(skillID, level, spec, data)
		if err != nil {
			return nil, err
		}
		rotation.Slots[slot] = effect
	}
	return rotation, nil
}

func compileSupportedFairyEffect(skillID, level int, spec fairyActiveSpec, data fairyCombatData) (*domaincombat.FairyActiveEffect, error) {
	skill := data.GetSkill(skillID)
	expectedBuffID := spec.baseBuffID + level - 1
	if skill == nil || int(skill.ID) != skillID || !wholeNumberEquals(skill.Kind, 1) || !wholeNumberEquals(skill.Level, level) || !wholeNumberEquals(skill.BuffID, expectedBuffID) || !wholeNumberEquals(skill.BuffRound, spec.duration) {
		return nil, fmt.Errorf("fairy skill catalog drift for %d", skillID)
	}
	buff := data.GetBuff(expectedBuffID)
	expectedValue := spec.values[level-1]
	if buff == nil || int(buff.ID) != expectedBuffID || !wholeNumberEquals(buff.Level, level) || !wholeNumberEquals(buff.PercentFlag, 1) {
		return nil, fmt.Errorf("fairy buff catalog drift for %d", expectedBuffID)
	}
	if spec.finalReduce {
		if !wholeNumberEquals(buff.EffectNum, expectedValue) {
			return nil, fmt.Errorf("fairy final reduction drift for %d", expectedBuffID)
		}
	} else if !wholeNumberEquals(buff.Prop1, spec.propID) || !wholeNumberEquals(buff.PropNum1, expectedValue) {
		return nil, fmt.Errorf("fairy stat buff drift for %d", expectedBuffID)
	}
	percent := float64(expectedValue) / 100
	if spec.finalReduce {
		percent = -percent
	}
	return &domaincombat.FairyActiveEffect{
		SkillID:   skillID,
		Family:    fmt.Sprintf("fairy:%d", spec.baseSkillID),
		Duration:  spec.duration,
		Supported: true,
		Modifiers: []domaincombat.StatModifier{{Stat: spec.stat, Percent: percent}},
	}, nil
}

func fairyActiveFamily(skillID int) (int, int, bool) {
	for _, base := range fairyActiveFamilyBases {
		if skillID >= base && skillID <= base+2 {
			return base, skillID - base + 1, true
		}
	}
	return 0, 0, false
}

func wholeNumberEquals(value float64, expected int) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Trunc(value) == value && int(value) == expected
}
