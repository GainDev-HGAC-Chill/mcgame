// Exact-template Pet active skill plans cover creature and evolution skills that cannot be keyed safely by display name.
package combat

import domaincombat "mcgame-server/internal/domain/combat"

var petActiveSkillTemplateCatalog = map[int]domaincombat.SkillPlan{
	// Lôi Lân Ảnh: normal, ∮ and ㄨ versions of Lôi Đình Thần Phạt.
	7038: loiDinhThanPhatPlan(1, 1.40, .10, .10, 0),
	7039: loiDinhThanPhatPlan(2, 1.50, .10, .10, 0),
	7040: loiDinhThanPhatPlan(3, 1.70, .10, .10, 0),
	7041: loiDinhThanPhatPlan(1, 1.80, .15, .15, 0),
	7042: loiDinhThanPhatPlan(2, 1.90, .20, .20, 0),
	7043: loiDinhThanPhatPlan(3, 2.00, .30, .30, 0),
	7044: loiDinhThanPhatPlan(1, 2.10, .30, .30, .05),
	7045: loiDinhThanPhatPlan(2, 2.20, .30, .30, .10),
	7046: loiDinhThanPhatPlan(3, 2.40, .30, .30, .15),
	7002: uMinhHuyenVuPlan(1, 4, 0.15, 0.50),
	7003: uMinhHuyenVuPlan(2, 6, 0.20, 0.55),
	7004: uMinhHuyenVuPlan(3, 8, 0.25, 0.60),
	7005: uMinhHuyenVuPlan(1, 4, 0.20, 0.55),
	7006: uMinhHuyenVuPlan(2, 6, 0.25, 0.60),
	7007: uMinhHuyenVuPlan(3, 8, 0.27, 0.65),
	7008: uMinhHuyenVuPlan(1, 4, 0.25, 0.60),
	7009: uMinhHuyenVuPlan(2, 6, 0.27, 0.65),
	7010: uMinhHuyenVuPlan(3, 8, 0.30, 0.70),
	6712: bianStrikePlan(1, 1.4, 6724),
	6713: bianStrikePlan(2, 1.7, 6727),
	6714: bianStrikePlan(3, 2.0, 6730),
	6715: bianStrikePlan(1, 1.6, 6733),
	6716: bianStrikePlan(2, 1.9, 6736),
	6717: bianStrikePlan(3, 2.5, 6739),
	6718: bianStrikePlan(1, 1.9, 6742),
	6719: bianStrikePlan(2, 2.2, 6745),
	6720: bianStrikePlan(3, 3.0, 6748),
	6724: followUpStrikePlan(2.5),
	6725: followUpStrikePlan(2.5),
	6726: followUpStrikePlan(2.5),
	6727: followUpStrikePlan(2.75),
	6728: followUpStrikePlan(2.75),
	6729: followUpStrikePlan(2.75),
	6730: followUpStrikePlan(3.0),
	6731: followUpStrikePlan(3.0),
	6732: followUpStrikePlan(3.0),
	6733: followUpStrikePlan(2.8),
	6734: followUpStrikePlan(2.8),
	6735: followUpStrikePlan(2.8),
	6736: followUpStrikePlan(3.15),
	6737: followUpStrikePlan(3.15),
	6738: followUpStrikePlan(3.15),
	6739: followUpStrikePlan(3.5),
	6740: followUpStrikePlan(3.5),
	6741: followUpStrikePlan(3.5),
	6742: followUpStrikePlan(3.0),
	6743: followUpStrikePlan(3.0),
	6744: followUpStrikePlan(3.0),
	6745: followUpStrikePlan(3.5),
	6746: followUpStrikePlan(3.5),
	6747: followUpStrikePlan(3.5),
	6748: followUpStrikePlan(4.0),
	6749: followUpStrikePlan(4.0),
	6750: followUpStrikePlan(4.0),
}

// loiDinhThanPhatPlan always uses the pet's MAX HP for its extra damage,
// never the enemy's HP. Extra-hit variants may repeat once without extra MP.
func loiDinhThanPhatPlan(level int, coefficient, bonusChance, bonusMaxHP, repeatChance float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family:       "Lôi Đình Thần Phạt",
		Level:        level,
		Target:       domaincombat.TargetEnemy,
		TargetCount:  1,
		RepeatChance: repeatChance,
		Effects: []domaincombat.SkillEffect{{
			Kind:                   domaincombat.EffectPhysicalDamage,
			Coefficient:            coefficient,
			BonusCasterMaxHPChance: bonusChance,
			BonusCasterMaxHPRatio:  bonusMaxHP,
		}},
	}
}

func uMinhHuyenVuPlan(level, targetCount int, maxHPRatio, soulLossChance float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family:      "U Minh Huyền Vũ",
		Level:       level,
		Target:      domaincombat.TargetEnemy,
		Area:        domaincombat.AreaHorizontalRows,
		TargetCount: targetCount,
		Effects: []domaincombat.SkillEffect{
			{Kind: domaincombat.EffectPhysicalDamage, Coefficient: 0.25, CasterMaxHPRatio: maxHPRatio},
			{Kind: domaincombat.EffectStatus, Chance: soulLossChance, Duration: 1, Polarity: domaincombat.PolarityHarmful, Removable: true, Family: "U Minh Huyền Vũ:Thất Hồn", Control: domaincombat.ControlEffect{Stun: true}, BuffID: 14},
		},
	}
}

func petActiveSkillPlanByTemplateID(id int) *domaincombat.SkillPlan {
	plan, ok := petActiveSkillTemplateCatalog[id]
	if !ok {
		return nil
	}
	copyPlan := plan
	copyPlan.Effects = append([]domaincombat.SkillEffect(nil), plan.Effects...)
	if plan.OnKill != nil {
		followUp := *plan.OnKill
		followUp.PassiveTargetCounts = cloneIntMap(plan.OnKill.PassiveTargetCounts)
		followUp.TemplateIDs = cloneIntMap(plan.OnKill.TemplateIDs)
		copyPlan.OnKill = &followUp
	}
	return &copyPlan
}

func bianStrikePlan(level int, coefficient float64, followUpTemplateStart int) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family:      "Bách Biến Trùng Kích",
		Level:       level,
		Target:      domaincombat.TargetEnemy,
		TargetCount: 1,
		Effects: []domaincombat.SkillEffect{{
			Kind:        domaincombat.EffectPhysicalDamage,
			Coefficient: coefficient,
		}},
		OnKill: &domaincombat.SkillOnKillFollowUp{
			PassiveTargetCounts: map[int]int{6721: 1, 6722: 2, 6723: 3},
			TemplateIDs:         map[int]int{1: followUpTemplateStart, 2: followUpTemplateStart + 1, 3: followUpTemplateStart + 2},
			Target:              domaincombat.TargetRandomEnemy,
		},
	}
}

func followUpStrikePlan(coefficient float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family:      "Trùng Kích Truy Kích",
		Target:      domaincombat.TargetEnemy,
		TargetCount: 1,
		Effects: []domaincombat.SkillEffect{{
			Kind:        domaincombat.EffectPhysicalDamage,
			Coefficient: coefficient,
		}},
	}
}

func cloneIntMap(source map[int]int) map[int]int {
	clone := make(map[int]int, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
