// Open-sourced by BaoLT

package item

import (
	"encoding/json"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

func TestChangePrefixPersistsPreviewProperties(t *testing.T) {
	char := &domainchar.Character{ID: 1, Name: "tester", Level: 1, MoneyBind: 2000}
	equipment := &domainitem.Item{
		ID: 1801, CharacterID: char.ID, TemplateID: 3801, ItemType: domainitem.ItemTypeEquipment,
		SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1, ColorCode: 4,
		Properties: map[string]interface{}{
			"q": 18, "quality": 18, "preNameType": 3,
			"mainProp1": 11, "mainPropNum1": 999,
			"bindMainPropNum1": 999, "bindMainPropNum2": 999,
			"activeProp": 0, "activePropNum": 0,
		},
	}
	materials := []*domainitem.Item{
		{ID: 1802, CharacterID: char.ID, TemplateID: 4801, ItemType: domainitem.ItemTypeMaterial, SlotType: domainitem.SlotTypeBag, SlotIndex: 1, StackCount: 1, Properties: map[string]interface{}{"level": 4}},
		{ID: 1803, CharacterID: char.ID, TemplateID: 4802, ItemType: domainitem.ItemTypeMaterial, SlotType: domainitem.SlotTypeBag, SlotIndex: 2, StackCount: 1, Properties: map[string]interface{}{"level": 4}},
		{ID: 1804, CharacterID: char.ID, TemplateID: 4803, ItemType: domainitem.ItemTypeMaterial, SlotType: domainitem.SlotTypeBag, SlotIndex: 3, StackCount: 1, Properties: map[string]interface{}{"level": 4}},
	}
	handler, ctx := buildItemTestHandler(t, char, equipment, materials[0], materials[1], materials[2])
	template := models.EquiptTemplateTemplate{
		ID: 3801, Position: 1, ReqLevel: 1,
		RequireItem1: 4801, RequireNum1: 1, RequireItem2: 4802, RequireNum2: 1, RequireItem3: 4803, RequireNum3: 1,
		MainProp1: 11, MainPropNum1: 100, Prop1: 8, PropNum1: 60, BindPropNum: 10,
		ActivePropType: 8, ActivePropNum: 7,
	}
	raw, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("marshal equipment template: %v", err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableEquiptTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load equipment template: %v", err)
	}

	response, err := handler.ChangePrefix(ctx, []interface{}{map[string]interface{}{
		"e":  float64(equipment.ID),
		"i1": map[string]interface{}{"idx": float64(materials[0].ID), "flag": false},
		"i2": map[string]interface{}{"idx": float64(materials[1].ID), "flag": false},
		"i3": map[string]interface{}{"idx": float64(materials[2].ID), "flag": false},
	}})
	if err != nil {
		t.Fatalf("change prefix: %v", err)
	}
	preview := response.(map[string]interface{})["equIns"].(map[string]interface{})
	if got := craftDisplayInt(preview["mainPropNum1"]); got <= 0 || got == 999 {
		t.Fatalf("expected rerolled main property, got %d", got)
	}
	if got := craftDisplayInt(preview["activePropNum"]); got != 7 {
		t.Fatalf("expected soul fallback 7, got %d", got)
	}

	if _, err := handler.SureChangePrefix(ctx, []interface{}{float64(1)}); err != nil {
		t.Fatalf("confirm change prefix: %v", err)
	}
	updated, err := handler.itemService.GetItemByID(ctx.Context, char.ID, equipment.ID)
	if err != nil {
		t.Fatalf("load updated equipment: %v", err)
	}
	for _, key := range []string{"mainProp1", "mainPropNum1", "prop1", "propNum1", "bindMainPropNum1", "bindMainPropNum2", "activeProp", "activePropNum"} {
		if craftDisplayInt(updated.Properties[key]) != craftDisplayInt(preview[key]) {
			t.Fatalf("property %s persisted as %#v, preview was %#v", key, updated.Properties[key], preview[key])
		}
	}
}

func TestChangeSoulPartialLowerRollDoesNotApply(t *testing.T) {
	char := &domainchar.Character{ID: 1, Name: "tester", Level: 1, MoneyBind: 200000}
	equipment := &domainitem.Item{
		ID: 1901, CharacterID: char.ID, TemplateID: 3901, ItemType: domainitem.ItemTypeEquipment,
		SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1,
		Properties: map[string]interface{}{"activeProp": 8, "activePropNum": 5},
	}
	material := &domainitem.Item{ID: 1902, CharacterID: char.ID, TemplateID: 1659, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 1, StackCount: 2}
	handler, ctx := buildItemTestHandler(t, char, equipment, material)
	template := models.EquiptTemplateTemplate{ID: 3901, ReqLevel: 60, ActivePropType: 8, ActivePropNum: 1}
	raw, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("marshal equipment template: %v", err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableEquiptTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load equipment template: %v", err)
	}

	response, err := handler.ChangeSoul(ctx, []interface{}{float64(equipment.ID), float64(material.ID), float64(2)})
	if err != nil {
		t.Fatalf("change soul partial: %v", err)
	}
	result := response.(map[string]interface{})
	if got := craftDisplayInt(result["apro"]); got != 1 {
		t.Fatalf("expected lower preview value 1, got %d", got)
	}
	updated, err := handler.itemService.GetItemByID(ctx.Context, char.ID, equipment.ID)
	if err != nil {
		t.Fatalf("load equipment: %v", err)
	}
	if got := changeSoulPropertyInt(updated.Properties["activePropNum"]); got != 5 {
		t.Fatalf("expected rejected lower roll to preserve 5, got %d", got)
	}
}

func TestMaterialMixRejectsLevelSixWithoutConsumption(t *testing.T) {
	char := &domainchar.Character{ID: 1, Name: "tester", Level: 1}
	material := &domainitem.Item{
		ID: 2001, CharacterID: char.ID, TemplateID: 4901, ItemType: domainitem.ItemTypeMaterial,
		SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 5, ColorCode: 5,
		Properties: map[string]interface{}{"level": 6},
	}
	handler, ctx := buildItemTestHandler(t, char, material)
	template := models.ItemTemplateTemplate{ID: 4901, Name: "Nguyen lieu cap 6", Kind: 6, Type: 600, StackMax: 99}
	raw, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("marshal material template: %v", err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load material template: %v", err)
	}

	response, err := handler.MaterialMixOne(ctx, []interface{}{float64(5), float64(material.ID), false})
	if err != nil {
		t.Fatalf("mix level six material: %v", err)
	}
	if response.(map[string]interface{})["flag"] != false {
		t.Fatalf("expected max-tier mix rejection, got %#v", response)
	}
	unchanged, err := handler.itemService.GetItemByID(ctx.Context, char.ID, material.ID)
	if err != nil {
		t.Fatalf("load material after rejection: %v", err)
	}
	if unchanged.StackCount != 5 {
		t.Fatalf("expected stack to remain 5, got %d", unchanged.StackCount)
	}
}
