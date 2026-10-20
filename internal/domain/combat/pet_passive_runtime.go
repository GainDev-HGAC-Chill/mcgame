// Shared passive trigger engine for Pet skills. Behavior is configured by pet_passive_rules.json.
// Keep audited active-skill plans in the existing pet_active_skill_templates catalog.
package combat

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"mcgame-server/internal/domain/skill"
)

//go:embed pet_passive_rules.json
var petPassiveRulesJSON []byte

type petPassiveRank struct {
	SkillID              int     `json:"skill_id"`
	Chance               float64 `json:"chance"`
	BuffID               int     `json:"buff_id"`
	ReflectRatio         float64 `json:"reflect_ratio"`
	MissingHPAttackScale float64 `json:"missing_hp_attack_scale"`
}

type petPassiveRule struct {
	Kind            string           `json:"kind"`
	Family          string           `json:"family"`
	DisplayName     string           `json:"display_name"`
	TriggerSkillIDs []int            `json:"trigger_skill_ids"`
	BuffID          int              `json:"buff_id"`
	Duration        int              `json:"duration"`
	TargetCount     int              `json:"target_count"`
	Coefficient     float64          `json:"coefficient"`
	Levels          []petPassiveRank `json:"levels"`
}

type petPassiveRulebook struct {
	Version int              `json:"version"`
	Rules   []petPassiveRule `json:"rules"`
}

var petPassiveOnce sync.Once
var petPassiveLoaded petPassiveRulebook
var petPassiveError error

func parsePetPassiveRules(raw []byte) (petPassiveRulebook, error) {
	var result petPassiveRulebook
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&result); err != nil {
		return result, err
	}
	if result.Version != 1 {
		return result, fmt.Errorf("invalid passive rulebook version %d", result.Version)
	}
	seenIDs := make(map[int]bool)
	seenTriggers := make(map[string]bool)
	for _, rule := range result.Rules {
		if rule.Family == "" || len(rule.Levels) == 0 {
			return result, fmt.Errorf("missing family or levels")
		}
		if rule.Kind != "on_hit_control" && rule.Kind != "turn_buff" && rule.Kind != "on_kill_strike" {
			return result, fmt.Errorf("unknown rule kind %s", rule.Kind)
		}
		for _, trigger := range rule.TriggerSkillIDs {
			key := fmt.Sprintf("%s:%d", rule.Kind, trigger)
			if trigger <= 0 || seenTriggers[key] {
				return result, fmt.Errorf("duplicate/invalid trigger %s", key)
			}
			seenTriggers[key] = true
		}
		if rule.Kind != "turn_buff" && len(rule.TriggerSkillIDs) == 0 {
			return result, fmt.Errorf("missing trigger IDs: %s", rule.Kind)
		}
		if rule.Kind == "on_hit_control" && (rule.BuffID <= 0 || rule.Duration <= 0) {
			return result, fmt.Errorf("missing control buff metadata")
		}
		if rule.Kind == "on_kill_strike" && (rule.TargetCount <= 0 || rule.Coefficient <= 0) {
			return result, fmt.Errorf("missing on-kill attack config")
		}
		for _, level := range rule.Levels {
			if level.SkillID <= 0 || seenIDs[level.SkillID] || level.Chance < 0 || level.Chance > 2 || level.ReflectRatio < 0 || level.ReflectRatio > 1 || level.MissingHPAttackScale < 0 {
				return result, fmt.Errorf("invalid/duplicate passive skill ID %d", level.SkillID)
			}
			if rule.Kind == "turn_buff" && (level.BuffID <= 0 || level.ReflectRatio <= 0) {
				return result, fmt.Errorf("invalid buff level %d", level.SkillID)
			}
			seenIDs[level.SkillID] = true
		}
	}
	return result, nil
}

func petPassiveRules() []petPassiveRule {
	petPassiveOnce.Do(func() { petPassiveLoaded, petPassiveError = parsePetPassiveRules(petPassiveRulesJSON) })
	if petPassiveError != nil {
		return nil
	}
	return petPassiveLoaded.Rules
}

// Multiple Pet families can share one trigger type. Select by active skill or
// owned passive skill rather than hardcoding a single Pet per event type.
func petRuleForTrigger(kind string, activeSkillID int) *petPassiveRule {
	for _, rule := range petPassiveRules() {
		if rule.Kind == kind && ruleTriggered(&rule, activeSkillID) {
			copy := rule
			return &copy
		}
	}
	return nil
}

func petRuleForOwner(kind string, owned map[int]bool) (*petPassiveRule, petPassiveRank, int) {
	for _, rule := range petPassiveRules() {
		if rule.Kind != kind {
			continue
		}
		level, rank := ownedPassiveLevel(&rule, owned)
		if rank > 0 {
			copy := rule
			return &copy, level, rank
		}
	}
	return nil, petPassiveRank{}, 0
}

func ruleTriggered(rule *petPassiveRule, skillID int) bool {
	if rule == nil {
		return false
	}
	for _, id := range rule.TriggerSkillIDs {
		if id == skillID {
			return true
		}
	}
	return false
}

func ownedPassiveLevel(rule *petPassiveRule, owned map[int]bool) (petPassiveRank, int) {
	if rule == nil {
		return petPassiveRank{}, 0
	}
	result := petPassiveRank{}
	rank := 0
	for i, level := range rule.Levels {
		if owned[level.SkillID] && i+1 > rank {
			result = level
			rank = i + 1
		}
	}
	return result, rank
}

// A linked passive may be triggered by any active skill ID listed in its JSON rule.
func petLinkedControlEffect(skillID int, owned map[int]bool) (SkillEffect, bool) {
	rule := petRuleForTrigger("on_hit_control", skillID)
	if rule == nil {
		return SkillEffect{}, false
	}
	rank, _ := ownedPassiveLevel(rule, owned)
	if rank.SkillID == 0 {
		return SkillEffect{}, false
	}
	return SkillEffect{
		Kind: EffectStatus, Chance: rank.Chance, Duration: rule.Duration,
		Polarity: PolarityHarmful, Removable: true, Family: rule.Family,
		BuffID: rule.BuffID, Control: ControlEffect{TauntSourceID: "caster"},
	}, true
}

func petHasOnKillProc(skillID int) bool {
	return petRuleForTrigger("on_kill_strike", skillID) != nil
}

func petSecretStrikeRank(owned map[int]bool) (float64, int) {
	_, rank, index := petRuleForOwner("on_kill_strike", owned)
	return rank.Chance, index
}

func petTrueDamageBonus(actor *Participant, effect SkillEffect, roll float64) int {
	if actor == nil || effect.TrueDamageChance <= 0 || effect.TrueDamageAttackRatio <= 0 || roll >= effect.TrueDamageChance {
		return 0
	}
	return max(0, int(math.Round(float64(actor.Attack)*effect.TrueDamageAttackRatio)))
}

func petTurnBuffRank(owned map[int]bool) int {
	_, _, rank := petRuleForOwner("turn_buff", owned)
	return rank
}

func (m *StatusEffectManager) petActiveTurnBuff(participantID string) (*petPassiveRule, petPassiveRank, int) {
	if m == nil {
		return nil, petPassiveRank{}, 0
	}
	for _, effect := range m.effects[participantID] {
		if effect.Duration <= 0 {
			continue
		}
		for _, rule := range petPassiveRules() {
			if rule.Kind != "turn_buff" || rule.Family != effect.Family {
				continue
			}
			for index, level := range rule.Levels {
				if level.BuffID == effect.BuffID {
					copy := rule
					return &copy, level, index + 1
				}
			}
		}
	}
	return nil, petPassiveRank{}, 0
}

func (m *StatusEffectManager) petBattleBuffRank(participantID string) int {
	_, _, rank := m.petActiveTurnBuff(participantID)
	return rank
}

func (m *StatusEffectManager) petMissingHPAttackPercent(p *Participant) float64 {
	if p == nil || p.MaxHP <= 0 {
		return 0
	}
	_, level, rank := m.petActiveTurnBuff(p.ID)
	if rank == 0 {
		return 0
	}
	missing := float64(max(0, p.MaxHP-p.CurrentHP)) / float64(p.MaxHP)
	return missing * level.MissingHPAttackScale
}

// Called once before the first attack of each round, from either battle runner.
func (tp *TurnProcessor) activatePetTurnBuff(p *Participant) *BattleAction {
	if tp == nil || p == nil || !p.IsAlive {
		return nil
	}
	rule, level, rank := petRuleForOwner("turn_buff", p.SkillIDs)
	if rule == nil || rank == 0 {
		return nil
	}
	currentRule, _, currentRank := tp.statusManager.petActiveTurnBuff(p.ID)
	if currentRule != nil && currentRule.Family == rule.Family && currentRank == rank {
		for _, effect := range tp.statusManager.effects[p.ID] {
			if effect.Family == rule.Family {
				effect.Duration = rule.Duration
			}
		}
		return nil
	}
	tp.statusManager.RemoveFamily(p.ID, rule.Family)
	tp.statusManager.AddEffect(p.ID, &StatusEffect{
		Duration: rule.Duration, SourceID: p.ID, TemplateID: level.SkillID,
		BuffID: level.BuffID, Family: rule.Family, Polarity: PolarityBeneficial,
		Removable: true, StackCount: 1,
	})
	return &BattleAction{Round: tp.battle.CurrentRound, ActorID: p.ID, ActorPosition: p.Position, TargetID: p.ID,
		TargetPosition: p.Position, ActionType: ActionTypeAuto,
		SourceState: map[string]interface{}{"buffAdd": map[string]interface{}{"id": level.BuffID}}}
}

// Reflection is a shared damage-response hook. It never reflects a reflection.
func (tp *TurnProcessor) petReflectActions(attacker, defender *Participant, incoming int) []*BattleAction {
	if tp == nil || attacker == nil || defender == nil || attacker.ID == defender.ID || !attacker.IsAlive || !defender.IsAlive || incoming <= 0 {
		return nil
	}
	rule, level, rank := tp.statusManager.petActiveTurnBuff(defender.ID)
	if rule == nil || rank == 0 {
		return nil
	}
	amount := int(math.Round(float64(incoming) * level.ReflectRatio))
	if amount <= 0 {
		return nil
	}
	before := attacker.CurrentHP
	applied := tp.applyDamage(attacker, amount, false)
	if applied <= 0 {
		return nil
	}
	displayName := rule.DisplayName
	if displayName == "" {
		displayName = rule.Family
	}
	tpl := &skill.SkillTemplate{ID: level.SkillID, Name: displayName, SkillType: skill.SkillTypePhysical}
	action := tp.damageAction(defender, attacker, level.SkillID, tpl, applied, false, false, before)
	return tp.appendPlannedDamagePlayback(nil, defender, attacker, action)
}

// One proc check, up to TargetCount distinct living enemies, no recursive procs.
func (tp *TurnProcessor) executePetOnKillStrike(actor *Participant, primary *skill.SkillTemplate) []*BattleAction {
	if tp == nil || actor == nil || !actor.IsAlive || primary == nil {
		return nil
	}
	rule := petRuleForTrigger("on_kill_strike", primary.ID)
	if rule == nil {
		return nil
	}
	rank, _ := ownedPassiveLevel(rule, actor.SkillIDs)
	if rank.SkillID == 0 || tp.rng.Float64() >= rank.Chance {
		return nil
	}
	targets := NewTargetResolverWithRNG(tp.battle, tp.rng).ResolveTargets(actor, "", TargetRandomEnemy, AreaNone, rule.TargetCount)
	if len(targets) == 0 {
		return nil
	}
	once := *primary
	once.AttackCount = 1
	effect := SkillEffect{Kind: EffectPhysicalDamage, Coefficient: rule.Coefficient}
	actions := make([]*BattleAction, 0, len(targets)*3)
	for _, target := range targets {
		if !actor.IsAlive || target == nil || !target.IsAlive {
			continue
		}
		actions = append(actions, tp.applyPlannedDamage(actor, target, once.ID, &once, effect)...)
	}
	return actions
}
