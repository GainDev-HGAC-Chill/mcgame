// Open-sourced by BaoLT

package item

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"

	"go.uber.org/zap"
)

func TestPetEquipmentMutationsPreserveInstanceState(t *testing.T) {
	repo := newItemServiceTestRepo()
	durability := 77
	repo.items[100] = &domainitem.Item{
		ID: 100, CharacterID: 1, TemplateID: 5001, ItemType: domainitem.ItemTypeEquipment,
		SlotType: domainitem.SlotTypePetEquipped, SlotIndex: 17, StackCount: 1, IsBound: true,
		Durability: &durability, MaxDurability: &durability, StarLevel: 6, ColorCode: 1,
		Properties: map[string]interface{}{
			"custom": "keep", "preNameType": 4,
			"tid": 5001, "cid": 5001, "giid": 5001, "tplId": 5001,
		},
	}
	service := NewService(repo, zap.NewNop())

	upgraded, err := service.ReplacePetEquipmentTemplate(context.Background(), 1, 100, 5002)
	if err != nil {
		t.Fatalf("ReplacePetEquipmentTemplate() error = %v", err)
	}
	if upgraded.ID != 100 || upgraded.TemplateID != 5002 || upgraded.SlotType != domainitem.SlotTypePetEquipped || upgraded.SlotIndex != 17 {
		t.Fatalf("upgraded identity/slot = %#v", upgraded)
	}
	if !upgraded.IsBound || upgraded.StarLevel != 6 || upgraded.Properties["custom"] != "keep" {
		t.Fatalf("upgraded preserved state = %#v", upgraded)
	}
	for _, key := range []string{"tid", "cid", "giid", "tplId"} {
		if upgraded.Properties[key] != 5002 {
			t.Fatalf("upgraded properties[%q] = %v, want 5002", key, upgraded.Properties[key])
		}
	}
	dto := service.BuildClientItemDTO(upgraded)
	for _, key := range []string{"tid", "cid", "giid", "tplId"} {
		if dto[key] != 5002 {
			t.Fatalf("upgraded dto[%q] = %v, want 5002", key, dto[key])
		}
	}

	colored, err := service.SetPetEquipmentDisplayColor(context.Background(), 1, 100, 3)
	if err != nil {
		t.Fatalf("SetPetEquipmentDisplayColor() error = %v", err)
	}
	if colored.ColorCode != 4 || colored.Properties["color"] != "3" || colored.Properties["preNameType"] != 4 {
		t.Fatalf("colored equipment = %#v", colored)
	}

	bound, err := service.SetPetEquipmentBindProps(context.Background(), 1, 100, 8, 9)
	if err != nil {
		t.Fatalf("SetPetEquipmentBindProps() error = %v", err)
	}
	if bound.Properties["bindMainPropNum1"] != 8 || bound.Properties["bindMainPropNum2"] != 9 {
		t.Fatalf("bind props = %#v", bound.Properties)
	}
	if bound.ID != 100 || bound.TemplateID != 5002 || bound.CalculateSID() != 1018 || bound.StarLevel != 6 {
		t.Fatalf("final preserved state = %#v", bound)
	}
}

func TestPetEquipmentMutationsRejectForeignOwner(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[100] = &domainitem.Item{ID: 100, CharacterID: 2, TemplateID: 5001, ItemType: domainitem.ItemTypeEquipment}
	service := NewService(repo, zap.NewNop())

	if _, err := service.ReplacePetEquipmentTemplate(context.Background(), 1, 100, 5002); err == nil {
		t.Fatal("ReplacePetEquipmentTemplate() accepted foreign item")
	}
	if _, err := service.SetPetEquipmentDisplayColor(context.Background(), 1, 100, 1); err == nil {
		t.Fatal("SetPetEquipmentDisplayColor() accepted foreign item")
	}
	if _, err := service.SetPetEquipmentBindProps(context.Background(), 1, 100, 1, 1); err == nil {
		t.Fatal("SetPetEquipmentBindProps() accepted foreign item")
	}
}
