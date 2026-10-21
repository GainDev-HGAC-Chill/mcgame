// Open-sourced by BaoLT

package pet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type bookSkillItemRepo struct {
	items          map[int64]*domainitem.Item
	updateStackErr error
	deleteErr      error
}

func (r *bookSkillItemRepo) FindByID(ctx context.Context, id int64) (*domainitem.Item, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, pkgerrors.ErrItemNotFound
	}
	cloned := *item
	cloned.Properties = clonePetMap(item.Properties)
	return &cloned, nil
}

func (r *bookSkillItemRepo) FindByCharacterID(ctx context.Context, charID int64) ([]*domainitem.Item, error) {
	items := make([]*domainitem.Item, 0)
	for _, item := range r.items {
		if item.CharacterID != charID {
			continue
		}
		cloned := *item
		cloned.Properties = clonePetMap(item.Properties)
		items = append(items, &cloned)
	}
	return items, nil
}

func (r *bookSkillItemRepo) FindByCharacterAndSlotType(ctx context.Context, charID int64, slotType domainitem.SlotType) ([]*domainitem.Item, error) {
	items := make([]*domainitem.Item, 0)
	for _, item := range r.items {
		if item.CharacterID != charID || item.SlotType != slotType {
			continue
		}
		cloned := *item
		cloned.Properties = clonePetMap(item.Properties)
		items = append(items, &cloned)
	}
	return items, nil
}

func (r *bookSkillItemRepo) FindBySlot(ctx context.Context, charID int64, slotType domainitem.SlotType, slotIndex int) (*domainitem.Item, error) {
	return nil, pkgerrors.ErrItemNotFound
}

func (r *bookSkillItemRepo) FindEquipped(ctx context.Context, charID int64) ([]*domainitem.Item, error) {
	return nil, nil
}

func (r *bookSkillItemRepo) Create(ctx context.Context, item *domainitem.Item) error {
	return nil
}

func (r *bookSkillItemRepo) Update(ctx context.Context, item *domainitem.Item) error {
	return nil
}

func (r *bookSkillItemRepo) Delete(ctx context.Context, id int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.items, id)
	return nil
}

func (r *bookSkillItemRepo) DeleteByCharacterID(ctx context.Context, charID int64) error {
	return nil
}

func (r *bookSkillItemRepo) MoveItem(ctx context.Context, id int64, slotType domainitem.SlotType, slotIndex int) error {
	return nil
}

func (r *bookSkillItemRepo) UpdateStack(ctx context.Context, id int64, stackCount int) error {
	if r.updateStackErr != nil {
		return r.updateStackErr
	}
	item, ok := r.items[id]
	if !ok {
		return pkgerrors.ErrItemNotFound
	}
	item.StackCount = stackCount
	return nil
}

func (r *bookSkillItemRepo) FindFirstEmptySlot(ctx context.Context, charID int64, slotType domainitem.SlotType, maxSlots int) (int, error) {
	return 0, nil
}

func (r *bookSkillItemRepo) CountBySlotType(ctx context.Context, charID int64, slotType domainitem.SlotType) (int, error) {
	return 0, nil
}

func TestLearnSkillFromBookUsesItemInstanceAndAppliesBuff(t *testing.T) {
	petRepo := &openSkillPetRepo{
		pets: map[int64]*domainpet.Pet{
			2484410: {
				ID:              2484410,
				CharacterID:     1,
				TemplateID:      1001,
				Level:           1,
				AptStrength:     10,
				AptAgility:      10,
				AptStamina:      10,
				AptIntelligence: 10,
				AptEnergy:       10,
				GrowRate:        1,
				CreatureData: map[string]interface{}{
					"classId":  3,
					"classIds": 3,
					"qLevel":   3,
				},
				Property: map[string]interface{}{
					"skill1": 2830,
					"skill2": 3345,
					"skill3": 5156,
				},
			},
		},
	}
	itemRepo := &bookSkillItemRepo{
		items: map[int64]*domainitem.Item{
			9001: {
				ID:          9001,
				CharacterID: 1,
				TemplateID:  3216,
				StackCount:  2,
			},
		},
	}

	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(newPetSkillBookGameData(t))
	if spec := petSuperSkillBooks[3216]; service.validatePetSuperSkillBookCatalog(service.gameDataManager.GetItem(3216), spec) == nil {
		t.Fatalf("invalid fixture: item=%#v pre=%#v/%#v/%#v super=%#v/%#v/%#v", service.gameDataManager.GetItem(3216), service.gameDataManager.GetSkill(2830), service.gameDataManager.GetSkill(3345), service.gameDataManager.GetSkill(5156), service.gameDataManager.GetSkill(5527), service.gameDataManager.GetSkill(5528), service.gameDataManager.GetSkill(5529))
	}

	result, err := service.LearnSkillFromBook(context.Background(), 1, 2484410, 9001)
	if err != nil {
		t.Fatalf("LearnSkillFromBook() error = %v", err)
	}
	if result.Slot != 1 {
		t.Fatalf("LearnSkillFromBook() slot = %d, want 1", result.Slot)
	}
	if result.SkillID != 5527 {
		t.Fatalf("LearnSkillFromBook() skillID = %d, want 5527", result.SkillID)
	}
	if len(result.SlotUpdates) != 3 || result.SlotUpdates[0].Slot != 1 || result.SlotUpdates[0].SkillID != 5527 || result.SlotUpdates[1].Slot != 2 || result.SlotUpdates[1].SkillID != 0 || result.SlotUpdates[2].Slot != 3 || result.SlotUpdates[2].SkillID != 0 {
		t.Fatalf("LearnSkillFromBook() updates = %#v", result.SlotUpdates)
	}
	if result.RemainingStack != 1 {
		t.Fatalf("LearnSkillFromBook() remainingStack = %d, want 1", result.RemainingStack)
	}

	storedPet := petRepo.pets[2484410]
	if petSkillSlotValue(storedPet, 1) != 5527 {
		t.Fatalf("stored skill1 = %d, want 5527", petSkillSlotValue(storedPet, 1))
	}
	if petSkillSlotValue(storedPet, 2) != 0 || petSkillSlotValue(storedPet, 3) != 0 {
		t.Fatalf("consumed prerequisite slots = %d/%d, want 0/0", petSkillSlotValue(storedPet, 2), petSkillSlotValue(storedPet, 3))
	}
	if storedPet.MaxHP != 24 {
		t.Fatalf("stored MaxHP = %d, want 24", storedPet.MaxHP)
	}
	if finalHP, ok := storedPet.Property["finalHp"].(int); !ok || finalHP != 24 {
		t.Fatalf("stored finalHp = %v, want 24", storedPet.Property["finalHp"])
	}

	storedItem := itemRepo.items[9001]
	if storedItem.StackCount != 1 {
		t.Fatalf("stored item stack = %d, want 1", storedItem.StackCount)
	}
}

func TestLearnSkillFromBookRejectsDuplicateSkillFamily(t *testing.T) {
	petRepo := &openSkillPetRepo{
		pets: map[int64]*domainpet.Pet{
			2484410: {
				ID:              2484410,
				CharacterID:     1,
				TemplateID:      1001,
				Level:           1,
				AptStrength:     10,
				AptAgility:      10,
				AptStamina:      10,
				AptIntelligence: 10,
				AptEnergy:       10,
				GrowRate:        1,
				CreatureData: map[string]interface{}{
					"classId":  3,
					"classIds": 3,
					"qLevel":   3,
				},
				Property: map[string]interface{}{
					"skill1": 5528,
				},
			},
		},
	}
	itemRepo := &bookSkillItemRepo{
		items: map[int64]*domainitem.Item{
			9001: {
				ID:          9001,
				CharacterID: 1,
				TemplateID:  3216,
				StackCount:  2,
			},
		},
	}

	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	gameData := newPetSkillBookGameData(t)
	addUniversalPetSkillBookGameData(t, gameData)
	service.SetGameDataManager(gameData)

	_, err := service.LearnSkillFromBook(context.Background(), 1, 2484410, 9001)
	if err == nil {
		t.Fatalf("LearnSkillFromBook() error = nil, want duplicate failure")
	}

	var bookErr *PetSkillBookError
	if !errors.As(err, &bookErr) {
		t.Fatalf("LearnSkillFromBook() error type = %T, want *PetSkillBookError", err)
	}
	if bookErr.Failure != PetSkillBookFailureLearn {
		t.Fatalf("LearnSkillFromBook() failure = %d, want %d", bookErr.Failure, PetSkillBookFailureLearn)
	}
}

func TestLearnSkillFromBookRejectsInvalidBookTemplate(t *testing.T) {
	petRepo := &openSkillPetRepo{
		pets: map[int64]*domainpet.Pet{
			2484410: {
				ID:              2484410,
				CharacterID:     1,
				TemplateID:      1001,
				Level:           1,
				AptStrength:     10,
				AptAgility:      10,
				AptStamina:      10,
				AptIntelligence: 10,
				AptEnergy:       10,
				GrowRate:        1,
				CreatureData: map[string]interface{}{
					"classId":  3,
					"classIds": 3,
					"qLevel":   3,
				},
				Property: map[string]interface{}{},
			},
		},
	}
	itemRepo := &bookSkillItemRepo{
		items: map[int64]*domainitem.Item{
			9002: {
				ID:          9002,
				CharacterID: 1,
				TemplateID:  4000,
				StackCount:  1,
			},
		},
	}

	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(newPetSkillBookGameData(t))

	_, err := service.LearnSkillFromBook(context.Background(), 1, 2484410, 9002)
	if err == nil {
		t.Fatalf("LearnSkillFromBook() error = nil, want invalid book failure")
	}

	var bookErr *PetSkillBookError
	if !errors.As(err, &bookErr) {
		t.Fatalf("LearnSkillFromBook() error type = %T, want *PetSkillBookError", err)
	}
	if bookErr.Failure != PetSkillBookFailureBook {
		t.Fatalf("LearnSkillFromBook() failure = %d, want %d", bookErr.Failure, PetSkillBookFailureBook)
	}
}

func TestLearnSkillFromBookRejectsWhenNoOpenSkillSlot(t *testing.T) {
	petRepo := &openSkillPetRepo{
		pets: map[int64]*domainpet.Pet{
			2484410: {
				ID:              2484410,
				CharacterID:     1,
				TemplateID:      1001,
				Level:           1,
				AptStrength:     10,
				AptAgility:      10,
				AptStamina:      10,
				AptIntelligence: 10,
				AptEnergy:       10,
				GrowRate:        1,
				CreatureData: map[string]interface{}{
					"classId":  3,
					"classIds": 3,
					"qLevel":   3,
				},
				Property: map[string]interface{}{
					"skill1":  5001,
					"skill2":  5002,
					"skill3":  5003,
					"skill4":  5004,
					"skill5":  5005,
					"skill6":  -1,
					"skill7":  -1,
					"skill8":  -1,
					"skill9":  -1,
					"skill10": -1,
					"skill11": -1,
					"skill12": -1,
					"skill13": -1,
					"skill14": -1,
					"skill15": -1,
				},
			},
		},
	}
	itemRepo := &bookSkillItemRepo{
		items: map[int64]*domainitem.Item{
			9001: {
				ID:          9001,
				CharacterID: 1,
				TemplateID:  3216,
				StackCount:  1,
			},
		},
	}

	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(newPetSkillBookGameData(t))

	_, err := service.LearnSkillFromBook(context.Background(), 1, 2484410, 9001)
	if err == nil {
		t.Fatalf("LearnSkillFromBook() error = nil, want no-slot failure")
	}

	var bookErr *PetSkillBookError
	if !errors.As(err, &bookErr) {
		t.Fatalf("LearnSkillFromBook() error type = %T, want *PetSkillBookError", err)
	}
	if bookErr.Failure != PetSkillBookFailureLearn {
		t.Fatalf("LearnSkillFromBook() failure = %d, want %d", bookErr.Failure, PetSkillBookFailureLearn)
	}
}

func TestLearnSkillFromBookAllowsDifferentPetFamilyAtRequiredQuality(t *testing.T) {
	petRepo := &openSkillPetRepo{
		pets: map[int64]*domainpet.Pet{
			2484410: {
				ID:              2484410,
				CharacterID:     1,
				TemplateID:      1002,
				Level:           1,
				AptStrength:     10,
				AptAgility:      10,
				AptStamina:      10,
				AptIntelligence: 10,
				AptEnergy:       10,
				GrowRate:        1,
				CreatureData: map[string]interface{}{
					"classId":  3,
					"classIds": 2,
					"qLevel":   3,
				},
				Property: map[string]interface{}{
					"skill1": 2830,
					"skill2": 3345,
					"skill3": 5156,
				},
			},
		},
	}
	itemRepo := &bookSkillItemRepo{
		items: map[int64]*domainitem.Item{
			9001: {
				ID:          9001,
				CharacterID: 1,
				TemplateID:  3216,
				StackCount:  1,
			},
		},
	}

	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(newPetSkillBookGameData(t))

	result, err := service.LearnSkillFromBook(context.Background(), 1, 2484410, 9001)
	if err != nil {
		t.Fatalf("LearnSkillFromBook() error = %v", err)
	}
	if result.SkillID != 5527 {
		t.Fatalf("LearnSkillFromBook() skill = %d, want 5527", result.SkillID)
	}
}

func TestLearnSkillFromBookAllowsUniversalPetSkill(t *testing.T) {
	petRepo := &openSkillPetRepo{
		pets: map[int64]*domainpet.Pet{
			2484410: {
				ID:              2484410,
				CharacterID:     1,
				TemplateID:      1002,
				Level:           1,
				AptStrength:     10,
				AptAgility:      10,
				AptStamina:      10,
				AptIntelligence: 10,
				AptEnergy:       10,
				GrowRate:        1,
				CreatureData: map[string]interface{}{
					"classId": 2,
					"qLevel":  3,
				},
				Property: map[string]interface{}{},
			},
		},
	}
	itemRepo := &bookSkillItemRepo{
		items: map[int64]*domainitem.Item{
			9003: {
				ID:          9003,
				CharacterID: 1,
				TemplateID:  4001,
				StackCount:  1,
			},
		},
	}

	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	gameData := newPetSkillBookGameData(t)
	addUniversalPetSkillBookGameData(t, gameData)
	service.SetGameDataManager(gameData)

	result, err := service.LearnSkillFromBook(context.Background(), 1, 2484410, 9003)
	if err != nil {
		t.Fatalf("LearnSkillFromBook() error = %v", err)
	}
	if result.SkillID != 6001 {
		t.Fatalf("LearnSkillFromBook() skillID = %d, want 6001", result.SkillID)
	}
}

func TestLearnSkillFromBookRejectsOrdinaryTierWhenSuperFamilyOwned(t *testing.T) {
	manager := newPetSkillBookGameData(t)
	cache := manager.GetCache()
	if err := cache.LoadTable(models.TableItemTemplate, []json.RawMessage{
		json.RawMessage(`{"id":1374,"name":"Thụ Bì Ngoại Sáo","type":506,"use_type":2}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.LoadTable(models.TableSkill, []json.RawMessage{
		json.RawMessage(`{"id":2825,"name":"Thụ Bì Ngoại Sáo","code_name":"SKILL450801","kind":2,"level":1,"use_env":1,"cre_kind":"3|1"}`),
	}); err != nil {
		t.Fatal(err)
	}
	petRepo := &openSkillPetRepo{pets: map[int64]*domainpet.Pet{
		1: {ID: 1, CharacterID: 7, TemplateID: 1001, Level: 1, GrowRate: 1, Property: map[string]interface{}{"skill1": 5529}},
	}}
	itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		9: {ID: 9, CharacterID: 7, TemplateID: 1374, StackCount: 1},
	}}
	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(manager)

	_, err := service.LearnSkillFromBook(context.Background(), 7, 1, 9)
	var bookErr *PetSkillBookError
	if !errors.As(err, &bookErr) || bookErr.Failure != PetSkillBookFailureLearn {
		t.Fatalf("error = %v, want non-consuming learn failure", err)
	}
	if petSkillSlotValue(petRepo.pets[1], 1) != 5529 || itemRepo.items[9].StackCount != 1 {
		t.Fatal("lower-tier rejection mutated Pet or book")
	}
}

func newPetSkillBookGameData(t *testing.T) *gamedata.Manager {
	t.Helper()

	manager := gamedata.NewManager(nil, zap.NewNop())
	cache := manager.GetCache()

	loadTable := func(table string, rows ...string) {
		t.Helper()
		records := make([]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			records = append(records, json.RawMessage(row))
		}
		if err := cache.LoadTable(table, records); err != nil {
			t.Fatalf("LoadTable(%s) error = %v", table, err)
		}
	}

	loadTable(models.TableCreature,
		`{"id":1001,"name":"Test Pet","apt_strength":10,"apt_agility":10,"apt_stamina":10,"apt_intelligence":10,"apt_energy":10,"grow_base":1,"element":1,"q_level":3,"class_id":3,"class_ids":3}`,
		`{"id":1002,"name":"Other Test Pet","apt_strength":10,"apt_agility":10,"apt_stamina":10,"apt_intelligence":10,"apt_energy":10,"grow_base":1,"element":1,"q_level":3,"class_id":3,"class_ids":2}`,
	)
	loadTable(models.TableItemTemplate,
		`{"id":3216,"name":"Thụ Bì Ngoại Sáo Siêu Cấp","kind":5,"type":506,"use_type":2,"prop_type":237,"propl_num":3,"req_class":"|1|2|3|4|5|6|","skill_id":0}`,
		`{"id":4000,"name":"Táo","kind":5,"type":550,"use_type":1,"prop_type":0,"skill_id":0}`,
	)
	loadTable(models.TableSkill,
		`{"id":2830,"name":"Thụ Bì Ngoại Sáo","code_name":"thu_bi_ngoai_sao","buff_id":-1,"kind":2,"level":3,"cre_kind":"3|1","use_env":1}`,
		`{"id":3345,"name":"Thụ Bì Ngoại Sáo Trung Cấp","code_name":"thu_bi_ngoai_sao_mid","buff_id":-1,"kind":2,"level":3,"cre_kind":"3|2","use_env":1}`,
		`{"id":5156,"name":"Thụ Bì Ngoại Sáo Cao Cấp","code_name":"thu_bi_ngoai_sao_high","buff_id":-1,"kind":2,"level":3,"cre_kind":"3|3","use_env":1}`,
		`{"id":5527,"name":"Thụ Bì Ngoại Sáo Siêu Cấp","code_name":"thu_bi_ngoai_sao_super","buff_id":2113,"kind":2,"level":1,"cre_kind":"3|3","use_env":1}`,
		`{"id":5528,"name":"Thụ Bì Ngoại Sáo Siêu Cấp","code_name":"thu_bi_ngoai_sao_super","buff_id":2114,"kind":2,"level":2,"cre_kind":"3|3","use_env":1}`,
		`{"id":5529,"name":"Thụ Bì Ngoại Sáo Siêu Cấp","code_name":"thu_bi_ngoai_sao_super","buff_id":2115,"kind":2,"level":3,"cre_kind":"3|3","use_env":1}`,
	)
	loadTable(models.TableBuff,
		`{"id":2113,"percent_flag":1,"prop1":1,"prop_num1":22}`,
		`{"id":2114,"percent_flag":1,"prop1":1,"prop_num1":31}`,
	)

	return manager
}

func addUniversalPetSkillBookGameData(t *testing.T, manager *gamedata.Manager) {
	t.Helper()

	cache := manager.GetCache()
	loadTable := func(table string, rows ...string) {
		t.Helper()
		records := make([]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			records = append(records, json.RawMessage(row))
		}
		if err := cache.LoadTable(table, records); err != nil {
			t.Fatalf("LoadTable(%s) error = %v", table, err)
		}
	}

	loadTable(models.TableItemTemplate,
		`{"id":4001,"name":"Man Lực","kind":5,"type":506,"use_type":2,"prop_type":205,"skill_id":6001}`,
	)
	loadTable(models.TableSkill,
		`{"id":6001,"name":"Man Lực","code_name":"man_luc","buff_id":2200,"level":1,"cre_kind":"0"}`,
	)
	loadTable(models.TableBuff,
		`{"id":2200,"percent_flag":0,"prop1":4,"prop_num1":12}`,
	)
}

func TestResolvePetSkillBookSkillExcludesNonPetDuplicateName(t *testing.T) {
	manager := newPetSkillBookGameData(t)
	cache := manager.GetCache()
	if err := cache.LoadTable(models.TableItemTemplate, []json.RawMessage{
		json.RawMessage(`{"id":2242,"name":"Man Lực","type":506,"use_type":2,"skill_id":0}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.LoadTable(models.TableSkill, []json.RawMessage{
		json.RawMessage(`{"id":4172,"name":"Man Lực","code_name":"SKILL505701","kind":2,"level":1,"use_env":1}`),
		json.RawMessage(`{"id":4171,"name":"Man Lực","code_name":"SKILL505701","kind":2,"level":2,"use_env":1}`),
		json.RawMessage(`{"id":4170,"name":"Man Lực","code_name":"SKILL505701","kind":2,"level":3,"use_env":1}`),
		json.RawMessage(`{"id":60903,"name":"Man Lực","code_name":"SKILL1010703","kind":2,"level":1,"use_env":-1}`),
	}); err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, zap.NewNop())
	service.SetGameDataManager(manager)

	resolved := service.resolvePetSkillBookSkill(manager.GetItem(2242))
	if resolved == nil || resolved.GetID() != 4172 {
		t.Fatalf("resolvePetSkillBookSkill(Man Lực) = %#v, want skill 4172", resolved)
	}
}

func TestResolvePetSkillBookSkillAllowsPetUseEnvironmentTwo(t *testing.T) {
	manager := newPetSkillBookGameData(t)
	cache := manager.GetCache()
	if err := cache.LoadTable(models.TableItemTemplate, []json.RawMessage{
		json.RawMessage(`{"id":1622,"name":"Thuốc Kích Thích","type":506,"use_type":2,"skill_id":0}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.LoadTable(models.TableSkill, []json.RawMessage{
		json.RawMessage(`{"id":3590,"name":"Thuốc Kích Thích","code_name":"SKILL483301","level":1,"use_env":2,"cre_kind":"1|3"}`),
		json.RawMessage(`{"id":3672,"name":"Thuốc Kích Thích","code_name":"SKILL483301","level":2,"use_env":2,"cre_kind":"1|3"}`),
	}); err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, zap.NewNop())
	service.SetGameDataManager(manager)

	resolved := service.resolvePetSkillBookSkill(manager.GetItem(1622))
	if resolved == nil || resolved.GetID() != 3590 {
		t.Fatalf("resolvePetSkillBookSkill(Thuốc Kích Thích) = %#v, want skill 3590", resolved)
	}
}

func TestLearnReportedPetSkillBooksByClassIDs(t *testing.T) {
	tests := []struct {
		name     string
		itemID   int
		skillID  int
		classIDs int
		qLevel   int
	}{
		{name: "Báo Chi Tấn Tiệp", itemID: 1382, skillID: 2741, classIDs: 2, qLevel: 1},
		{name: "Cổ Long Chi Hồn", itemID: 1377, skillID: 2820, classIDs: 6, qLevel: 1},
		{name: "Cổ Long Chi Hồn Trung Cấp", itemID: 1389, skillID: 3352, classIDs: 6, qLevel: 2},
		{name: "Ngạnh Xác Hóa", itemID: 1378, skillID: 2696, classIDs: 4, qLevel: 1},
		{name: "Ngạnh Xác Hóa Trung Cấp", itemID: 1390, skillID: 3355, classIDs: 4, qLevel: 2},
		{name: "Vẻ Ngoài Yếu Ớt", itemID: 1379, skillID: 2841, classIDs: 5, qLevel: 1},
		{name: "Thụ Bì Ngoại Sáo", itemID: 1374, skillID: 2825, classIDs: 3, qLevel: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := gamedata.NewManager(nil, zap.NewNop())
			cache := manager.GetCache()
			load := func(table string, records ...json.RawMessage) {
				t.Helper()
				if err := cache.LoadTable(table, records); err != nil {
					t.Fatal(err)
				}
			}
			load(models.TableCreature, json.RawMessage(`{"id":1001,"name":"Test Pet","apt_strength":10,"apt_agility":10,"apt_stamina":10,"apt_intelligence":10,"apt_energy":10,"grow_base":1,"q_level":3,"class_id":1,"class_ids":10}`))
			load(models.TableItemTemplate, json.RawMessage(fmt.Sprintf(`{"id":%d,"name":%q,"type":506,"use_type":2}`, test.itemID, test.name)))
			load(models.TableSkill, json.RawMessage(fmt.Sprintf(`{"id":%d,"name":%q,"kind":2,"level":1,"use_env":1,"cre_kind":%q}`, test.skillID, test.name, fmt.Sprintf("%d|%d", test.classIDs, test.qLevel))))

			petRepo := &openSkillPetRepo{pets: map[int64]*domainpet.Pet{1: {ID: 1, CharacterID: 7, TemplateID: 1001, Level: 1, GrowRate: 1, Property: map[string]interface{}{}}}}
			itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{9001: {ID: 9001, CharacterID: 7, TemplateID: test.itemID, StackCount: 1}}}
			service := NewService(petRepo, zap.NewNop())
			service.SetItemRepository(itemRepo)
			service.SetGameDataManager(manager)

			result, err := service.LearnSkillFromBook(context.Background(), 7, 1, 9001)
			if err != nil {
				t.Fatalf("LearnSkillFromBook() error = %v", err)
			}
			if result.SkillID != test.skillID || petSkillSlotValue(result.Pet, 1) != test.skillID {
				t.Fatalf("learned skill = %d/%d, want %d", result.SkillID, petSkillSlotValue(result.Pet, 1), test.skillID)
			}
		})
	}
}
