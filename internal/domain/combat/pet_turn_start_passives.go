// Pet turn-start passives apply exact catalog-backed ownership effects before status ticks.
package combat

const diHinhHoanViResistanceFamily = "Di Hình Hoán Vị:kháng debuff"
const canVeThanhNuShieldFamily = "Cận Vệ Thánh Nữ:thuẫn"

type diHinhHoanViLevel struct {
	chance     float64
	resistance int
}

var diHinhHoanViLevels = map[int]diHinhHoanViLevel{
	6751: {chance: 0.20, resistance: 80},
	6752: {chance: 0.30, resistance: 120},
	6753: {chance: 0.40, resistance: 150},
}

var daKhangTinhLevels = map[int]float64{
	3808: 0.50,
	3812: 0.75,
	3813: 1.00,
}

type canVeThanhNuLevel struct {
	ratio  float64
	buffID int
}

var canVeThanhNuLevels = map[int]canVeThanhNuLevel{
	3848: {ratio: 0.50, buffID: 2015},
	3849: {ratio: 0.75, buffID: 2014},
	3850: {ratio: 1.00, buffID: 2013},
}

func (tp *TurnProcessor) activatePetTurnStartPassives(actor *Participant) []*BattleAction {
	actions := make([]*BattleAction, 0, 2)
	if buffID := tp.activateCanVeThanhNu(actor); buffID > 0 {
		actions = append(actions, &BattleAction{
			Round: tp.battle.CurrentRound, ActorID: actor.ID, ActorPosition: actor.Position,
			TargetID: actor.ID, TargetPosition: actor.Position, ActionType: ActionTypeAuto,
			SourceState: map[string]interface{}{"buffAdd": map[string]interface{}{"id": buffID}},
		})
	}
	if removed, buffID := tp.activateDaKhangTinh(actor); removed && buffID > 0 {
		actions = append(actions, &BattleAction{
			Round: tp.battle.CurrentRound, ActorID: actor.ID, ActorPosition: actor.Position,
			TargetID: actor.ID, TargetPosition: actor.Position, ActionType: ActionTypeAuto,
			SourceState: map[string]interface{}{"buffDel": buffID},
		})
	}
	tp.activateDiHinhHoanVi(actor)
	return actions
}

func (tp *TurnProcessor) activateCanVeThanhNu(actor *Participant) int {
	if actor == nil {
		return 0
	}
	level := canVeThanhNuLevel{}
	skillID := 0
	for candidateSkillID, candidate := range canVeThanhNuLevels {
		if actor.SkillIDs[candidateSkillID] && candidate.ratio > level.ratio {
			level = candidate
			skillID = candidateSkillID
		}
	}
	if level.buffID <= 0 {
		return 0
	}
	tp.statusManager.AddEffect(actor.ID, &StatusEffect{
		Duration: 2, SourceID: actor.ID, TemplateID: skillID, BuffID: level.buffID,
		Family: canVeThanhNuShieldFamily, Polarity: PolarityBeneficial, Removable: true,
		Shield: &ShieldEffect{Pool: int(float64(actor.MaxMP) * level.ratio), Hits: 3}, StackCount: 1,
	})
	return level.buffID
}

func (tp *TurnProcessor) activateDaKhangTinh(actor *Participant) (bool, int) {
	if actor == nil {
		return false, 0
	}
	chance := 0.0
	for skillID, candidate := range daKhangTinhLevels {
		if actor.SkillIDs[skillID] && candidate > chance {
			chance = candidate
		}
	}
	if chance <= 0 || tp.rng.Float64() >= chance {
		return false, 0
	}
	return tp.statusManager.RemoveRandomEffectWithBuffID(actor.ID, PolarityHarmful, tp.rng)
}

func (tp *TurnProcessor) activateDiHinhHoanVi(actor *Participant) bool {
	if actor == nil {
		return false
	}
	tp.statusManager.RemoveFamily(actor.ID, diHinhHoanViResistanceFamily)
	level := diHinhHoanViLevel{}
	owned := false
	for skillID, candidate := range diHinhHoanViLevels {
		if actor.SkillIDs[skillID] && (!owned || candidate.resistance > level.resistance) {
			level = candidate
			owned = true
		}
	}
	if !owned || tp.rng.Float64() >= level.chance {
		return false
	}
	tp.statusManager.RemoveAllEffects(actor.ID, PolarityHarmful)
	tp.statusManager.AddEffect(actor.ID, &StatusEffect{
		Duration:   2,
		SourceID:   actor.ID,
		Family:     diHinhHoanViResistanceFamily,
		Polarity:   PolarityBeneficial,
		Removable:  true,
		Modifiers:  []StatModifier{{Stat: StatDebuffResistance, Flat: level.resistance}},
		StackCount: 1,
	})
	return true
}
