// Pet active skill catalog helpers define special, control, and support families.
package combat

import domaincombat "mcgame-server/internal/domain/combat"

func addFixedSpecials(catalog map[string]petSkillLevels) {
	set := func(name string, levels ...domaincombat.SkillPlan) {
		entries := make(petSkillLevels, len(levels))
		for _, plan := range levels {
			plan.Family = name
			entries[plan.Level] = plan
		}
		catalog[name] = entries
	}
	for i, ratio := range []float64{.35, .45, .55} {
		level := i + 1
		setLevel(catalog, "Hủy Diệt", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetEnemy, Area: domaincombat.AreaHorizontal, TargetCount: 3, RequireHPMax: .9, Sacrifice: true, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectFixedMaxHPDamage, Coefficient: ratio}}})
		setLevel(catalog, "Phóng Thích Năng Lượng", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetEnemy, TargetCount: 1, RestTurns: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectFixedMaxHPDamage, Coefficient: []float64{.3, .45, .6}[i]}}})
		setLevel(catalog, "Táng Thần Đơn Thể", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetEnemy, TargetCount: 1, RestTurns: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectPhysicalDamage, Coefficient: []float64{1.4, 1.6, 1.9}[i]}}})
	}
	for i, coefficient := range []float64{.3, .6, .9} {
		setLevel(catalog, "Độc Thích Đơn Thể", poisonPlan(i+1, domaincombat.TargetEnemy, domaincombat.AreaNone, 1, .88, coefficient))
	}
	for i, coefficient := range []float64{.3, .4, .5} {
		count := []int{4, 5, 10}[i]
		target := domaincombat.TargetEnemy
		area := domaincombat.AreaHorizontal
		if i == 2 {
			target = domaincombat.TargetAllEnemy
			area = domaincombat.AreaNone
		}
		setLevel(catalog, "Độc Thích Quần Thể", poisonPlan(i+1, target, area, count, .77, coefficient))
	}
	for i, pool := range []float64{.5, .75, 1} {
		setLevel(catalog, "Ma Pháp Thuẫn", domaincombat.SkillPlan{Level: i + 1, Target: domaincombat.TargetTeam, TargetCount: 1, Costs: []domaincombat.SkillCost{{Resource: domaincombat.ResourceMP, Percent: .6}}, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Ma Pháp Thuẫn", Duration: 4, Polarity: domaincombat.PolarityBeneficial, Removable: true, ShieldPercentMaxMP: pool, ShieldMagicOnly: true, ShieldHits: 3}}})
		setLevel(catalog, "Nuốt Chửng", domaincombat.SkillPlan{Level: i + 1, Target: domaincombat.TargetSelectedAny, TargetCount: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectRemoveOne}, {Kind: domaincombat.EffectRestoreHP, Coefficient: []float64{.1, .15, .2}[i], RequiresPreviousSuccess: true}, {Kind: domaincombat.EffectRestoreMP, Coefficient: []float64{.1, .15, .2}[i], RequiresPreviousSuccess: true}}})
		mpRatio := []float64{.2, .3, .4}[i]
		setLevel(catalog, "Quang Hợp", domaincombat.SkillPlan{Level: i + 1, Target: domaincombat.TargetSelfPlayer, TargetCount: 1, CostsOnTarget: true, Costs: []domaincombat.SkillCost{{Resource: domaincombat.ResourceMP, Percent: mpRatio}}, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectRestoreHP, Coefficient: mpRatio, ResourceBasis: domaincombat.ResourceMP, ApplyToTarget: true}, {Kind: domaincombat.EffectStatus, Family: "Quang Hợp", Duration: 5, Polarity: domaincombat.PolarityBeneficial, Removable: true, Periodic: &domaincombat.PeriodicEffect{Resource: domaincombat.ResourceMP, Coefficient: []float64{.02, .036, .056}[i]}}}})
	}
	set("Thanh Phong Phục Sinh",
		domaincombat.SkillPlan{Level: 2, Target: domaincombat.TargetSelectedDeadTeam, TargetCount: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectRevive, Coefficient: .2}}},
		domaincombat.SkillPlan{Level: 3, Target: domaincombat.TargetSelectedDeadTeam, TargetCount: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectRevive, Coefficient: .3}}},
	)
	setLevel(catalog, "Thiên Hựu", domaincombat.SkillPlan{Level: 1, Target: domaincombat.TargetTeamPlayer, TargetCount: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Thiên Hựu", Duration: 2, Polarity: domaincombat.PolarityBeneficial, Removable: true, Control: domaincombat.ControlEffect{Invulnerable: true, CannotAct: true}}}})
	for level := 1; level <= 3; level++ {
		ratio := []float64{.3, .4, .5}[level-1]
		rate := []int{14, 21, 30}[level-1]
		setLevel(catalog, "Thuốc Kích Thích", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetSelfPlayer, TargetCount: 1, CostsOnTarget: true, Costs: []domaincombat.SkillCost{{Resource: domaincombat.ResourceHP, Percent: ratio}}, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Thuốc Kích Thích", BuffID: []int{968, 1015, 1016}[level-1], Duration: 3, Polarity: domaincombat.PolarityBeneficial, Removable: true, Modifiers: []domaincombat.StatModifier{{Stat: domaincombat.StatSpeed, Percent: ratio}, {Stat: domaincombat.StatCombo, Flat: rate}, {Stat: domaincombat.StatCritical, Flat: rate}}}}})
	}
}

func poisonPlan(level int, target domaincombat.TargetType, area domaincombat.AreaType, count int, chance, coefficient float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{Level: level, Target: target, Area: area, TargetCount: count, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Độc Thích", Chance: chance, Duration: 3, Polarity: domaincombat.PolarityHarmful, Removable: true, Periodic: &domaincombat.PeriodicEffect{Damage: true, Magic: true, Coefficient: coefficient, Basis: domaincombat.StatMagicAttack}}}}
}

func setLevel(catalog map[string]petSkillLevels, name string, plan domaincombat.SkillPlan) {
	if catalog[name] == nil {
		catalog[name] = make(petSkillLevels)
	}
	plan.Family = name
	catalog[name][plan.Level] = plan
}

func addPureStatuses(catalog map[string]petSkillLevels, status func(string, float64, int, domaincombat.EffectPolarity, ...domaincombat.StatModifier) domaincombat.SkillEffect, modifier func(domaincombat.CombatStat, float64, int) domaincombat.StatModifier) {
	setLevel(catalog, "Ảo Hoa Kỳ Hương", domaincombat.SkillPlan{Level: 1, Target: domaincombat.TargetEnemy, TargetCount: 1, Effects: []domaincombat.SkillEffect{control("Ảo Hoa Kỳ Hương", .3, 2, false, false, true, false)}})
	setLevel(catalog, "Ảo Hoa Kỳ Hương", domaincombat.SkillPlan{Level: 3, Target: domaincombat.TargetEnemy, TargetCount: 1, Effects: []domaincombat.SkillEffect{control("Ảo Hoa Kỳ Hương", .4, 3, false, false, true, false)}})
	for i := 0; i < 3; i++ {
		level := i + 1
		setLevel(catalog, "Đế Vương Nộ", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetSelfPlayer, TargetCount: 1, RequireHPMax: .4, Effects: []domaincombat.SkillEffect{status("Đế Vương Nộ", 0, 2, domaincombat.PolarityBeneficial, modifier(domaincombat.StatAttack, []float64{.5, .75, 1}[i], 0), modifier(domaincombat.StatMagicAttack, []float64{.5, .75, 1}[i], 0))}})
		setLevel(catalog, "Duy Trì Tốc Độ", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetTeam, TargetCount: 1, Effects: []domaincombat.SkillEffect{status("Duy Trì Tốc Độ", 0, 3, domaincombat.PolarityBeneficial, modifier(domaincombat.StatDodge, 0, []int{10, 20, 30}[i]))}})
		setLevel(catalog, "Khóa Mục Tiêu", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetEnemy, Area: domaincombat.AreaVertical, TargetCount: 2, Effects: []domaincombat.SkillEffect{status("Khóa Mục Tiêu", .85, 3, domaincombat.PolarityHarmful, modifier(domaincombat.StatIncomingDamage, []float64{.15, .25, .4}[i], 0))}})
		setLevel(catalog, "Suy Yếu Linh Hồn", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetEnemy, TargetCount: 1, Effects: []domaincombat.SkillEffect{status("Suy Yếu Linh Hồn", 0, 3, domaincombat.PolarityHarmful, modifier(domaincombat.StatReducePhysical, 0, -[]int{10, 20, 30}[i]), modifier(domaincombat.StatReduceMagic, 0, -[]int{10, 20, 30}[i]))}})
		setLevel(catalog, "Thú Tính Gào Thét", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetAllEnemy, TargetCount: 10, Effects: []domaincombat.SkillEffect{status("Thú Tính Gào Thét", 0, 5, domaincombat.PolarityHarmful, modifier(domaincombat.StatDefense, -[]float64{.1, .2, .3}[i], 0))}})
		setLevel(catalog, "Tích Lực Sơ Cấp", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetSelfPlayer, TargetCount: 1, Effects: []domaincombat.SkillEffect{status("Tích Lực Sơ Cấp", 0, 1, domaincombat.PolarityBeneficial, modifier(domaincombat.StatCritical, 0, []int{48, 67, 88}[i]))}})
		setLevel(catalog, "Tích Lực Trung Cấp", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetSelfPlayer, TargetCount: 1, Effects: []domaincombat.SkillEffect{status("Tích Lực Trung Cấp", 0, 2, domaincombat.PolarityBeneficial, modifier(domaincombat.StatCritical, 0, []int{31, 48, 67}[i]))}})
		setLevel(catalog, "Trào Phúng Đơn Thể", domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetEnemy, TargetCount: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Trào Phúng Đơn Thể", Chance: .78, Duration: []int{1, 3, 4}[i], Polarity: domaincombat.PolarityHarmful, Removable: true, Control: domaincombat.ControlEffect{TauntSourceID: "caster"}}}})
	}
	setLevel(catalog, "Hiệp Ước Tử Thần", domaincombat.SkillPlan{Level: 1, Target: domaincombat.TargetSelfPlayer, TargetCount: 1, Effects: []domaincombat.SkillEffect{status("Hiệp Ước Tử Thần", 0, 3, domaincombat.PolarityBeneficial, modifier(domaincombat.StatAttack, .5, 0), modifier(domaincombat.StatMagicAttack, .5, 0), modifier(domaincombat.StatDefense, .5, 0), modifier(domaincombat.StatMagicDefense, .5, 0))}})
	for i, ratio := range []float64{.1, .15, .2} {
		setLevel(catalog, "Hấp Thu Năng Lượng", domaincombat.SkillPlan{Level: i + 1, Target: domaincombat.TargetEnemyNoBoss, TargetCount: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectTransfer, Family: "Hấp Thu Năng Lượng", Duration: []int{2, 2, 3}[i], Coefficient: ratio, TransferCount: i + 1}}})
	}
	addNatureShields(catalog)
	setLevel(catalog, "Vạn Thú Tề Phát", beastPlan(2, 2, .14))
	setLevel(catalog, "Vạn Thú Tề Phát", beastPlan(3, 3, .2))
}

func addNatureShields(catalog map[string]petSkillLevels) {
	for i := 0; i < 3; i++ {
		target := domaincombat.TargetTeam
		area := domaincombat.AreaNone
		count := 1
		if i == 2 {
			area = domaincombat.AreaHorizontal
			count = 5
		}
		setLevel(catalog, "Tự Nhiên Thuẫn Đơn Thể", domaincombat.SkillPlan{Level: i + 1, Target: target, Area: area, TargetCount: count, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Tự Nhiên Thuẫn Đơn Thể", Duration: 4, Polarity: domaincombat.PolarityBeneficial, Removable: true, Modifiers: []domaincombat.StatModifier{{Stat: domaincombat.StatDefense, Flat: []int{1000, 2000, 3000}[i]}}, Periodic: &domaincombat.PeriodicEffect{Coefficient: .65, Basis: domaincombat.StatMagicAttack}}}})
		target = domaincombat.TargetTeam
		area = domaincombat.AreaHorizontal
		count = []int{4, 5, 10}[i]
		if i == 2 {
			target = domaincombat.TargetAllTeam
			area = domaincombat.AreaNone
		}
		setLevel(catalog, "Tự Nhiên Thuẫn Quần Thể", domaincombat.SkillPlan{Level: i + 1, Target: target, Area: area, TargetCount: count, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Tự Nhiên Thuẫn Quần Thể", Duration: 4, Polarity: domaincombat.PolarityBeneficial, Removable: true, Modifiers: []domaincombat.StatModifier{{Stat: domaincombat.StatDefense, Flat: 1000}}, Periodic: &domaincombat.PeriodicEffect{Coefficient: .25, Basis: domaincombat.StatMagicAttack}}}})
	}
}

func beastPlan(level, duration int, ratio float64) domaincombat.SkillPlan {
	return domaincombat.SkillPlan{Level: level, Target: domaincombat.TargetTeam, TargetCount: 1, Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectStatus, Family: "Vạn Thú Tề Phát", Duration: duration, Polarity: domaincombat.PolarityBeneficial, Removable: true, Modifiers: []domaincombat.StatModifier{{Stat: domaincombat.StatAttack, Percent: ratio}, {Stat: domaincombat.StatSpeed, Percent: ratio}, {Stat: domaincombat.StatDefense, Percent: -ratio}, {Stat: domaincombat.StatMagicDefense, Percent: -ratio}}}}}
}
