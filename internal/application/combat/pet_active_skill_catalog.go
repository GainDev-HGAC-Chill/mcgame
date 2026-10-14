// Pet active skill catalog is the single source of truth for audited family and level mechanics.
package combat

import domaincombat "mcgame-server/internal/domain/combat"

type petSkillLevels map[int]domaincombat.SkillPlan

var petActiveSkillCatalog = buildPetActiveSkillCatalog()

func petActiveSkillPlan(name string, level int) *domaincombat.SkillPlan {
	levels := petActiveSkillCatalog[name]
	plan, ok := levels[level]
	if !ok {
		return nil
	}
	copyPlan := plan
	copyPlan.Effects = append([]domaincombat.SkillEffect(nil), plan.Effects...)
	copyPlan.Costs = append([]domaincombat.SkillCost(nil), plan.Costs...)
	return &copyPlan
}

func buildPetActiveSkillCatalog() map[string]petSkillLevels {
	catalog := make(map[string]petSkillLevels, 55)
	damage := func(kind domaincombat.EffectKind, coefficient float64) domaincombat.SkillEffect {
		return domaincombat.SkillEffect{Kind: kind, Coefficient: coefficient}
	}
	status := func(family string, chance float64, duration int, polarity domaincombat.EffectPolarity, modifiers ...domaincombat.StatModifier) domaincombat.SkillEffect {
		return domaincombat.SkillEffect{Kind: domaincombat.EffectStatus, Family: family, Chance: chance, Duration: duration, Polarity: polarity, Removable: true, Modifiers: modifiers}
	}
	modifier := func(stat domaincombat.CombatStat, percent float64, flat int) domaincombat.StatModifier {
		return domaincombat.StatModifier{Stat: stat, Percent: percent, Flat: flat}
	}
	add := func(name string, targets domaincombat.TargetType, area domaincombat.AreaType, counts []int, effects [][]domaincombat.SkillEffect) {
		levels := make(petSkillLevels, len(effects))
		for i, levelEffects := range effects {
			count := counts[min(i, len(counts)-1)]
			levels[i+1] = domaincombat.SkillPlan{Family: name, Level: i + 1, Target: targets, Area: area, TargetCount: count, Effects: levelEffects}
		}
		catalog[name] = levels
	}
	magic := domaincombat.EffectMagicDamage
	physical := domaincombat.EffectPhysicalDamage
	enemy := domaincombat.TargetEnemy
	addDamage := func(name string, kind domaincombat.EffectKind, target domaincombat.TargetType, area domaincombat.AreaType, counts []int, coefficients ...float64) {
		effects := make([][]domaincombat.SkillEffect, len(coefficients))
		for i, coefficient := range coefficients {
			effects[i] = []domaincombat.SkillEffect{damage(kind, coefficient)}
		}
		add(name, target, area, counts, effects)
	}

	addDamage("Ác Ma Chi Kích", magic, enemy, domaincombat.AreaNone, []int{1}, 1, 1.1, 1.2)
	addDamage("Cự Mộc Áp Thân", magic, enemy, domaincombat.AreaNone, []int{1}, .8, .9, 1)
	addDamage("Cự Mộc Áp Thân Cao Cấp", magic, enemy, domaincombat.AreaNone, []int{1}, 1.15, 1.35, 1.55)
	addDamage("Dã Thú Cuồng Vũ", physical, enemy, domaincombat.AreaNone, []int{1}, 1, 1, 1)
	addDamage("Diệp Vũ Chi Thuật Cao Cấp", magic, enemy, domaincombat.AreaHorizontal, []int{3}, .55, .65, .75)
	addDamage("Diệt Vong Chi Ca", magic, enemy, domaincombat.AreaNone, []int{1}, .8)
	addDamage("Lời Ngọt Ngào", magic, domaincombat.TargetAllEnemy, domaincombat.AreaNone, []int{10}, 1, 1.1, 1.2)
	addDamage("Phong Hôn Chi Nhẫn", magic, enemy, domaincombat.AreaHorizontal, []int{3, 3, 4}, .6, .75, .6)
	addDamage("Quyến Luyến Không Rời", magic, enemy, domaincombat.AreaHorizontal, []int{3}, .36, .48, .6)
	addDamage("Tấn Công Cấp Tốc", physical, enemy, domaincombat.AreaNone, []int{1}, 1, 1, 1)
	addDamage("Thích Nhĩ Hô Khiếu", physical, enemy, domaincombat.AreaNone, []int{1}, 1.1, 1.2, 1.3)
	delete(catalog["Thích Nhĩ Hô Khiếu"], 1)
	addDamage("Toái Băng Phong Bạo", magic, enemy, domaincombat.AreaNone, []int{1}, .8, .9, 1)
	add("Hổ Khiếu Sơn Lâm", domaincombat.TargetAllEnemy, domaincombat.AreaNone, []int{10}, [][]domaincombat.SkillEffect{
		{damage(physical, .8), {Kind: domaincombat.EffectRemoveOne, Chance: .3}},
		{damage(physical, 1), {Kind: domaincombat.EffectRemoveOne, Chance: .4}},
		{damage(physical, 1.2), {Kind: domaincombat.EffectRemoveOne, Chance: .5}},
	})
	add("Thợ Săn Sát Thủ", domaincombat.TargetAllTeam, domaincombat.AreaNone, []int{10}, [][]domaincombat.SkillEffect{
		{status("Thợ Săn Sát Thủ", 0, 5, domaincombat.PolarityBeneficial, modifier(domaincombat.StatSpeed, .1, 0), modifier(domaincombat.StatAccuracy, 0, 10))},
		{status("Thợ Săn Sát Thủ", 0, 5, domaincombat.PolarityBeneficial, modifier(domaincombat.StatSpeed, .15, 0), modifier(domaincombat.StatAccuracy, 0, 15))},
		{status("Thợ Săn Sát Thủ", 0, 5, domaincombat.PolarityBeneficial, modifier(domaincombat.StatSpeed, .2, 0), modifier(domaincombat.StatAccuracy, 0, 20))},
	})
	add("Tứ Phương Khiếp Đảm", domaincombat.TargetRandomEnemy, domaincombat.AreaNone, []int{4, 6, 8}, [][]domaincombat.SkillEffect{
		{status("Tứ Phương Khiếp Đảm", 0, 1, domaincombat.PolarityHarmful, modifier(domaincombat.StatAttack, -.05, 0), modifier(domaincombat.StatMagicAttack, -.05, 0), modifier(domaincombat.StatSpeed, -.05, 0)), {Kind: domaincombat.EffectRemoveAll, Chance: .1}},
		{status("Tứ Phương Khiếp Đảm", 0, 1, domaincombat.PolarityHarmful, modifier(domaincombat.StatAttack, -.08, 0), modifier(domaincombat.StatMagicAttack, -.08, 0), modifier(domaincombat.StatSpeed, -.08, 0)), {Kind: domaincombat.EffectRemoveAll, Chance: .2}},
		{status("Tứ Phương Khiếp Đảm", 0, 1, domaincombat.PolarityHarmful, modifier(domaincombat.StatAttack, -.12, 0), modifier(domaincombat.StatMagicAttack, -.12, 0), modifier(domaincombat.StatSpeed, -.12, 0)), {Kind: domaincombat.EffectRemoveAll, Chance: .3}},
	})

	addDamageStatus := func(name string, kind domaincombat.EffectKind, area domaincombat.AreaType, counts []int, coefficients, chances []float64, durations []int, stat domaincombat.CombatStat, percents []float64) {
		effects := make([][]domaincombat.SkillEffect, len(coefficients))
		for i := range coefficients {
			effects[i] = []domaincombat.SkillEffect{damage(kind, coefficients[i]), status(name, chances[i], durations[i], domaincombat.PolarityHarmful, modifier(stat, percents[i], 0))}
		}
		add(name, enemy, area, counts, effects)
	}
	addDamageStatus("Bát Diệp Trảm", physical, domaincombat.AreaNone, []int{1}, []float64{.8, .9, 1}, []float64{.1, .15, .2}, []int{2, 2, 2}, domaincombat.StatAttack, []float64{-.15, -.24, -.25})
	add("Dập Hỏa Thuật", enemy, domaincombat.AreaVertical, []int{2}, [][]domaincombat.SkillEffect{
		{damage(magic, .6), status("Dập Hỏa Thuật", .1, 2, domaincombat.PolarityHarmful, modifier(domaincombat.StatDefense, -.05, 0), modifier(domaincombat.StatMagicDefense, -.05, 0))},
		{damage(magic, .7), status("Dập Hỏa Thuật", .15, 2, domaincombat.PolarityHarmful, modifier(domaincombat.StatDefense, -.1, 0), modifier(domaincombat.StatMagicDefense, -.1, 0))},
		{damage(magic, .9), status("Dập Hỏa Thuật", .25, 2, domaincombat.PolarityHarmful, modifier(domaincombat.StatDefense, -.15, 0), modifier(domaincombat.StatMagicDefense, -.15, 0))},
	})
	addDamageStatus("Độc Huyết Trảo", physical, domaincombat.AreaNone, []int{1}, []float64{.65, .75, 1.05}, []float64{.3, .34, .4}, []int{2, 2, 3}, domaincombat.StatAttack, []float64{-.18, -.24, -.33})
	delete(catalog["Độc Huyết Trảo"], 1)
	addDamageStatus("Hắc Dạ Ma Ảnh", magic, domaincombat.AreaNone, []int{1}, []float64{.55, .75, 1.05}, []float64{.3, .34, .4}, []int{2, 2, 3}, domaincombat.StatMagicAttack, []float64{-.1, -.16, -.22})
	addDamageStatus("Kí Sinh Chủng Tử Cao Cấp", magic, domaincombat.AreaNone, []int{1}, []float64{1.15, 1.35, 1.55}, []float64{.25, .4, .55}, []int{2, 2, 2}, domaincombat.StatMagicAttack, []float64{-.1, -.1, -.1})
	addDamageStatus("Ký Sinh Chủng Tử", magic, domaincombat.AreaNone, []int{1}, []float64{.8, .9, 1}, []float64{.1, .15, .2}, []int{2, 2, 2}, domaincombat.StatMagicAttack, []float64{-.1, -.16, -.22})
	delete(catalog["Ký Sinh Chủng Tử"], 3)
	add("Long Khiếu", domaincombat.TargetAllEnemy, domaincombat.AreaNone, []int{10}, [][]domaincombat.SkillEffect{
		{damage(magic, .2), control("Long Khiếu", .1, 1, false, true, false, false)},
		{damage(magic, .22), control("Long Khiếu", .15, 1, false, true, false, false)},
		{damage(magic, .26), control("Long Khiếu", .25, 1, false, true, false, false)},
	})
	add("Pháo Kích", enemy, domaincombat.AreaCross, []int{4}, damageControlLevels(physical, []float64{1, 1, 1}, []float64{.6, .7, .8}, []int{1, 1, 1}, "Pháo Kích", true, false))
	addDamageStatus("Sương Đông Đơn Thể", magic, domaincombat.AreaNone, []int{1}, []float64{1, 1.1, 1.2}, []float64{.97, .97, .97}, []int{3, 3, 3}, domaincombat.StatSpeed, []float64{-.1, -.2, -.3})
	addDamageStatus("Sương Đông Quần Thể", magic, domaincombat.AreaCross, []int{4}, []float64{.7, .75, .8}, []float64{.97, .97, .97}, []int{3, 3, 3}, domaincombat.StatSpeed, []float64{-.1, -.2, -.3})
	add("Suy Tiến Trang Trí", enemy, domaincombat.AreaVertical, []int{2}, damageControlLevels(physical, []float64{.3, .4, .6}, []float64{.3, .34, .4}, []int{1, 1, 2}, "Suy Tiến Trang Trí", true, false))
	add("Tàn Nhẫn Tư Liệt", enemy, domaincombat.AreaNone, []int{1}, bleedLevels())
	add("Thủy Cam Lộ", domaincombat.TargetTeam, domaincombat.AreaNone, []int{1}, healHotLevels())

	add("Đoạt Mệnh", enemy, domaincombat.AreaNone, []int{1}, currentHPBonusLevels(physical, []float64{1.3, 1.6, 1.9}))
	add("Liêu Hồn", enemy, domaincombat.AreaNone, []int{1}, currentHPBonusLevels(magic, []float64{1.2, 1.4, 1.6}))
	addFixedSpecials(catalog)
	addPureStatuses(catalog, status, modifier)
	return catalog
}

func control(family string, chance float64, duration int, stun, confusion, sleep, invulnerable bool) domaincombat.SkillEffect {
	return domaincombat.SkillEffect{Kind: domaincombat.EffectStatus, Family: family, Chance: chance, Duration: duration, Polarity: domaincombat.PolarityHarmful, Removable: true, Control: domaincombat.ControlEffect{Stun: stun, Confusion: confusion, Sleep: sleep, Invulnerable: invulnerable, CannotAct: stun || sleep || invulnerable}}
}

func damageControlLevels(kind domaincombat.EffectKind, damage, chances []float64, durations []int, family string, stun, confusion bool) [][]domaincombat.SkillEffect {
	levels := make([][]domaincombat.SkillEffect, len(damage))
	for i := range damage {
		levels[i] = []domaincombat.SkillEffect{{Kind: kind, Coefficient: damage[i]}, control(family, chances[i], durations[i], stun, confusion, false, false)}
	}
	return levels
}

func bleedLevels() [][]domaincombat.SkillEffect {
	damage := []float64{.55, .75, 1.05}
	chance := []float64{.2, .25, .3}
	duration := []int{2, 2, 3}
	tick := []float64{.6, .9, 1.4}
	levels := make([][]domaincombat.SkillEffect, 3)
	for i := range levels {
		levels[i] = []domaincombat.SkillEffect{{Kind: domaincombat.EffectPhysicalDamage, Coefficient: damage[i]}, {Kind: domaincombat.EffectStatus, Family: "Tàn Nhẫn Tư Liệt", Chance: chance[i], Duration: duration[i], Polarity: domaincombat.PolarityHarmful, Removable: true, Periodic: &domaincombat.PeriodicEffect{Damage: true, Coefficient: tick[i], Basis: domaincombat.StatEnergy}}}
	}
	return levels
}

func healHotLevels() [][]domaincombat.SkillEffect {
	heal := []float64{1.1, 1.3, 1.6}
	hot := []int{75, 145, 238}
	levels := make([][]domaincombat.SkillEffect, 3)
	for i := range levels {
		levels[i] = []domaincombat.SkillEffect{{Kind: domaincombat.EffectHeal, Coefficient: heal[i]}, {Kind: domaincombat.EffectStatus, Family: "Thủy Cam Lộ", Duration: 2, Polarity: domaincombat.PolarityBeneficial, Removable: true, Periodic: &domaincombat.PeriodicEffect{Fixed: hot[i], Resource: domaincombat.ResourceHP}}}
	}
	return levels
}

func currentHPBonusLevels(kind domaincombat.EffectKind, coefficients []float64) [][]domaincombat.SkillEffect {
	levels := make([][]domaincombat.SkillEffect, 3)
	for i := range levels {
		levels[i] = []domaincombat.SkillEffect{{Kind: kind, Coefficient: coefficients[i], BonusCurrentHPChance: float64(i+1) * .1, BonusCurrentHPRatio: float64(i+1) * .05}}
	}
	return levels
}
