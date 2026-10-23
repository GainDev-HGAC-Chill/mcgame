// Open-sourced by BaoLT

package shop

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	appitem "mcgame-server/internal/application/item"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type purchaseSyncItemRepo struct {
	nextID int64
	items  map[int64]*domainitem.Item
}

func newPurchaseSyncItemRepo() *purchaseSyncItemRepo {
	return &purchaseSyncItemRepo{nextID: 1, items: make(map[int64]*domainitem.Item)}
}

func (r *purchaseSyncItemRepo) FindByID(ctx context.Context, id int64) (*domainitem.Item, error) {
	it, ok := r.items[id]
	if !ok {
		return nil, pkgerrors.ErrItemNotFound
	}
	copyItem := *it
	return &copyItem, nil
}

func (r *purchaseSyncItemRepo) FindByCharacterID(ctx context.Context, charID int64) ([]*domainitem.Item, error) {
	items := make([]*domainitem.Item, 0)
	for _, it := range r.items {
		if it.CharacterID != charID {
			continue
		}
		copyItem := *it
		items = append(items, &copyItem)
	}
	return items, nil
}

func (r *purchaseSyncItemRepo) FindByCharacterAndSlotType(ctx context.Context, charID int64, slotType domainitem.SlotType) ([]*domainitem.Item, error) {
	items := make([]*domainitem.Item, 0)
	for _, it := range r.items {
		if it.CharacterID != charID || it.SlotType != slotType {
			continue
		}
		copyItem := *it
		items = append(items, &copyItem)
	}
	return items, nil
}

func (r *purchaseSyncItemRepo) FindBySlot(ctx context.Context, charID int64, slotType domainitem.SlotType, slotIndex int) (*domainitem.Item, error) {
	for _, it := range r.items {
		if it.CharacterID == charID && it.SlotType == slotType && it.SlotIndex == slotIndex {
			copyItem := *it
			return &copyItem, nil
		}
	}
	return nil, pkgerrors.ErrItemNotFound
}

func (r *purchaseSyncItemRepo) FindEquipped(ctx context.Context, charID int64) ([]*domainitem.Item, error) {
	return r.FindByCharacterAndSlotType(ctx, charID, domainitem.SlotTypeEquipped)
}

func (r *purchaseSyncItemRepo) Create(ctx context.Context, it *domainitem.Item) error {
	copyItem := *it
	copyItem.ID = r.nextID
	r.nextID++
	r.items[copyItem.ID] = &copyItem
	it.ID = copyItem.ID
	return nil
}

func (r *purchaseSyncItemRepo) Update(ctx context.Context, it *domainitem.Item) error {
	if _, ok := r.items[it.ID]; !ok {
		return pkgerrors.ErrItemNotFound
	}
	copyItem := *it
	r.items[it.ID] = &copyItem
	return nil
}

func (r *purchaseSyncItemRepo) Delete(ctx context.Context, id int64) error {
	delete(r.items, id)
	return nil
}

func (r *purchaseSyncItemRepo) DeleteByCharacterID(ctx context.Context, charID int64) error {
	for id, it := range r.items {
		if it.CharacterID == charID {
			delete(r.items, id)
		}
	}
	return nil
}

func (r *purchaseSyncItemRepo) MoveItem(ctx context.Context, id int64, slotType domainitem.SlotType, slotIndex int) error {
	it, ok := r.items[id]
	if !ok {
		return pkgerrors.ErrItemNotFound
	}
	it.SlotType = slotType
	it.SlotIndex = slotIndex
	return nil
}

func (r *purchaseSyncItemRepo) UpdateStack(ctx context.Context, id int64, stackCount int) error {
	it, ok := r.items[id]
	if !ok {
		return pkgerrors.ErrItemNotFound
	}
	it.StackCount = stackCount
	return nil
}

func (r *purchaseSyncItemRepo) FindFirstEmptySlot(ctx context.Context, charID int64, slotType domainitem.SlotType, maxSlots int) (int, error) {
	occupied := make(map[int]bool)
	for _, it := range r.items {
		if it.CharacterID == charID && it.SlotType == slotType {
			occupied[it.SlotIndex] = true
		}
	}
	for i := 0; i < maxSlots; i++ {
		if !occupied[i] {
			return i, nil
		}
	}
	return -1, pkgerrors.ErrInventoryFull
}

func (r *purchaseSyncItemRepo) CountBySlotType(ctx context.Context, charID int64, slotType domainitem.SlotType) (int, error) {
	count := 0
	for _, it := range r.items {
		if it.CharacterID == charID && it.SlotType == slotType {
			count++
		}
	}
	return count, nil
}

func TestBuyItemReturnsTouchedBagSlotsForStackAndRemainder(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	itemRepo := newPurchaseSyncItemRepo()
	itemService := appitem.NewService(itemRepo, logger)
	manager := newPurchaseSyncTestManager(t, models.ItemTemplateTemplate{
		ID:         158,
		Name:       "Potion",
		Price:      1,
		StackMax:   99999,
		SingleFlag: 0,
		Kind:       0,
	})
	itemService.SetGameDataManager(manager)

	service := NewService()
	service.SetGameDataManager(manager)
	service.SetItemService(itemService)
	service.SetCharacterRepository(newVIPShopCharacterRepo(&character.Character{
		ID:         1,
		Name:       "buyer",
		MoneyBind:  200000,
		BagSlotNum: 1,
	}))

	existing := domainitem.NewItem(1, 158, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 0)
	existing.StackCount = 15
	existing.IsBound = true
	if err := itemRepo.Create(ctx, existing); err != nil {
		t.Fatalf("seed existing item: %v", err)
	}

	result, err := service.BuyItem(ctx, 1, 158, 99999)
	if err != nil {
		t.Fatalf("BuyItem returned error: %v", err)
	}

	if len(result.AddedItemDTOs) != 2 {
		t.Fatalf("AddedItemDTOs len = %d, want 2", len(result.AddedItemDTOs))
	}

	if got := dtoInt(t, result.AddedItemDTOs[0], "itemId"); got != int(existing.ID) {
		t.Fatalf("first dto itemId = %d, want %d", got, existing.ID)
	}
	if got := dtoInt(t, result.AddedItemDTOs[0], "stackNum"); got != 99999 {
		t.Fatalf("first dto stackNum = %d, want 99999", got)
	}
	if got := dtoString(t, result.AddedItemDTOs[0], "binded"); got != "1" {
		t.Fatalf("first dto binded = %q, want 1", got)
	}

	if got := dtoInt(t, result.AddedItemDTOs[1], "stackNum"); got != 15 {
		t.Fatalf("second dto stackNum = %d, want 15", got)
	}
	if got := dtoString(t, result.AddedItemDTOs[1], "binded"); got != "1" {
		t.Fatalf("second dto binded = %q, want 1", got)
	}

	items, err := itemService.GetAllItems(ctx, 1)
	if err != nil {
		t.Fatalf("GetAllItems returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("item count = %d, want 2", len(items))
	}
}

func TestBuyItemKeepsBoundStacksSeparateInPurchaseUpdates(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	itemRepo := newPurchaseSyncItemRepo()
	itemService := appitem.NewService(itemRepo, logger)
	manager := newPurchaseSyncTestManager(t, models.ItemTemplateTemplate{
		ID:         158,
		Name:       "Potion",
		Price:      1,
		StackMax:   99999,
		SingleFlag: 0,
		Kind:       0,
	})
	itemService.SetGameDataManager(manager)

	service := NewService()
	service.SetGameDataManager(manager)
	service.SetItemService(itemService)
	service.SetCharacterRepository(newVIPShopCharacterRepo(&character.Character{
		ID:         1,
		Name:       "buyer",
		MoneyBind:  1000,
		BagSlotNum: 1,
	}))

	existing := domainitem.NewItem(1, 158, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 0)
	existing.StackCount = 15
	existing.IsBound = false
	if err := itemRepo.Create(ctx, existing); err != nil {
		t.Fatalf("seed existing item: %v", err)
	}

	result, err := service.BuyItem(ctx, 1, 158, 10)
	if err != nil {
		t.Fatalf("BuyItem returned error: %v", err)
	}

	if len(result.AddedItemDTOs) != 1 {
		t.Fatalf("AddedItemDTOs len = %d, want 1", len(result.AddedItemDTOs))
	}
	if got := dtoInt(t, result.AddedItemDTOs[0], "itemId"); got == int(existing.ID) {
		t.Fatalf("expected new bound stack, got existing item id %d", got)
	}
	if got := dtoInt(t, result.AddedItemDTOs[0], "stackNum"); got != 10 {
		t.Fatalf("new dto stackNum = %d, want 10", got)
	}
	if got := dtoString(t, result.AddedItemDTOs[0], "binded"); got != "1" {
		t.Fatalf("new dto binded = %q, want 1", got)
	}

	storedExisting, err := itemRepo.FindByID(ctx, existing.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if storedExisting.StackCount != 15 {
		t.Fatalf("existing stackCount = %d, want 15", storedExisting.StackCount)
	}
	if storedExisting.IsBound {
		t.Fatalf("existing stack unexpectedly became bound")
	}
}

func TestBuyItemQualityZeroKeepsGatheringGloveWhite(t *testing.T) {
	ctx := context.Background()
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadPurchaseSyncRecord(t, manager, models.TableEquiptTemplate, models.EquiptTemplateTemplate{
		ID: 1123, Name: "Găng Thu Thập", Kind: 1, StackMax: 1, SingleFlag: 1, ColorCode: 5,
	})
	loadPurchaseSyncRecord(t, manager, models.TableShopSlot, models.ShopSlotTemplate{
		ID: 3439, Sid: 32, ItemID: 1123, Type: 19, Money: 1, Quality: 0, Amount: -1,
	})
	itemRepo := newPurchaseSyncItemRepo()
	itemService := appitem.NewService(itemRepo, zap.NewNop())
	itemService.SetGameDataManager(manager)
	charRepo := newVIPShopCharacterRepo(&character.Character{ID: 1, Name: "buyer", MoneyBind: 10, BagSlotNum: 30})
	service := NewService()
	service.SetGameDataManager(manager)
	service.SetItemService(itemService)
	service.SetCharacterRepository(charRepo)

	result, err := service.BuyItem(ctx, 1, 3439, 1)
	if err != nil {
		t.Fatal(err)
	}
	items, err := itemService.GetAllItems(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].ColorCode != 0 || items[0].Properties["preNameType"] != 0 {
		t.Fatalf("purchased glove = %+v", items[0])
	}
	if got := result.ItemData["n"]; got != "Găng Thu Thập" {
		t.Fatalf("popup name = %v", got)
	}
	if len(result.AddedItemDTOs) != 1 || dtoInt(t, result.AddedItemDTOs[0], "colorCode") != 0 || dtoInt(t, result.AddedItemDTOs[0], "preNameType") != 0 {
		t.Fatalf("added DTOs = %#v", result.AddedItemDTOs)
	}
}

func TestShopQualityValidationRejectsOutOfRangeBeforeDebit(t *testing.T) {
	ctx := context.Background()
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadPurchaseSyncRecord(t, manager, models.TableEquiptTemplate, models.EquiptTemplateTemplate{ID: 1123, Name: "Găng", Kind: 1})
	loadPurchaseSyncRecord(t, manager, models.TableShopSlot, models.ShopSlotTemplate{ID: 3439, ItemID: 1123, Type: 19, Money: 1, Quality: 26})
	itemService := appitem.NewService(newPurchaseSyncItemRepo(), zap.NewNop())
	itemService.SetGameDataManager(manager)
	charRepo := newVIPShopCharacterRepo(&character.Character{ID: 1, MoneyBind: 10, BagSlotNum: 30})
	service := NewService()
	service.SetGameDataManager(manager)
	service.SetItemService(itemService)
	service.SetCharacterRepository(charRepo)

	if _, err := service.BuyItem(ctx, 1, 3439, 1); !errors.Is(err, errInvalidShopQuality) {
		t.Fatalf("error = %v, want invalid quality", err)
	}
	char, _ := charRepo.FindByID(ctx, 1)
	if char.MoneyBind != 10 {
		t.Fatalf("money after rejected quality = %d, want 10", char.MoneyBind)
	}
}

func TestBuySystemItemUsesExchangePointsWithoutTouchingHonor(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	itemRepo := newPurchaseSyncItemRepo()
	itemService := appitem.NewService(itemRepo, logger)
	manager := newPurchaseSyncTestManager(t, models.ItemTemplateTemplate{
		ID: 158, Name: "Point item", StackMax: 99, Kind: 5,
	})
	loadPurchaseSyncRecord(t, manager, models.TableShopSlot, models.ShopSlotTemplate{
		ID: 700, ItemID: 158, Type: 29, Point: 5,
	})
	itemService.SetGameDataManager(manager)
	char := &character.Character{ID: 1, Name: "buyer", ExPoint: 5, Honor: 99, BagSlotNum: 30}
	charRepo := newVIPShopCharacterRepo(char)
	service := NewService()
	service.SetGameDataManager(manager)
	service.SetItemService(itemService)
	service.SetCharacterRepository(charRepo)

	result, err := service.BuySystemItem(ctx, 1, 700, 1)
	if err != nil {
		t.Fatal(err)
	}
	storedChar, err := charRepo.FindByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.NewMoney != 0 || result.CurrencyType != 0 || storedChar.ExPoint != 0 || storedChar.Honor != 99 {
		t.Fatalf("unexpected balances/result: result=%+v exPoint=%d honor=%d", result, storedChar.ExPoint, storedChar.Honor)
	}
	if _, err := service.BuySystemItem(ctx, 1, 700, 1); err == nil {
		t.Fatal("second purchase without exchange points succeeded")
	}
	items, err := itemService.GetAllItems(ctx, 1)
	if err != nil || len(items) != 1 || items[0].StackCount != 1 {
		t.Fatalf("failed purchase changed inventory: items=%+v err=%v", items, err)
	}
}

func TestBuySystemItemSerializesConcurrentPointPurchases(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	itemRepo := newPurchaseSyncItemRepo()
	itemService := appitem.NewService(itemRepo, logger)
	manager := newPurchaseSyncTestManager(t, models.ItemTemplateTemplate{ID: 158, Name: "Point item", StackMax: 99, Kind: 5})
	loadPurchaseSyncRecord(t, manager, models.TableShopSlot, models.ShopSlotTemplate{ID: 700, ItemID: 158, Type: 29, Point: 5})
	itemService.SetGameDataManager(manager)
	charRepo := newVIPShopCharacterRepo(&character.Character{ID: 1, Name: "buyer", ExPoint: 5, BagSlotNum: 30})
	service := NewService()
	service.SetGameDataManager(manager)
	service.SetItemService(itemService)
	service.SetCharacterRepository(charRepo)

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.BuySystemItem(ctx, 1, 700, 1)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		}
	}
	items, err := itemService.GetAllItems(ctx, 1)
	if err != nil || succeeded != 1 || len(items) != 1 || items[0].StackCount != 1 {
		t.Fatalf("success=%d items=%+v err=%v", succeeded, items, err)
	}
}

func TestExchangeItemUsesCrystalCreditCatalogAndReturnsCompleteDTOs(t *testing.T) {
	ctx := context.Background()
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadPurchaseSyncRecord(t, manager, models.TableEquiptTemplate, models.EquiptTemplateTemplate{
		ID: 1255, Name: "Pet Helm", Kind: 8, StackMax: 1, SingleFlag: 1, ColorCode: 2,
	})
	loadPurchaseSyncRecord(t, manager, models.TableCredit, models.CreditTemplate{
		ID: 1, ShopID: 1, ItemID: 1255, Type: 19, CType1: 32, CNum1: 1, LimitType: 0, LimitNum: 0,
	})
	itemRepo := newPurchaseSyncItemRepo()
	itemService := appitem.NewService(itemRepo, zap.NewNop())
	itemService.SetGameDataManager(manager)
	charRepo := newVIPShopCharacterRepo(&character.Character{ID: 1, Name: "buyer", WisdomCrystal: 3, BagSlotNum: 30})
	service := NewService()
	service.SetGameDataManager(manager)
	service.SetItemService(itemService)
	service.SetCharacterRepository(charRepo)

	result, err := service.ExchangeItem(ctx, 1, "1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.NewPoints != 1 || result.Cost != 2 || result.CurrencyKey != "wisdonCrystal" {
		t.Fatalf("currency result = %#v", result)
	}
	if result.ItemData["n"] == nil || result.ItemData["n"] == "" || result.ItemData["t"] != 19 || result.ItemData["i"] != 1255 {
		t.Fatalf("popup DTO = %#v", result.ItemData)
	}
	if len(result.AddedItemDTOs) != 2 {
		t.Fatalf("added DTO count = %d, want 2", len(result.AddedItemDTOs))
	}
	updated, _ := charRepo.FindByID(ctx, 1)
	if updated.WisdomCrystal != 1 {
		t.Fatalf("persisted wisdom crystal = %d, want 1", updated.WisdomCrystal)
	}
}

func newPurchaseSyncTestManager(t *testing.T, itemTpl models.ItemTemplateTemplate) *gamedata.Manager {
	t.Helper()

	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, err := json.Marshal(itemTpl)
	if err != nil {
		t.Fatalf("marshal item template: %v", err)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatalf("load item template: %v", err)
	}

	return manager
}

func loadPurchaseSyncRecord(t *testing.T, manager *gamedata.Manager, table string, record interface{}) {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(table, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
}

func dtoInt(t *testing.T, dto map[string]interface{}, key string) int {
	t.Helper()

	switch value := dto[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		t.Fatalf("dto[%q] has unsupported type %T", key, dto[key])
	}

	return 0
}

func dtoString(t *testing.T, dto map[string]interface{}, key string) string {
	t.Helper()

	value, ok := dto[key].(string)
	if !ok {
		t.Fatalf("dto[%q] = %#v, want string", key, dto[key])
	}
	return value
}
