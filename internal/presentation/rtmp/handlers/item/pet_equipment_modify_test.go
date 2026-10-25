// Open-sourced by BaoLT

package item

import (
	"context"
	"encoding/json"
	"testing"

	apppet "mcgame-server/internal/application/pet"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	infrartmp "mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap/zaptest"
)

type petEquipmentModifyPetRepo struct {
	pets map[int64]*domainpet.Pet
}

func (r *petEquipmentModifyPetRepo) Save(ctx context.Context, pet *domainpet.Pet) error {
	copy := *pet
	copy.Property = clonePetEquipmentModifyMap(pet.Property)
	copy.CreatureData = clonePetEquipmentModifyMap(pet.CreatureData)
	r.pets[pet.ID] = &copy
	return nil
}

func (r *petEquipmentModifyPetRepo) FindByID(ctx context.Context, id int64) (*domainpet.Pet, error) {
	pet := r.pets[id]
	if pet == nil {
		return nil, nil
	}
	copy := *pet
	copy.Property = clonePetEquipmentModifyMap(pet.Property)
	copy.CreatureData = clonePetEquipmentModifyMap(pet.CreatureData)
	return &copy, nil
}

func (r *petEquipmentModifyPetRepo) FindByCharacterID(ctx context.Context, characterID int64) ([]*domainpet.Pet, error) {
	result := make([]*domainpet.Pet, 0)
	for _, pet := range r.pets {
		if pet.CharacterID != characterID {
			continue
		}
		copy := *pet
		copy.Property = clonePetEquipmentModifyMap(pet.Property)
		copy.CreatureData = clonePetEquipmentModifyMap(pet.CreatureData)
		result = append(result, &copy)
	}
	return result, nil
}

func (r *petEquipmentModifyPetRepo) FindFollowingPet(ctx context.Context, characterID int64) (*domainpet.Pet, error) {
	return nil, nil
}

func (r *petEquipmentModifyPetRepo) Delete(ctx context.Context, id int64) error {
	delete(r.pets, id)
	return nil
}

func (r *petEquipmentModifyPetRepo) Count(ctx context.Context, characterID int64) (int, error) {
	return len(r.pets), nil
}

func (r *petEquipmentModifyPetRepo) UpdateFollowState(ctx context.Context, petID int64, isFollowing bool) error {
	return nil
}

func (r *petEquipmentModifyPetRepo) ClearFollowing(ctx context.Context, characterID int64) error {
	return nil
}

func clonePetEquipmentModifyMap(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}
	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func TestPetEquLevelUpPreservesInstanceAndRefreshesEquippedPet(t *testing.T) {
	handler, ctx, itemRepo, petRepo, char, calls := buildPetEquipmentModifyHandler(t)

	response, err := handler.PetEquLevelUp(ctx, []interface{}{float64(9001), float64(9101)})
	if err != nil {
		t.Fatalf("PetEquLevelUp() error = %v", err)
	}
	result := response.(map[string]interface{})
	assertPetEquipmentValue(t, result, "flag", true)
	assertPetEquipmentValue(t, result, "num", 5)
	assertPetEquipmentValue(t, result, "i", int64(9001))
	assertPetEquipmentValue(t, result, "tid", 5002)
	assertPetEquipmentValue(t, result, "sid", 1001)
	assertPetEquipmentValue(t, result, "next", 5003)

	updated := itemRepo.items[9001]
	if updated.TemplateID != 5002 || updated.ID != 9001 || updated.SlotType != domainitem.SlotTypePetEquipped || updated.SlotIndex != 0 {
		t.Fatalf("updated equipment identity/slot = %#v", updated)
	}
	if !updated.IsBound || updated.StarLevel != 7 || updated.Properties["custom"] != "keep" {
		t.Fatalf("preserved equipment fields = %#v", updated)
	}
	if char.MoneyBind != 164000 {
		t.Fatalf("moneyBind = %d, want 164000", char.MoneyBind)
	}
	if petRepo.pets[7001].Property["finalStrength"] != 55 {
		t.Fatalf("refreshed finalStrength = %#v, want 55", petRepo.pets[7001].Property["finalStrength"])
	}
	if !hasPetEquipmentCallback(calls.calls, "onRefreshPetProp") {
		t.Fatal("missing onRefreshPetProp callback")
	}
}

func TestPetEquLevelUpRejectsWrongMaterialWithoutMutation(t *testing.T) {
	handler, ctx, itemRepo, _, char, _ := buildPetEquipmentModifyHandler(t)
	itemRepo.items[9101].TemplateID = 2334

	response, err := handler.PetEquLevelUp(ctx, []interface{}{float64(9001), float64(9101)})
	if err != nil {
		t.Fatalf("PetEquLevelUp() error = %v", err)
	}
	result := response.(map[string]interface{})
	assertPetEquipmentValue(t, result, "flag", false)
	if itemRepo.items[9001].TemplateID != 5001 || itemRepo.items[9101].StackCount != 10 || char.MoneyBind != 200000 {
		t.Fatalf("rejected request mutated state: equipment=%d material=%d money=%d", itemRepo.items[9001].TemplateID, itemRepo.items[9101].StackCount, char.MoneyBind)
	}
}

func TestPetEquModColorConsumesOnFailureAndMutatesOnSuccess(t *testing.T) {
	handler, ctx, itemRepo, _, char, calls := buildPetEquipmentModifyHandler(t)
	equipment := itemRepo.items[9001]
	equipment.SlotType = domainitem.SlotTypeBag
	equipment.SlotIndex = 3
	equipment.Properties["color"] = "1"
	equipment.ColorCode = 2
	itemRepo.items[9102].StackCount = 6
	handler.petEquipmentRoll = func(max int) int { return 100 }

	failedResponse, err := handler.PetEquModColor(ctx, []interface{}{float64(9001), float64(9102)})
	if err != nil {
		t.Fatalf("PetEquModColor(failure) error = %v", err)
	}
	failed := failedResponse.(map[string]interface{})
	assertPetEquipmentValue(t, failed, "flag", false)
	assertPetEquipmentValue(t, failed, "num", 2)
	if itemRepo.items[9102].StackCount != 4 || char.MoneyBind != 184000 || itemRepo.items[9001].ColorCode != 2 {
		t.Fatalf("failed roll state = material %d money %d color %d", itemRepo.items[9102].StackCount, char.MoneyBind, itemRepo.items[9001].ColorCode)
	}

	handler.petEquipmentRoll = func(max int) int { return 1 }
	successResponse, err := handler.PetEquModColor(ctx, []interface{}{float64(9001), float64(9102)})
	if err != nil {
		t.Fatalf("PetEquModColor(success) error = %v", err)
	}
	success := successResponse.(map[string]interface{})
	assertPetEquipmentValue(t, success, "flag", true)
	assertPetEquipmentValue(t, success, "num", 2)
	assertPetEquipmentValue(t, success, "i", int64(9001))
	assertPetEquipmentValue(t, success, "c", 2)
	assertPetEquipmentValue(t, success, "sid", 2104)
	if itemRepo.items[9001].ColorCode != 3 || itemRepo.items[9001].Properties["color"] != "2" {
		t.Fatalf("successful color mutation = code %d props %#v", itemRepo.items[9001].ColorCode, itemRepo.items[9001].Properties)
	}
	if char.MoneyBind != 168000 {
		t.Fatalf("moneyBind = %d, want 168000", char.MoneyBind)
	}
	if hasPetEquipmentCallback(calls.calls, "onRefreshPetProp") {
		t.Fatal("bag equipment emitted pet refresh")
	}
	if hasPetEquipmentCallback(calls.calls, "onAddCharactorSlot") || hasPetEquipmentCallback(calls.calls, "onDelCharactorSlot") {
		t.Fatal("color responder already applies consumed quantity and must not receive a duplicate inventory callback")
	}
}

func TestPetEquModBindPreviewRejectAndAccept(t *testing.T) {
	handler, ctx, itemRepo, _, char, calls := buildPetEquipmentModifyHandler(t)
	rolls := []int{7, 8, 9, 10}
	handler.petEquipmentRoll = func(max int) int {
		value := rolls[0]
		rolls = rolls[1:]
		return value
	}

	previewResponse, err := handler.PetEquModBind(ctx, []interface{}{float64(9001), float64(9103)})
	if err != nil {
		t.Fatalf("PetEquModBind() error = %v", err)
	}
	preview := previewResponse.(map[string]interface{})
	assertPetEquipmentValue(t, preview, "f", true)
	assertPetEquipmentValue(t, preview, "ii", int64(9103))
	assertPetEquipmentValue(t, preview, "n", 2)
	assertPetEquipmentValue(t, preview, "b1", 2)
	assertPetEquipmentValue(t, preview, "b11", 7)
	assertPetEquipmentValue(t, preview, "b2", 3)
	assertPetEquipmentValue(t, preview, "b12", 8)
	assertPetEquipmentValue(t, preview, "mainProp1", 20)
	assertPetEquipmentValue(t, preview, "mainProp2", 21)
	if itemRepo.items[9001].Properties["bindMainPropNum1"] != 2 {
		t.Fatal("bind preview persisted before confirmation")
	}

	rejectedResponse, err := handler.SurePetEquModBind(ctx, []interface{}{float64(-1)})
	if err != nil {
		t.Fatalf("SurePetEquModBind(reject) error = %v", err)
	}
	assertPetEquipmentValue(t, rejectedResponse.(map[string]interface{}), "f", "false")
	if itemRepo.items[9001].Properties["bindMainPropNum1"] != 2 {
		t.Fatal("rejected bind preview mutated equipment")
	}

	_, err = handler.PetEquModBind(ctx, []interface{}{float64(9001), float64(9103)})
	if err != nil {
		t.Fatalf("PetEquModBind(second) error = %v", err)
	}
	acceptedResponse, err := handler.SurePetEquModBind(ctx, []interface{}{float64(1)})
	if err != nil {
		t.Fatalf("SurePetEquModBind(accept) error = %v", err)
	}
	accepted := acceptedResponse.(map[string]interface{})
	assertPetEquipmentValue(t, accepted, "f", "sure")
	assertPetEquipmentValue(t, accepted, "b1", 9)
	assertPetEquipmentValue(t, accepted, "b2", 10)
	if itemRepo.items[9001].Properties["bindMainPropNum1"] != 9 || itemRepo.items[9001].Properties["bindMainPropNum2"] != 10 {
		t.Fatalf("accepted bind values = %#v", itemRepo.items[9001].Properties)
	}
	if char.MoneyBind != 140000 || itemRepo.items[9103].StackCount != 1 {
		t.Fatalf("two previews cost state = money %d material %d", char.MoneyBind, itemRepo.items[9103].StackCount)
	}
	if !hasPetEquipmentCallback(calls.calls, "onRefreshPetProp") {
		t.Fatal("accepted equipped bind missing pet refresh")
	}

	reusedResponse, err := handler.SurePetEquModBind(ctx, []interface{}{float64(1)})
	if err != nil {
		t.Fatalf("SurePetEquModBind(reuse) error = %v", err)
	}
	assertPetEquipmentValue(t, reusedResponse.(map[string]interface{}), "f", "false")
}

func TestPetEquipmentSpecialQuantityRules(t *testing.T) {
	wrapped := &models.EquiptTemplateTemplate{Position: 56, ReqClassID: "|1001|"}
	unwrapped := &models.EquiptTemplateTemplate{Position: 57, ReqClassID: "1001"}
	normal := &models.EquiptTemplateTemplate{Position: 55, ReqClassID: "|1001|"}
	if got := petEquipmentSpecialQuantity(wrapped, 5, 2, 1.5); got != 10 {
		t.Fatalf("wrapped level quantity = %d, want 10", got)
	}
	if got := petEquipmentSpecialQuantity(unwrapped, 5, 2, 1.5); got != 8 {
		t.Fatalf("unwrapped level quantity = %d, want 8", got)
	}
	if got := petEquipmentSpecialQuantity(wrapped, 2, 4, 2); got != 8 {
		t.Fatalf("wrapped color/bind quantity = %d, want 8", got)
	}
	if got := petEquipmentSpecialQuantity(normal, 5, 2, 1.5); got != 5 {
		t.Fatalf("normal quantity = %d, want 5", got)
	}
}

func buildPetEquipmentModifyHandler(t *testing.T) (*Handler, *infrartmp.RPCContext, *itemTestRepo, *petEquipmentModifyPetRepo, *domainchar.Character, *equipCallbackTransport) {
	t.Helper()
	character := &domainchar.Character{ID: 1, Name: "tester", MoneyBind: 200000, SelectedMoneyType: 1}
	equipment := &domainitem.Item{
		ID: 9001, CharacterID: 1, TemplateID: 5001, ItemType: domainitem.ItemTypeEquipment,
		SlotType: domainitem.SlotTypePetEquipped, SlotIndex: 0, StackCount: 1, IsBound: true, StarLevel: 7,
		ColorCode: 1, Properties: map[string]interface{}{"color": "0", "bindMainPropNum1": 2, "bindMainPropNum2": 3, "custom": "keep"},
	}
	levelMaterial := &domainitem.Item{ID: 9101, CharacterID: 1, TemplateID: 2333, ItemType: domainitem.ItemTypeMaterial, SlotType: domainitem.SlotTypeBag, SlotIndex: 10, StackCount: 10}
	colorMaterial := &domainitem.Item{ID: 9102, CharacterID: 1, TemplateID: 2334, ItemType: domainitem.ItemTypeMaterial, SlotType: domainitem.SlotTypeBag, SlotIndex: 11, StackCount: 10}
	bindMaterial := &domainitem.Item{ID: 9103, CharacterID: 1, TemplateID: 2809, ItemType: domainitem.ItemTypeMaterial, SlotType: domainitem.SlotTypeBag, SlotIndex: 12, StackCount: 3}
	itemRepo := newItemTestRepo(equipment, levelMaterial, colorMaterial, bindMaterial)
	handler, ctx := buildItemTestHandlerWithRepo(t, character, itemRepo)
	manager := newPetEquipmentModifyGameData(t)
	handler.gameData = manager
	handler.itemService.SetGameDataManager(manager)

	petRepo := &petEquipmentModifyPetRepo{pets: map[int64]*domainpet.Pet{
		7001: {
			ID: 7001, CharacterID: 1, TemplateID: 1001, Level: 40,
			AptStrength: 10, AptAgility: 10, AptStamina: 10, AptIntelligence: 10, AptEnergy: 10,
			GrowRate: 1, CurrentHP: 20, CurrentMP: 20, Property: map[string]interface{}{"equ1": int64(9001)},
		},
	}}
	petService := apppet.NewService(petRepo, zaptest.NewLogger(t))
	petService.SetItemRepository(itemRepo)
	petService.SetGameDataManager(manager)
	handler.SetPetService(petService)
	calls := &equipCallbackTransport{}
	ctx.Connection.SetTransport(calls)
	return handler, ctx, itemRepo, petRepo, character, calls
}

func newPetEquipmentModifyGameData(t *testing.T) *gamedata.Manager {
	t.Helper()
	manager := gamedata.NewManager(nil, zaptest.NewLogger(t))
	load := func(table string, values ...interface{}) {
		t.Helper()
		rows := make([]json.RawMessage, 0, len(values))
		for _, value := range values {
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("marshal %s: %v", table, err)
			}
			rows = append(rows, raw)
		}
		if err := manager.GetCache().LoadTable(table, rows); err != nil {
			t.Fatalf("load %s: %v", table, err)
		}
	}
	load(models.TableCreature, models.CreatureTemplate{
		ID: 1001, Name: "Pet", AptStrength: 10, AptAgility: 10, AptStamina: 10, AptIntelligence: 10, AptEnergy: 10,
		GrowBase: 1, QLevel: 1, ClassID: 3, ClassIds: 8,
	})
	load(models.TableEquiptTemplate,
		models.EquiptTemplateTemplate{ID: 5001, Name: "Pet Fang 40", Kind: 9, UseType: 2, Type: 900, Position: 50, ReqLevel: 40, NextEquTid: 5002, MainProp1: 20, MainProp2: 21, MainPropNum1: 5, MainPropNum2: 2, BindPropNum: 10},
		models.EquiptTemplateTemplate{ID: 5002, Name: "Pet Fang 60", Kind: 9, UseType: 2, Type: 900, Position: 50, ReqLevel: 60, NextEquTid: 5003, MainProp1: 20, MainProp2: 21, MainPropNum1: 15, MainPropNum2: 4, BindPropNum: 10},
		models.EquiptTemplateTemplate{ID: 5003, Name: "Pet Fang 80", Kind: 9, UseType: 2, Type: 900, Position: 50, ReqLevel: 80, MainProp1: 20, MainPropNum1: 20, BindPropNum: 10},
	)
	return manager
}

func assertPetEquipmentValue(t *testing.T, values map[string]interface{}, key string, want interface{}) {
	t.Helper()
	if got := values[key]; got != want {
		t.Fatalf("%s = %#v, want %#v in %#v", key, got, want, values)
	}
}

func hasPetEquipmentCallback(calls []callbackCall, method string) bool {
	for _, call := range calls {
		if call.method == method {
			return true
		}
	}
	return false
}
