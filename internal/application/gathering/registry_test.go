package gathering

import (
	"testing"

	domaingathering "mcgame-server/internal/domain/gathering"
	"mcgame-server/internal/gamedata/models"
)

type registryCatalog map[int]*models.ItemTemplateTemplate

func (c registryCatalog) GetItem(id int) *models.ItemTemplateTemplate {
	return c[id]
}

func TestOrdinaryRewardRegistries(t *testing.T) {
	expectedFish := [][2]int{{2361, 2362}, {2363, 2364}, {2365, 2366}, {2367, 2368}, {2369, 2370}, {2589, 2588}, {2591, 2592}, {2594, 2595}, {2597, 2599}, {2600, 2602}}
	expectedHerbs := [][2]int{{2491, 2492}, {2493, 2494}, {2495, 2496}, {2497, 2498}, {2499, 2500}, {2604, 2603}, {2606, 2605}, {2607, 2608}, {2609, 2610}, {2611, 2612}}

	for index, expected := range expectedFish {
		actual, ok := OrdinaryRewards(domaingathering.ResourceKindFishing, index+1)
		if !ok || actual != expected {
			t.Fatalf("fish level %d = %v, %v; want %v, true", index+1, actual, ok, expected)
		}
	}
	for index, expected := range expectedHerbs {
		actual, ok := OrdinaryRewards(domaingathering.ResourceKindHerb, index+1)
		if !ok || actual != expected {
			t.Fatalf("herb level %d = %v, %v; want %v, true", index+1, actual, ok, expected)
		}
	}
	for _, templateID := range []int{2590, 2596, 2598, 2601} {
		if !IsExcludedSpecialFish(templateID) {
			t.Fatalf("special fish %d is not excluded", templateID)
		}
		for level := 1; level <= 10; level++ {
			if IsOrdinaryReward(domaingathering.ResourceKindFishing, level, templateID) {
				t.Fatalf("special fish %d appears at ordinary level %d", templateID, level)
			}
		}
	}
}

func TestOrdinaryFishingNodesExcludeSpecialFamiliesAndHighLevels(t *testing.T) {
	for npcID, expectedLevel := range map[int]int{1189: 1, 1199: 2, 1205: 3, 1208: 4, 1212: 5, 1216: 6, 1596: 7, 1598: 7} {
		level, ok := OrdinaryFishingNodeLevel(npcID)
		if !ok || level != expectedLevel {
			t.Fatalf("node %d = level %d, %v; want %d, true", npcID, level, ok, expectedLevel)
		}
	}
	for _, npcID := range []int{1573, 1574, 1575, 1576, 1577, 1635, 1241, 1250} {
		if level, ok := OrdinaryFishingNodeLevel(npcID); ok {
			t.Fatalf("unsupported fishing node %d was enabled at level %d", npcID, level)
		}
	}
}

func TestHerbTemplatesAreRegistryOnly(t *testing.T) {
	for npcID := 1241; npcID <= 1250; npcID++ {
		level, ok := HerbTemplateLevel(npcID)
		if !ok || level != npcID-1240 {
			t.Fatalf("herb template %d = level %d, %v", npcID, level, ok)
		}
		if _, ordinaryNode := OrdinaryFishingNodeLevel(npcID); ordinaryNode {
			t.Fatalf("placeholder herb %d became an ordinary fishing node", npcID)
		}
	}
}

func TestMasteryGain(t *testing.T) {
	tests := []struct {
		skillLevel    int
		resourceLevel int
		want          int
	}{
		{skillLevel: 5, resourceLevel: 5, want: 2},
		{skillLevel: 5, resourceLevel: 4, want: 1},
		{skillLevel: 5, resourceLevel: 3, want: 0},
		{skillLevel: 4, resourceLevel: 5, want: 0},
	}
	for _, test := range tests {
		if got := MasteryGain(test.skillLevel, test.resourceLevel); got != test.want {
			t.Fatalf("MasteryGain(%d, %d) = %d; want %d", test.skillLevel, test.resourceLevel, got, test.want)
		}
	}
}

func TestValidateOrdinaryCatalog(t *testing.T) {
	catalog := registryCatalog{}
	for _, kind := range []domaingathering.ResourceKind{domaingathering.ResourceKindFishing, domaingathering.ResourceKindHerb} {
		expectedType := fishItemCatalogType
		if kind == domaingathering.ResourceKindHerb {
			expectedType = herbItemCatalogType
		}
		for level := 1; level <= 10; level++ {
			rewards, ok := OrdinaryRewards(kind, level)
			if !ok {
				t.Fatalf("missing registry level %d for kind %d", level, kind)
			}
			for _, templateID := range rewards {
				catalog[templateID] = &models.ItemTemplateTemplate{ID: int64(templateID), Type: float64(expectedType), ItemLevel: float64(level)}
			}
		}
	}
	if err := ValidateOrdinaryCatalog(catalog); err != nil {
		t.Fatalf("ValidateOrdinaryCatalog() error = %v", err)
	}

	delete(catalog, 2361)
	if err := ValidateOrdinaryCatalog(catalog); err == nil {
		t.Fatal("ValidateOrdinaryCatalog() accepted a missing reward")
	}
}
