// Open-sourced by BaoLT

package activity

import (
	"context"
	"encoding/json"
	"testing"

	appactivity "mcgame-server/internal/application/activity"
	appitem "mcgame-server/internal/application/item"
	domainactivity "mcgame-server/internal/domain/activity"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestBuySendCombine_SuccessUpdatesBuyCountAndGold(t *testing.T) {
	charRepo := &fakeSendCombineCharacterRepo{
		character: &domainchar.Character{
			ID:            3,
			AccountID:     uuid.New(),
			Gold:          100000,
			PetGuardData:  map[string]interface{}{},
			PMProcessData: map[string]interface{}{},
		},
	}
	itemRepo := newFakeSendCombineItemRepo()
	itemService := newSendCombineItemService(t, itemRepo)
	handler := NewHandler(itemService, zap.NewNop())
	handler.SetCharacterRepository(charRepo)
	commandService := &fakeSendCombineService{gold: 100000, counts: make(map[string]int)}
	handler.SetSendCombineService(commandService)

	ctx := &rtmp.RPCContext{
		Context:     context.Background(),
		ConnID:      20,
		CharacterID: "3",
	}

	result, err := handler.BuySendCombine(ctx, []interface{}{"0", false, false, false})
	if err != nil {
		t.Fatalf("BuySendCombine() error = %v", err)
	}

	buyNum, ok := result.(int)
	if !ok {
		t.Fatalf("BuySendCombine() type = %T, want int", result)
	}
	if buyNum != 1 {
		t.Fatalf("BuySendCombine() = %d, want 1", buyNum)
	}

	expectedGold := int64(100000 - 6999)
	if commandService.gold != expectedGold {
		t.Fatalf("committed gold = %d, want %d", commandService.gold, expectedGold)
	}

	if len(commandService.items) == 0 {
		t.Fatal("expected reward items to be added")
	}

	panelPayload, err := handler.GetSendCombineAct(&rtmp.RPCContext{
		Context:     context.Background(),
		ConnID:      21,
		CharacterID: "3",
	}, nil)
	if err != nil {
		t.Fatalf("GetSendCombineAct() error = %v", err)
	}

	activities := panelPayload.(map[string]interface{})
	activity0 := activities["0"].(map[string]interface{})
	if got := activity0["buyNum"]; got != 1 {
		t.Fatalf("activities[0][buyNum] = %v, want 1", got)
	}
}

func TestBuySendCombine_InsufficientGoldDoesNotConsumeOrGrant(t *testing.T) {
	charRepo := &fakeSendCombineCharacterRepo{
		character: &domainchar.Character{
			ID:            3,
			AccountID:     uuid.New(),
			Gold:          1000,
			PetGuardData:  map[string]interface{}{},
			PMProcessData: map[string]interface{}{},
		},
	}
	itemRepo := newFakeSendCombineItemRepo()
	itemService := newSendCombineItemService(t, itemRepo)
	handler := NewHandler(itemService, zap.NewNop())
	handler.SetCharacterRepository(charRepo)
	commandService := &fakeSendCombineService{gold: 1000, counts: make(map[string]int)}
	handler.SetSendCombineService(commandService)

	result, err := handler.BuySendCombine(&rtmp.RPCContext{
		Context:     context.Background(),
		ConnID:      22,
		CharacterID: "3",
	}, []interface{}{"0", false, false, false})
	if err != nil {
		t.Fatalf("BuySendCombine() error = %v", err)
	}

	buyNum, ok := result.(int)
	if !ok {
		t.Fatalf("BuySendCombine() type = %T, want int", result)
	}
	if buyNum != 0 {
		t.Fatalf("BuySendCombine() = %d, want 0", buyNum)
	}

	if commandService.gold != 1000 {
		t.Fatalf("committed gold = %d, want 1000", commandService.gold)
	}
	if len(commandService.items) != 0 {
		t.Fatalf("len(committed items) = %d, want 0", len(commandService.items))
	}
}

type fakeSendCombineService struct {
	gold   int64
	counts map[string]int
	items  []*domainitem.Item
}

func (s *fakeSendCombineService) Purchase(ctx context.Context, command domainactivity.SendCombinePurchaseCommand) (*domainactivity.SendCombinePurchaseResult, error) {
	if s.gold < command.Cost {
		return nil, pkgerrors.ErrInsufficientFunds
	}
	if s.counts[command.ActivityID] >= command.Limit {
		return nil, appactivity.ErrSendCombineLimitReached
	}
	s.gold -= command.Cost
	s.counts[command.ActivityID]++
	s.items = make([]*domainitem.Item, 0, len(command.Awards))
	for index, award := range command.Awards {
		s.items = append(s.items, &domainitem.Item{ID: int64(index + 1), CharacterID: command.CharacterID, TemplateID: award.TemplateID, ItemType: award.ItemType, SlotType: domainitem.SlotTypeBag, SlotIndex: index, StackCount: award.Count, IsBound: award.IsBound, Properties: map[string]interface{}{}})
	}
	return &domainactivity.SendCombinePurchaseResult{BuyCount: s.counts[command.ActivityID], Gold: s.gold, Items: s.items}, nil
}

func (s *fakeSendCombineService) BuyCounts(ctx context.Context, characterID int64, eventEndMs int64) (map[string]int, error) {
	result := make(map[string]int, len(s.counts))
	for activityID, count := range s.counts {
		result[activityID] = count
	}
	return result, nil
}

type fakeSendCombineCharacterRepo struct {
	character *domainchar.Character
}

func newSendCombineItemService(t *testing.T, repo *fakeSendCombineItemRepo) *appitem.Service {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	rows := make([]json.RawMessage, 0, 3)
	for _, templateID := range []int64{6861, 408, 5351} {
		raw, err := json.Marshal(models.ItemTemplateTemplate{ID: templateID, Kind: 1, StackMax: 99})
		if err != nil {
			t.Fatalf("marshal item template %d: %v", templateID, err)
		}
		rows = append(rows, raw)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, rows); err != nil {
		t.Fatalf("load item templates: %v", err)
	}
	service := appitem.NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)
	return service
}

func (r *fakeSendCombineCharacterRepo) FindByID(ctx context.Context, id int64) (*domainchar.Character, error) {
	if r.character == nil || r.character.ID != id {
		return nil, pkgerrors.ErrCharacterNotFound
	}
	return r.character, nil
}

func (r *fakeSendCombineCharacterRepo) FindByAccountID(ctx context.Context, accountID uuid.UUID) ([]*domainchar.Character, error) {
	return nil, nil
}

func (r *fakeSendCombineCharacterRepo) FindByMapID(ctx context.Context, mapID int) ([]*domainchar.Character, error) {
	return nil, nil
}

func (r *fakeSendCombineCharacterRepo) FindByName(ctx context.Context, name string) (*domainchar.Character, error) {
	return nil, pkgerrors.ErrCharacterNotFound
}

func (r *fakeSendCombineCharacterRepo) Create(ctx context.Context, character *domainchar.Character) error {
	r.character = character
	return nil
}

func (r *fakeSendCombineCharacterRepo) Update(ctx context.Context, character *domainchar.Character) error {
	r.character = character
	return nil
}

func (r *fakeSendCombineCharacterRepo) Delete(ctx context.Context, id int64) error {
	if r.character != nil && r.character.ID == id {
		r.character = nil
	}
	return nil
}

func (r *fakeSendCombineCharacterRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	return false, nil
}

func (r *fakeSendCombineCharacterRepo) UpdatePosition(ctx context.Context, id int64, pos domainchar.Position) error {
	return nil
}

func (r *fakeSendCombineCharacterRepo) UpdateStats(ctx context.Context, id int64, hp, mp, sp int) error {
	return nil
}

type fakeSendCombineItemRepo struct {
	nextID int64
	items  map[int64]*domainitem.Item
}

func newFakeSendCombineItemRepo() *fakeSendCombineItemRepo {
	return &fakeSendCombineItemRepo{
		nextID: 1,
		items:  map[int64]*domainitem.Item{},
	}
}

func (r *fakeSendCombineItemRepo) FindByID(ctx context.Context, id int64) (*domainitem.Item, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, pkgerrors.ErrItemNotFound
	}
	return item, nil
}

func (r *fakeSendCombineItemRepo) FindByCharacterID(ctx context.Context, charID int64) ([]*domainitem.Item, error) {
	result := make([]*domainitem.Item, 0)
	for _, item := range r.items {
		if item.CharacterID == charID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *fakeSendCombineItemRepo) FindByCharacterAndSlotType(ctx context.Context, charID int64, slotType domainitem.SlotType) ([]*domainitem.Item, error) {
	result := make([]*domainitem.Item, 0)
	for _, item := range r.items {
		if item.CharacterID == charID && item.SlotType == slotType {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *fakeSendCombineItemRepo) FindBySlot(ctx context.Context, charID int64, slotType domainitem.SlotType, slotIndex int) (*domainitem.Item, error) {
	for _, item := range r.items {
		if item.CharacterID == charID && item.SlotType == slotType && item.SlotIndex == slotIndex {
			return item, nil
		}
	}
	return nil, pkgerrors.ErrItemNotFound
}

func (r *fakeSendCombineItemRepo) FindEquipped(ctx context.Context, charID int64) ([]*domainitem.Item, error) {
	result := make([]*domainitem.Item, 0)
	for _, item := range r.items {
		if item.CharacterID == charID && item.SlotType == domainitem.SlotTypeEquipped {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *fakeSendCombineItemRepo) Create(ctx context.Context, item *domainitem.Item) error {
	if item.ID == 0 {
		item.ID = r.nextID
		r.nextID++
	}
	r.items[item.ID] = item
	return nil
}

func (r *fakeSendCombineItemRepo) Update(ctx context.Context, item *domainitem.Item) error {
	if _, ok := r.items[item.ID]; !ok {
		return pkgerrors.ErrItemNotFound
	}
	r.items[item.ID] = item
	return nil
}

func (r *fakeSendCombineItemRepo) Delete(ctx context.Context, id int64) error {
	delete(r.items, id)
	return nil
}

func (r *fakeSendCombineItemRepo) DeleteByCharacterID(ctx context.Context, charID int64) error {
	for id, item := range r.items {
		if item.CharacterID == charID {
			delete(r.items, id)
		}
	}
	return nil
}

func (r *fakeSendCombineItemRepo) MoveItem(ctx context.Context, id int64, slotType domainitem.SlotType, slotIndex int) error {
	item, ok := r.items[id]
	if !ok {
		return pkgerrors.ErrItemNotFound
	}
	item.SlotType = slotType
	item.SlotIndex = slotIndex
	return nil
}

func (r *fakeSendCombineItemRepo) UpdateStack(ctx context.Context, id int64, stackCount int) error {
	item, ok := r.items[id]
	if !ok {
		return pkgerrors.ErrItemNotFound
	}
	item.StackCount = stackCount
	return nil
}

func (r *fakeSendCombineItemRepo) FindFirstEmptySlot(ctx context.Context, charID int64, slotType domainitem.SlotType, maxSlots int) (int, error) {
	used := make(map[int]struct{})
	for _, item := range r.items {
		if item.CharacterID == charID && item.SlotType == slotType {
			used[item.SlotIndex] = struct{}{}
		}
	}

	for slot := 0; slot < maxSlots; slot++ {
		if _, exists := used[slot]; !exists {
			return slot, nil
		}
	}

	return 0, pkgerrors.ErrInventoryFull
}

func (r *fakeSendCombineItemRepo) CountBySlotType(ctx context.Context, charID int64, slotType domainitem.SlotType) (int, error) {
	count := 0
	for _, item := range r.items {
		if item.CharacterID == charID && item.SlotType == slotType {
			count++
		}
	}
	return count, nil
}
