// Exact-template Pet active skill plans cover creature and evolution skills that cannot be keyed safely by display name.
package combat

import domaincombat "mcgame-server/internal/domain/combat"

var petActiveSkillTemplateCatalog = map[int]domaincombat.SkillPlan{
	5363: rabbitHeartPlan(1, 1.20),
	5362: rabbitHeartPlan(2, 1.50),
	5361: rabbitHeartPlan(3, 1.80),
	// Tho Baby lucky shield. Data buff ID is level-specific.
	5366: rabbitLuckyShieldPlan(1, .30, 1934),
	5365: rabbitLuckyShieldPlan(2, .40, 1933),
	5364: rabbitLuckyShieldPlan(3, .60, 1932),
	// Bạch Hổ Cuồng Nộ - Thú Vương Đột Kích, normal and both evolved branches.
	7017: bachHoRushPlan(1, 1.20, .20),
	7018: bachHoRushPlan(2, 1.50, .30),
	7019: bachHoRushPlan(3, 1.90, .40),
	7020: bachHoRushPlan(1, 1.80, .30),
	7021: bachHoRushPlan(2, 2.00, .40),
	7022: bachHoRushPlan(3, 2.20, .45),
	7023: bachHoRushPlan(1, 2.00, .35),
	7024: bachHoRushPlan(2, 2.20, .45),
	7025: bachHoRushPlan(3, 2.30, .50),
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
	2961: forumDamageStatusPlan("Cao Tần Long Hống Trung Cấp", 1, domaincombat.EffectMagicDamage, .95, .15, 2, 131, domaincombat.StatMagicAttack, -.10),
	2962: forumDamageStatusPlan("Cao Tần Long Hống Trung Cấp", 2, domaincombat.EffectMagicDamage, 1.10, .25, 2, 133, domaincombat.StatMagicAttack, -.16),
	2963: forumDamageStatusPlan("Cao Tần Long Hống Trung Cấp", 3, domaincombat.EffectMagicDamage, 1.25, .35, 2, 135, domaincombat.StatMagicAttack, -.22),
	2666: forumDamageStatusPlan("Đột Thích Xung Phong", 1, domaincombat.EffectPhysicalDamage, .80, .10, 2, 121, domaincombat.StatAttack, -.15),
	2735: forumDamageStatusPlan("Đột Thích Xung Phong", 2, domaincombat.EffectPhysicalDamage, .90, .15, 2, 123, domaincombat.StatAttack, -.24),
	2736: forumDamageStatusPlan("Đột Thích Xung Phong", 3, domaincombat.EffectPhysicalDamage, 1.00, .20, 2, 125, domaincombat.StatAttack, -.33),
	2845: forumControlPlan("Kích Trạng Loạn Tâm", 1, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .05, 2, 187, domaincombat.ControlEffect{Confusion: true}),
	2839: forumControlPlan("Kích Trạng Loạn Tâm", 2, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .10, 2, 187, domaincombat.ControlEffect{Confusion: true}),
	2846: forumControlPlan("Kích Trạng Loạn Tâm", 3, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .15, 2, 187, domaincombat.ControlEffect{Confusion: true}),
	2692: forumDamagePlan("Lạc Thạch Vẫn Tinh", 1, domaincombat.EffectMagicDamage, domaincombat.TargetEnemy, domaincombat.AreaHorizontal, 3, .30, 0),
	2777: forumDamagePlan("Lạc Thạch Vẫn Tinh", 2, domaincombat.EffectMagicDamage, domaincombat.TargetEnemy, domaincombat.AreaHorizontal, 3, .40, 0),
	2784: forumDamagePlan("Lạc Thạch Vẫn Tinh", 3, domaincombat.EffectMagicDamage, domaincombat.TargetEnemy, domaincombat.AreaHorizontal, 3, .50, 0),
	4739: forumControlPlan("Mê Tâm", 1, domaincombat.TargetRandomEnemy, domaincombat.AreaNone, 2, .60, 2, 187, domaincombat.ControlEffect{Confusion: true}),
	4741: forumControlPlan("Mê Tâm", 2, domaincombat.TargetRandomEnemy, domaincombat.AreaNone, 3, .65, 2, 187, domaincombat.ControlEffect{Confusion: true}),
	4742: forumControlPlan("Mê Tâm", 3, domaincombat.TargetRandomEnemy, domaincombat.AreaNone, 4, .70, 3, 187, domaincombat.ControlEffect{Confusion: true}),
	2838: forumStatusPlan("Phản Ứng Trì Độn", 1, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .05, 2, 497, domaincombat.StatSpeed, -.05),
	2843: forumStatusPlan("Phản Ứng Trì Độn", 2, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .10, 2, 498, domaincombat.StatSpeed, -.07),
	2844: forumStatusPlan("Phản Ứng Trì Độn", 3, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .15, 2, 499, domaincombat.StatSpeed, -.10),
	3542: forumDamagePlan("Táng Thần Quần Thể", 1, domaincombat.EffectPhysicalDamage, domaincombat.TargetEnemy, domaincombat.AreaCross, 4, .90, 1),
	3544: forumDamagePlan("Táng Thần Quần Thể", 2, domaincombat.EffectPhysicalDamage, domaincombat.TargetEnemy, domaincombat.AreaCross, 4, 1.08, 1),
	3546: forumDamagePlan("Táng Thần Quần Thể", 3, domaincombat.EffectPhysicalDamage, domaincombat.TargetEnemy, domaincombat.AreaCross, 4, 1.28, 1),
	3540: forumControlPlan("Trào Phúng Quần Thể", 1, domaincombat.TargetEnemy, domaincombat.AreaVertical, 2, .66, 1, 893, domaincombat.ControlEffect{TauntSourceID: "caster"}),
	3566: forumControlPlan("Trào Phúng Quần Thể", 2, domaincombat.TargetEnemy, domaincombat.AreaVertical, 2, .66, 2, 893, domaincombat.ControlEffect{TauntSourceID: "caster"}),
	3567: forumControlPlan("Trào Phúng Quần Thể", 3, domaincombat.TargetEnemy, domaincombat.AreaCross, 4, .66, 2, 893, domaincombat.ControlEffect{TauntSourceID: "caster"}),
	2725: forumDamagePlan("Uy Nhiếp Ý Chí", 1, domaincombat.EffectMagicDamage, domaincombat.TargetEnemy, domaincombat.AreaHorizontal, 3, .30, 0),
	2690: forumDamagePlan("Uy Nhiếp Ý Chí", 2, domaincombat.EffectMagicDamage, domaincombat.TargetEnemy, domaincombat.AreaHorizontal, 3, .40, 0),
	2727: forumDamagePlan("Uy Nhiếp Ý Chí", 3, domaincombat.EffectMagicDamage, domaincombat.TargetEnemy, domaincombat.AreaHorizontal, 3, .50, 0),
	2615: forumControlPlan("Thạch Hóa Công Kích", 1, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .05, 2, 299, domaincombat.ControlEffect{Stone: true}),
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
// bachHoRushPlan's true-damage proc adds 100% physical attack AFTER defense.
// It still respects invulnerability, shields and other incoming-damage rules.
func bachHoRushPlan(level int, coefficient, trueDamageChance float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{
		Family:      "Thú Vương Đột Kích",
		Level:       level,
		Target:      domaincombat.TargetEnemy,
		TargetCount: 1,
		Effects: []domaincombat.SkillEffect{{
			Kind:                  domaincombat.EffectPhysicalDamage,
			Coefficient:           coefficient,
			TrueDamageChance:      trueDamageChance,
			TrueDamageAttackRatio: 1.0,
		}},
	}
}

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

func rabbitHeartPlan(level int, coefficient float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{Family: "Nhịp Đập Trái Tim", Level: level,
		Target: domaincombat.TargetEnemy, TargetCount: 1,
		ComboChain: true, ComboChainMaxCasts: level * 5, ComboChainDamageStep: .20,
		Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectMagicDamage, Coefficient: coefficient}},
	}
}

func rabbitLuckyShieldPlan(level int, ratio float64, buffID int) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{Family: "Hào Quang May Mắn", Level: level,
		Target: domaincombat.TargetTeam, TargetCount: 2,
		Effects: []domaincombat.SkillEffect{{
			Kind: domaincombat.EffectStatus, Family: "Hào Quang May Mắn",
			Duration: 4, Polarity: domaincombat.PolarityBeneficial,
			Removable: true, BuffID: buffID,
			ShieldPercentMaxMP: ratio, ShieldHits: 3,
		}},
	}
}
