package statfeature

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestAggregateCharacterStatBonusesAppliesOwnedCatalogMonsterHeartsOnce(t *testing.T) {
	repo := &testRepository{characterState: []*domainfeature.CharacterFeatureState{{
		CharacterID: 7,
		FeatureKey:  domainfeature.FeatureMonsterHeart,
		State: map[string]interface{}{
			"bag": map[string]interface{}{
				"ren": map[string]interface{}{
					"9001": map[string]interface{}{"id": 9001, "itemId": 101, "n": 1},
				},
				"shou": map[string]interface{}{
					"9002": map[string]interface{}{"id": 9002, "itemId": 102, "n": 1},
				},
			},
			"data": map[string]interface{}{
				"hadActHole": map[string]interface{}{
					"1": map[string]interface{}{"1": 101, "2": 101, "3": 102, "4": 999999},
				},
			},
		},
	}}}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(loadTestManager(t, map[string][]map[string]interface{}{
		models.TableCreaturehHeart: {
			{"id": 101, "type": 1, "prop_type": 1, "propnum": 10500000},
			{"id": 102, "type": 2, "prop_type": 59, "propnum": 46},
		},
		models.TableCreaturehPoint: {
			{"id": 11, "type": -1},
			{"id": 12, "type": -1},
			{"id": 13, "type": -1},
			{"id": 14, "type": -1},
		},
	}))

	bonuses := service.AggregateCharacterStatBonuses(context.Background(), 7)
	if got := bonuses.Flat[domainchar.PropMaxHP]; got != 1050 {
		t.Fatalf("HP bonus = %d, want 1050 with duplicate placement ownership-capped", got)
	}
	if got := bonuses.Float[domainchar.PropFinalPhysReduce]; got != 0.46 {
		t.Fatalf("physical reduction = %v, want 0.46", got)
	}
}

func TestAggregateCharacterStatBonusesRejectsDuplicateMonsterHeartEntryIDs(t *testing.T) {
	repo := &testRepository{characterState: []*domainfeature.CharacterFeatureState{{
		CharacterID: 7,
		FeatureKey:  domainfeature.FeatureMonsterHeart,
		State: map[string]interface{}{
			"bag": map[string]interface{}{
				"ren":  map[string]interface{}{"9001": map[string]interface{}{"id": 9001, "itemId": 101, "n": 1}},
				"shou": map[string]interface{}{"9001": map[string]interface{}{"id": 9001, "itemId": 102, "n": 1}},
			},
			"data": map[string]interface{}{"hadActHole": map[string]interface{}{"1": map[string]interface{}{"1": 101}}},
		},
	}}}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(loadTestManager(t, map[string][]map[string]interface{}{
		models.TableCreaturehHeart: {{"id": 101, "type": 1, "prop_type": 1, "propnum": 10500000}},
		models.TableCreaturehPoint: {{"id": 11, "type": -1}},
	}))

	bonuses := service.AggregateCharacterStatBonuses(context.Background(), 7)
	if len(bonuses.Flat) != 0 || len(bonuses.Float) != 0 {
		t.Fatalf("duplicate source instance must fail closed: %#v", bonuses)
	}
}
