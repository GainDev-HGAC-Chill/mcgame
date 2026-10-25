package achievement

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

type characterLoadoutRepositoryFixture struct {
	items []*domainitem.Item
}

func (r characterLoadoutRepositoryFixture) FindEquipped(context.Context, int64) ([]*domainitem.Item, error) {
	return r.items, nil
}

type characterLoadoutCatalogFixture map[int]*models.EquiptTemplateTemplate

func (c characterLoadoutCatalogFixture) GetEquipment(id int) *models.EquiptTemplateTemplate {
	return c[id]
}

func TestCharacterLoadoutStateReaderUsesEquippedInstanceAndTemplateState(t *testing.T) {
	items := make([]*domainitem.Item, 0, 13)
	catalog := characterLoadoutCatalogFixture{}
	add := func(id int64, templateID, slot, position, kind, color, star int, properties map[string]interface{}) {
		items = append(items, &domainitem.Item{
			ID: id, CharacterID: 99, TemplateID: templateID, ItemType: domainitem.ItemTypeEquipment,
			SlotType: domainitem.SlotTypeEquipped, SlotIndex: slot, ColorCode: color, StarLevel: star, Properties: properties,
		})
		catalog[templateID] = &models.EquiptTemplateTemplate{ID: int64(templateID), Position: float64(position), Kind: float64(kind)}
	}
	for slot := 0; slot < 5; slot++ {
		properties := map[string]interface{}{}
		if slot == 0 {
			properties["maker"] = "Character"
		}
		add(int64(slot+1), 100+slot, slot, slot+1, 3, 5, 0, properties)
	}
	add(20, 200, 13, 14, 10, 3, 0, nil)
	add(21, 201, 14, 15, 8, 3, 20, nil)
	add(22, 202, 15, 16, 8, 3, 0, nil)
	add(23, 203, 16, 17, 8, 3, 0, nil)
	add(24, 204, 17, 18, 8, 3, 0, nil)
	add(25, 205, 20, 21, 11, 3, 0, nil)
	add(26, 206, 21, 22, 13, 4, 0, nil)
	reader := NewCharacterLoadoutStateReader(characterLoadoutRepositoryFixture{items: items}, catalog)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{23, 24, 25, 26, 32, 33, 360, 362, 512, 593} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{29, 30, 31, 361, 363} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestCharacterLoadoutStateReaderRejectsUnknownEquippedTemplate(t *testing.T) {
	item := &domainitem.Item{ID: 1, CharacterID: 99, TemplateID: 999, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped}
	reader := NewCharacterLoadoutStateReader(characterLoadoutRepositoryFixture{items: []*domainitem.Item{item}}, characterLoadoutCatalogFixture{})
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want unknown template error")
	}
}

func TestCharacterLoadoutStateReaderMapsExactFlyerFashionAndWingIdentities(t *testing.T) {
	items := []*domainitem.Item{
		{ID: 1, CharacterID: 99, TemplateID: 100, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 13},
		{ID: 2, CharacterID: 99, TemplateID: 1972, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 20},
		{ID: 3, CharacterID: 99, TemplateID: 1676, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 21, ColorCode: 3},
	}
	catalog := characterLoadoutCatalogFixture{
		100:  {ID: 100, Position: 14, Kind: 10, ResCode: 2070380000003},
		1972: {ID: 1972, Position: 21, Kind: 11},
		1676: {ID: 1676, Position: 22, Kind: 13},
	}
	proofs, err := NewCharacterLoadoutStateReader(characterLoadoutRepositoryFixture{items: items}, catalog).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{27, 38, 364} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{28, 36, 37, 365, 367, 710, 711} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestCharacterLoadoutStateReaderRequiresTwelveSublimatedSlots(t *testing.T) {
	items := make([]*domainitem.Item, 0, 12)
	catalog := characterLoadoutCatalogFixture{}
	for slot := 0; slot < 12; slot++ {
		templateID := 500 + slot
		items = append(items, &domainitem.Item{
			ID: int64(slot + 1), CharacterID: 99, TemplateID: templateID, ItemType: domainitem.ItemTypeEquipment,
			SlotType: domainitem.SlotTypeEquipped, SlotIndex: slot, ColorCode: 3,
			Properties: map[string]interface{}{"flag": `{"sublimeId":50,"sublimeElement":1}`},
		})
		catalog[templateID] = &models.EquiptTemplateTemplate{ID: int64(templateID), Position: float64(slot + 1), Kind: 3}
	}
	proofs, err := NewCharacterLoadoutStateReader(characterLoadoutRepositoryFixture{items: items}, catalog).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{636, 642, 643} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	if byID[637].Done {
		t.Fatalf("requirement 637 proof = %#v, want incomplete", byID[637])
	}
}

func TestCharacterLoadoutStateReaderUsesPersistedMagicWeaponStage(t *testing.T) {
	tests := []struct {
		name              string
		starLevel         int
		propertyRatio     float64
		doneRequirementID int64
	}{
		{name: "stage seven max level twenty", starLevel: 20, propertyRatio: 0.9, doneRequirementID: 592},
		{name: "stage eight", starLevel: 20, propertyRatio: 2, doneRequirementID: 594},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &domainitem.Item{
				ID: 1, CharacterID: 99, TemplateID: 100, ItemType: domainitem.ItemTypeEquipment,
				SlotType: domainitem.SlotTypeEquipped, SlotIndex: 14, StarLevel: tt.starLevel,
				Properties: map[string]interface{}{"mainPropNum1": 100 * tt.propertyRatio, "mainPropNum2": 200 * tt.propertyRatio},
			}
			catalog := characterLoadoutCatalogFixture{100: {
				ID: 100, Position: 15, Kind: 8, Type: 800, MainPropNum1: 100, MainPropNum2: 200,
			}}
			proofs, err := NewCharacterLoadoutStateReader(characterLoadoutRepositoryFixture{items: []*domainitem.Item{item}}, catalog).ReadStateProofs(context.Background(), 99)
			if err != nil {
				t.Fatalf("ReadStateProofs() error = %v", err)
			}
			byID := make(map[int64]StateProof, len(proofs))
			for _, proof := range proofs {
				byID[proof.RequirementID] = proof
			}
			if !byID[tt.doneRequirementID].Done {
				t.Fatalf("requirement %d proof = %#v, want done", tt.doneRequirementID, byID[tt.doneRequirementID])
			}
		})
	}
}
