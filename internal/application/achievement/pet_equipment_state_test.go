package achievement

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type petEquipmentItemRepositoryFixture struct {
	items []*domainitem.Item
}

func (r petEquipmentItemRepositoryFixture) FindByCharacterID(context.Context, int64) ([]*domainitem.Item, error) {
	return r.items, nil
}

type petEquipmentCatalogFixture map[int]*models.EquiptTemplateTemplate

func (c petEquipmentCatalogFixture) GetEquipment(id int) *models.EquiptTemplateTemplate {
	return c[id]
}

func TestPetEquipmentStateReaderRequiresCompleteOwnedSameColorSet(t *testing.T) {
	pet := &domainpet.Pet{ID: 10, CharacterID: 99, Property: map[string]interface{}{"equipGroup": 0}}
	items := make([]*domainitem.Item, 0, 9)
	catalog := petEquipmentCatalogFixture{}
	for slot := 1; slot <= domainpet.TotalPetEquipmentSlots; slot++ {
		itemID := int64(100 + slot)
		templateID := 200 + slot
		pet.SetEquipmentItemID(slot, itemID)
		items = append(items, &domainitem.Item{
			ID: itemID, CharacterID: 99, TemplateID: templateID, ItemType: domainitem.ItemTypeEquipment,
			SlotType: domainitem.SlotTypePetEquipped, SlotIndex: slot - 1, ColorCode: 3,
		})
		catalog[templateID] = &models.EquiptTemplateTemplate{ID: int64(templateID), Kind: 9, Position: float64(49 + slot)}
	}
	items = append(items, &domainitem.Item{
		ID: 999, CharacterID: 99, TemplateID: 300, ItemType: domainitem.ItemTypeEquipment,
		SlotType: domainitem.SlotTypeBag, ColorCode: 5,
	})
	catalog[300] = &models.EquiptTemplateTemplate{ID: 300, Kind: 9, Position: 50}
	reader := NewPetEquipmentStateReader(
		petRosterRepositoryFixture{pets: []*domainpet.Pet{pet}},
		petEquipmentItemRepositoryFixture{items: items},
		catalog,
	)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{87, 513} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{85, 86, 88} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestPetEquipmentStateReaderDoesNotAcceptPartialOrMixedSet(t *testing.T) {
	pet := &domainpet.Pet{ID: 10, CharacterID: 99, Property: map[string]interface{}{"equ1": int64(101), "equ2": int64(102)}}
	items := []*domainitem.Item{
		{ID: 101, CharacterID: 99, TemplateID: 201, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypePetEquipped, SlotIndex: 0, ColorCode: 1},
		{ID: 102, CharacterID: 99, TemplateID: 202, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypePetEquipped, SlotIndex: 1, ColorCode: 2},
	}
	catalog := petEquipmentCatalogFixture{
		201: {ID: 201, Kind: 9, Position: 50},
		202: {ID: 202, Kind: 9, Position: 51},
	}
	reader := NewPetEquipmentStateReader(petRosterRepositoryFixture{pets: []*domainpet.Pet{pet}}, petEquipmentItemRepositoryFixture{items: items}, catalog)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	for _, proof := range proofs {
		if proof.RequirementID != 513 && proof.Done {
			t.Fatalf("partial set requirement %d unexpectedly done", proof.RequirementID)
		}
	}
}

func TestPetEquipmentStateReaderRejectsStaleEquipmentLink(t *testing.T) {
	pet := &domainpet.Pet{ID: 10, CharacterID: 99, Property: map[string]interface{}{"equ1": int64(101)}}
	reader := NewPetEquipmentStateReader(petRosterRepositoryFixture{pets: []*domainpet.Pet{pet}}, petEquipmentItemRepositoryFixture{}, petEquipmentCatalogFixture{})
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want stale-link error")
	}
}

func TestPetEquipmentStateReaderUsesEightPersistedSublimationFlags(t *testing.T) {
	pet := &domainpet.Pet{ID: 10, CharacterID: 99, Property: map[string]interface{}{"equipGroup": 0}}
	items := make([]*domainitem.Item, 0, 8)
	catalog := petEquipmentCatalogFixture{}
	for slot := 1; slot <= domainpet.TotalPetEquipmentSlots; slot++ {
		itemID := int64(100 + slot)
		templateID := 200 + slot
		pet.SetEquipmentItemID(slot, itemID)
		items = append(items, &domainitem.Item{
			ID: itemID, CharacterID: 99, TemplateID: templateID, ItemType: domainitem.ItemTypeEquipment,
			SlotType: domainitem.SlotTypePetEquipped, SlotIndex: slot - 1, ColorCode: 3,
			Properties: map[string]interface{}{"flag": map[string]interface{}{"sublimeId": 50.0, "sublimeElement": 6.0}},
		})
		catalog[templateID] = &models.EquiptTemplateTemplate{ID: int64(templateID), Kind: 9, Position: float64(49 + slot)}
	}
	reader := NewPetEquipmentStateReader(petRosterRepositoryFixture{pets: []*domainpet.Pet{pet}}, petEquipmentItemRepositoryFixture{items: items}, catalog)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{649, 650, 651} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	if byID[644].Done {
		t.Fatalf("requirement 644 proof = %#v, want incomplete", byID[644])
	}
}
