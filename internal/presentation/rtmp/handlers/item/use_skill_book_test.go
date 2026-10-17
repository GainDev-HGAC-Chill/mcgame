// Open-sourced by BaoLT

package item

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	appskill "mcgame-server/internal/application/skill"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap/zaptest"
)

type itemSkillTestRepo struct {
	nextID int64
	skills map[int64]*domainskill.CharacterSkill
}

type itemMysteryRecipeLearner struct {
	learned map[int64]map[int]bool
}

func (l *itemMysteryRecipeLearner) LearnRecipeByFormula(_ context.Context, characterID int64, itemTemplateID int) (int, error) {
	if l.learned == nil {
		l.learned = make(map[int64]map[int]bool)
	}
	if l.learned[characterID] == nil {
		l.learned[characterID] = make(map[int]bool)
	}
	recipeID := itemTemplateID - 5041
	l.learned[characterID][recipeID] = true
	return recipeID, nil
}

func (l *itemMysteryRecipeLearner) ForgetRecipe(_ context.Context, characterID int64, recipeID int) error {
	delete(l.learned[characterID], recipeID)
	return nil
}

func newItemSkillTestRepo() *itemSkillTestRepo {
	return &itemSkillTestRepo{
		nextID: 1,
		skills: make(map[int64]*domainskill.CharacterSkill),
	}
}

func (r *itemSkillTestRepo) FindByID(ctx context.Context, id int64) (*domainskill.CharacterSkill, error) {
	sk, ok := r.skills[id]
	if !ok {
		return nil, pkgerrors.ErrNotFound
	}
	copy := *sk
	return &copy, nil
}

func (r *itemSkillTestRepo) FindByCharacterID(ctx context.Context, charID int64) ([]*domainskill.CharacterSkill, error) {
	result := make([]*domainskill.CharacterSkill, 0)
	for _, sk := range r.skills {
		if sk.CharacterID != charID {
			continue
		}
		copy := *sk
		result = append(result, &copy)
	}
	return result, nil
}

func (r *itemSkillTestRepo) FindByCharacterAndSkill(ctx context.Context, charID int64, skillID int) (*domainskill.CharacterSkill, error) {
	for _, sk := range r.skills {
		if sk.CharacterID == charID && sk.SkillID == skillID {
			copy := *sk
			return &copy, nil
		}
	}
	return nil, pkgerrors.ErrNotFound
}

func (r *itemSkillTestRepo) FindBySlot(ctx context.Context, charID int64, slot int) (*domainskill.CharacterSkill, error) {
	for _, sk := range r.skills {
		if sk.CharacterID == charID && sk.SlotPosition != nil && *sk.SlotPosition == slot {
			copy := *sk
			return &copy, nil
		}
	}
	return nil, pkgerrors.ErrNotFound
}

func (r *itemSkillTestRepo) Create(ctx context.Context, sk *domainskill.CharacterSkill) error {
	copy := *sk
	copy.ID = r.nextID
	r.nextID++
	r.skills[copy.ID] = &copy
	sk.ID = copy.ID
	return nil
}

func (r *itemSkillTestRepo) Update(ctx context.Context, sk *domainskill.CharacterSkill) error {
	copy := *sk
	r.skills[copy.ID] = &copy
	return nil
}

func (r *itemSkillTestRepo) Delete(ctx context.Context, id int64) error {
	delete(r.skills, id)
	return nil
}

func (r *itemSkillTestRepo) UpdateSlot(ctx context.Context, id int64, slot *int) error {
	return nil
}

func (r *itemSkillTestRepo) UpdateCooldown(ctx context.Context, id int64, cooldownEnd *string) error {
	return nil
}

func (r *itemSkillTestRepo) HasSkill(ctx context.Context, charID int64, skillID int) (bool, error) {
	for _, sk := range r.skills {
		if sk.CharacterID == charID && sk.SkillID == skillID {
			return true, nil
		}
	}
	return false, nil
}

func TestItemHandlerUseItemPlantSkillBookLearnsSkill(t *testing.T) {
	char := &domainchar.Character{ID: 1, Name: "tester", Level: 60, Experience: 4321}
	book := &domainitem.Item{
		ID:          240901,
		CharacterID: char.ID,
		TemplateID:  appskill.PlantSkillBookTemplateID,
		ItemType:    domainitem.ItemTypeConsumable,
		SlotType:    domainitem.SlotTypeBag,
		SlotIndex:   0,
		StackCount:  1,
	}

	handler, ctx := buildItemTestHandler(t, char, book)
	loadItemSkillBookTemplate(t, handler)
	loadPlantSkillTemplates(t, handler)

	skillRepo := newItemSkillTestRepo()
	skillService := appskill.NewService(skillRepo, zaptest.NewLogger(t))
	skillService.SetCharacterRepository(&itemTestCharacterRepo{char: char})
	skillService.SetGameDataManager(handler.gameData)
	handler.SetSkillService(skillService)

	resp, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(book.ID)})
	if err != nil {
		t.Fatalf("UseItem() error = %v", err)
	}

	result, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map response, got %T", resp)
	}
	if success, _ := result["success"].(bool); !success {
		t.Fatalf("expected success=true, got %#v", result["success"])
	}

	learned, err := skillService.GetSkill(ctx.Context, char.ID, 4965)
	if err != nil {
		t.Fatalf("expected learned plant skill, got %v", err)
	}
	if learned.Level != 1 {
		t.Fatalf("expected learned level 1, got %d", learned.Level)
	}
	if learned.Exp != char.PlantDex {
		t.Fatalf("expected learned exp %d, got %d", char.PlantDex, learned.Exp)
	}

	if _, err := handler.itemService.GetItemByID(ctx.Context, char.ID, book.ID); !pkgerrors.Is(err, pkgerrors.ErrItemNotFound) {
		t.Fatalf("expected skill book to be consumed, got %v", err)
	}

	skillDTOs, err := skillService.GetSkillsForCallback(ctx.Context, char.ID)
	if err != nil {
		t.Fatalf("GetSkillsForCallback() error = %v", err)
	}
	if len(skillDTOs) != 1 {
		t.Fatalf("expected 1 skill dto, got %d", len(skillDTOs))
	}
	if got := skillDTOs[0]["id"]; got != 4965 {
		t.Fatalf("expected callback id 4965, got %#v", got)
	}
	if got := skillDTOs[0]["sid"]; got != 4965 {
		t.Fatalf("expected callback sid 4965, got %#v", got)
	}
}

func TestItemHandlerUseItemOrdinarySkillBookLearnsSkill(t *testing.T) {
	char := &domainchar.Character{ID: 1, Name: "tester", ClassID: 1, Level: 20}
	book := &domainitem.Item{
		ID: 4401, CharacterID: char.ID, TemplateID: 44,
		ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 1,
	}
	handler, ctx := buildItemTestHandler(t, char, book)
	loadOrdinarySkillBookCatalog(t, handler)
	skillRepo := newItemSkillTestRepo()
	skillService := appskill.NewService(skillRepo, zaptest.NewLogger(t))
	skillService.SetCharacterRepository(&itemTestCharacterRepo{char: char})
	skillService.SetGameDataManager(handler.gameData)
	handler.SetSkillService(skillService)

	resp, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(book.ID)})
	if err != nil {
		t.Fatalf("UseItem() error = %v", err)
	}
	result, ok := resp.(map[string]interface{})
	if !ok || result["success"] != true {
		t.Fatalf("response = %#v", resp)
	}
	learned, err := skillService.GetSkill(ctx.Context, char.ID, 1010)
	if err != nil || learned.Level != 1 {
		t.Fatalf("learned = %#v, error = %v", learned, err)
	}
	if _, err := handler.itemService.GetItemByID(ctx.Context, char.ID, book.ID); !pkgerrors.Is(err, pkgerrors.ErrItemNotFound) {
		t.Fatalf("book was not consumed: %v", err)
	}
}

func TestItemHandlerUseItemFishingAndHerbBooksLearnLifeSkills(t *testing.T) {
	cases := []struct {
		bookID    int
		skillID   int
		skillType int
	}{
		{appskill.FishSkillBookTemplateID, 4931, appskill.FishSkillType},
		{appskill.HerbSkillBookTemplateID, 4975, appskill.HerbSkillType},
	}
	for _, tc := range cases {
		t.Run(strconv.Itoa(tc.bookID), func(t *testing.T) {
			char := &domainchar.Character{ID: 1, Name: "tester", Level: 60}
			book := &domainitem.Item{
				ID: int64(tc.bookID) * 100, CharacterID: char.ID, TemplateID: tc.bookID,
				ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 1,
			}
			handler, ctx := buildItemTestHandler(t, char, book)
			loadLifeSkillBookCatalog(t, handler, tc.bookID, tc.skillID, tc.skillType)
			skillRepo := newItemSkillTestRepo()
			skillService := appskill.NewService(skillRepo, zaptest.NewLogger(t))
			skillService.SetCharacterRepository(&itemTestCharacterRepo{char: char})
			skillService.SetGameDataManager(handler.gameData)
			handler.SetSkillService(skillService)

			if _, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(book.ID)}); err != nil {
				t.Fatalf("UseItem() error = %v", err)
			}
			if _, err := skillService.GetSkill(ctx.Context, char.ID, tc.skillID); err != nil {
				t.Fatalf("life skill missing: %v", err)
			}
		})
	}
}

func TestItemHandlerUseItemMysteryFormulaLearnsRecipe(t *testing.T) {
	char := &domainchar.Character{ID: 1, Name: "tester", Level: 60}
	book := &domainitem.Item{
		ID: 504701, CharacterID: char.ID, TemplateID: 5047,
		ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 1,
	}
	handler, ctx := buildItemTestHandler(t, char, book)
	loadMysteryFormulaTemplate(t, handler)
	learner := &itemMysteryRecipeLearner{}
	handler.SetMysteryRecipeLearner(learner)

	resp, err := handler.UseItem(ctx, []interface{}{float64(1), float64(-1), float64(book.ID)})
	if err != nil {
		t.Fatalf("UseItem() error = %v", err)
	}
	result, ok := resp.(map[string]interface{})
	if !ok || result["success"] != true {
		t.Fatalf("response = %#v", resp)
	}
	if !learner.learned[char.ID][6] {
		t.Fatalf("recipe 6 was not learned: %#v", learner.learned)
	}
	if _, err := handler.itemService.GetItemByID(ctx.Context, char.ID, book.ID); !pkgerrors.Is(err, pkgerrors.ErrItemNotFound) {
		t.Fatalf("formula book was not consumed: %v", err)
	}
}

func loadItemSkillBookTemplate(t *testing.T, handler *Handler) {
	t.Helper()

	raw := itemSkillRawJSON(t, map[string]interface{}{
		"id":        appskill.PlantSkillBookTemplateID,
		"kind":      1,
		"type":      1,
		"use_type":  2,
		"req_level": 50,
	})
	if err := handler.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load skill book item template: %v", err)
	}
}

func loadOrdinarySkillBookCatalog(t *testing.T, handler *Handler) {
	t.Helper()
	itemRaw := itemSkillRawJSON(t, map[string]interface{}{
		"id": 44, "name": "Sách Kỹ Năng - Khai Sơn Chi Lực", "kind": 5, "type": 505,
		"use_type": 1, "req_level": 1, "req_class": "|1|",
	})
	skillRaw := itemSkillRawJSON(t, map[string]interface{}{
		"id": 1010, "name": "Khai Sơn Chi Lực", "type": 1, "kind": 1, "use_env": 0,
		"level": 1, "req_level": 1, "req_class": "|1|", "code_name": "SKILL111001",
	})
	if err := handler.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{itemRaw}); err != nil {
		t.Fatalf("load ordinary book: %v", err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableSkill, []json.RawMessage{skillRaw}); err != nil {
		t.Fatalf("load ordinary skill: %v", err)
	}
}

func loadLifeSkillBookCatalog(t *testing.T, handler *Handler, bookID, skillID, skillType int) {
	t.Helper()
	itemRaw := itemSkillRawJSON(t, map[string]interface{}{
		"id": bookID, "kind": 5, "type": 505, "use_type": 2, "req_level": 50,
	})
	skillRaw := itemSkillRawJSON(t, map[string]interface{}{
		"id": skillID, "name": "Life", "type": skillType, "kind": 2, "use_env": 6,
		"level": 1, "req_level": 50, "req_class": "|1|2|3|4|5|6|", "code_name": "life",
	})
	if err := handler.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{itemRaw}); err != nil {
		t.Fatalf("load life book: %v", err)
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableSkill, []json.RawMessage{skillRaw}); err != nil {
		t.Fatalf("load life skill: %v", err)
	}
}

func loadMysteryFormulaTemplate(t *testing.T, handler *Handler) {
	t.Helper()
	raw := itemSkillRawJSON(t, map[string]interface{}{
		"id": 5047, "name": "Bí Kíp Chế Tạo Mật Bảo Hang Động Cấp 1", "kind": 5,
		"type": 516, "use_type": 1, "req_level": 50,
	})
	if err := handler.gameData.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load mystery formula: %v", err)
	}
}

func loadPlantSkillTemplates(t *testing.T, handler *Handler) {
	t.Helper()

	raw := []json.RawMessage{
		itemSkillRawJSON(t, map[string]interface{}{
			"id":        4965,
			"name":      "Trồng Trọt",
			"type":      appskill.PlantSkillType,
			"kind":      2,
			"use_env":   6,
			"level":     1,
			"req_level": 50,
			"dex_skill": 0,
		}),
	}
	if err := handler.gameData.GetCache().LoadTable(models.TableSkill, raw); err != nil {
		t.Fatalf("load plant skill template: %v", err)
	}
}

func itemSkillRawJSON(t *testing.T, value map[string]interface{}) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	return json.RawMessage(data)
}
