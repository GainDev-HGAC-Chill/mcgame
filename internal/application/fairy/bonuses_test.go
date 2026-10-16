// Open-sourced by BaoLT

package fairy

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

func TestAggregateCharacterStatBonusesRequiresExactlyOneGuard(t *testing.T) {
	svc, repo, _, _, _ := newTestService(nil)
	state := defaultState()
	state.Fairies[1] = bonusTestFairy(1)
	state.Fairies[2] = bonusTestFairy(2)
	repo.charStates[7] = seedFairyState(7, state)

	bonuses := svc.AggregateCharacterStatBonuses(context.Background(), 7)
	if len(bonuses.Flat) != 0 || len(bonuses.Percent) != 0 {
		t.Fatalf("multiple guards produced bonuses: %+v", bonuses)
	}

	state.Fairies[2].State = fairyStateRest
	repo.charStates[7] = seedFairyState(7, state)
	bonuses = svc.AggregateCharacterStatBonuses(context.Background(), 7)
	if got := bonuses.Flat[domainchar.PropStrength]; got != 2 {
		t.Fatalf("strength bonus = %d, want 2", got)
	}
	if got := bonuses.Flat[domainchar.PropStamina]; got != 4 {
		t.Fatalf("stamina bonus = %d, want 4", got)
	}
	if got := bonuses.Flat[domainchar.PropAgility]; got != 6 {
		t.Fatalf("agility bonus = %d, want 6", got)
	}
	if got := bonuses.Flat[domainchar.PropIntelligence]; got != 8 {
		t.Fatalf("intelligence bonus = %d, want 8", got)
	}
	if got := bonuses.Flat[domainchar.PropSpirit]; got != 10 {
		t.Fatalf("spirit bonus = %d, want 10", got)
	}
}

func TestAggregateCharacterStatBonusesAppliesCanonicalPassive(t *testing.T) {
	svc, repo, _, _, _ := newTestService(nil)
	f := bonusTestFairy(1)
	f.SkillFlag["s1"] = 5918
	state := defaultState()
	state.Fairies[f.ID] = f
	repo.charStates[7] = seedFairyState(7, state)

	data := newFakeSkillData([]*models.SkillTemplate{{ID: 5918, Level: 2, Kind: 2, BuffID: 2418}})
	data.buffs[2418] = &models.BuffTemplate{ID: 2418, PercentFlag: 1, Prop1: 4, PropNum1: 7}
	svc.SetSkillData(data)

	bonuses := svc.AggregateCharacterStatBonuses(context.Background(), 7)
	if got := bonuses.Percent[domainchar.PropAttack]; got != 7 {
		t.Fatalf("physical attack percent = %d, want 7", got)
	}
	if got := bonuses.Flat[domainchar.PropStrength]; got != 2 {
		t.Fatalf("base fairy strength = %d, want 2", got)
	}
}

func TestAggregateCharacterStatBonusesRejectsPassiveCatalogDriftAndDuplicates(t *testing.T) {
	tests := []struct {
		name   string
		skills map[string]int
		rows   []*models.SkillTemplate
	}{
		{
			name:   "buff mismatch",
			skills: map[string]int{"s1": 5918},
			rows:   []*models.SkillTemplate{{ID: 5918, Level: 2, Kind: 2, BuffID: 2419}},
		},
		{
			name:   "duplicate family",
			skills: map[string]int{"s1": 5917, "s2": 5918},
			rows: []*models.SkillTemplate{
				{ID: 5917, Level: 1, Kind: 2, BuffID: 2417},
				{ID: 5918, Level: 2, Kind: 2, BuffID: 2418},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, repo, _, _, _ := newTestService(nil)
			f := bonusTestFairy(1)
			f.SkillFlag = test.skills
			state := defaultState()
			state.Fairies[f.ID] = f
			repo.charStates[7] = seedFairyState(7, state)
			data := newFakeSkillData(test.rows)
			data.buffs[2417] = &models.BuffTemplate{ID: 2417, PercentFlag: 1, Prop1: 4, PropNum1: 5}
			data.buffs[2418] = &models.BuffTemplate{ID: 2418, PercentFlag: 1, Prop1: 4, PropNum1: 7}
			data.buffs[2419] = &models.BuffTemplate{ID: 2419, PercentFlag: 1, Prop1: 4, PropNum1: 10}
			svc.SetSkillData(data)

			bonuses := svc.AggregateCharacterStatBonuses(context.Background(), 7)
			if got := bonuses.Percent[domainchar.PropAttack]; got != 0 {
				t.Fatalf("invalid passive percent = %d, want 0", got)
			}
		})
	}
}

func TestFairyCatalogContainsTwentyTwoThreeLevelFamilies(t *testing.T) {
	if len(fairySkillFamilies) != 22 {
		t.Fatalf("family count = %d, want 22", len(fairySkillFamilies))
	}
	seen := map[int]struct{}{}
	for family, chain := range fairySkillFamilies {
		for level, skillID := range chain {
			if skillID <= 0 {
				t.Fatalf("family %d level %d has invalid id %d", family, level+1, skillID)
			}
			if _, duplicate := seen[skillID]; duplicate {
				t.Fatalf("duplicate fairy skill id %d", skillID)
			}
			seen[skillID] = struct{}{}
			gotFamily, gotLevel, ok := fairySkillFamily(skillID)
			if !ok || gotFamily != family || gotLevel != level+1 {
				t.Fatalf("skill %d resolved to (%d,%d,%v)", skillID, gotFamily, gotLevel, ok)
			}
		}
	}
}

func bonusTestFairy(id int64) *Fairy {
	f := newFairy(id)
	f.State = fairyStateGuard
	f.Doh = 7500
	f.Sta = 10
	f.Ste = 20
	f.Agi = 30
	f.Inte = 40
	f.Ener = 50
	return f
}

func seedFairyState(charID int64, state *State) []*domainfeature.CharacterFeatureState {
	return []*domainfeature.CharacterFeatureState{{CharacterID: charID, FeatureKey: rosterFeatureKey, State: state.toPersist()}}
}
