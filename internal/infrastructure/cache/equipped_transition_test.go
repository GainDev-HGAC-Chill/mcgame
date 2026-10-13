package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"go.uber.org/zap"

	appitem "mcgame-server/internal/application/item"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type pausedEquipRepository struct {
	*CachedItemRepository
	read   chan struct{}
	resume chan struct{}
}

func (r pausedEquipRepository) FindByID(ctx context.Context, id int64) (*item.Item, error) {
	current, err := r.CachedItemRepository.FindByID(ctx, id)
	close(r.read)
	<-r.resume
	return current, err
}

func equipTransitionFixture(t *testing.T, reqLevel int) (*PlayerCache, *PlayerData, *CachedItemRepository, *gamedata.Manager) {
	t.Helper()
	cache, data := transitionCache()
	data.AddItemWithoutDirty(&item.Item{ID: 1, CharacterID: 7, TemplateID: 101, ItemType: item.ItemTypeEquipment, SlotType: item.SlotTypeBag, SlotIndex: 0})
	data.AddItemWithoutDirty(&item.Item{ID: 2, CharacterID: 7, TemplateID: 102, ItemType: item.ItemTypeEquipment, SlotType: item.SlotTypeEquipped, SlotIndex: 0})
	manager := gamedata.NewManager(nil, zap.NewNop())
	var raws []json.RawMessage
	for _, template := range []models.EquiptTemplateTemplate{{ID: 101, Position: 1, ReqLevel: float64(reqLevel)}, {ID: 102, Position: 1, ReqLevel: 1}} {
		raw, err := json.Marshal(template)
		if err != nil {
			t.Fatal(err)
		}
		raws = append(raws, raw)
	}
	if err := manager.GetCache().LoadTable(models.TableEquiptTemplate, raws); err != nil {
		t.Fatal(err)
	}
	return cache, data, NewCachedItemRepository(cache, nil), manager
}

func TestEquipPausedBeforeAtomicCommitRejectsAfterDowngrade(t *testing.T) {
	cache, data, repo, manager := equipTransitionFixture(t, 10)
	before := data.GetItems()
	paused := pausedEquipRepository{CachedItemRepository: repo, read: make(chan struct{}), resume: make(chan struct{})}
	service := appitem.NewService(paused, zap.NewNop())
	service.SetGameDataManager(manager)
	finished := make(chan error, 1)
	go func() { _, err := service.EquipItem(context.Background(), 7, 1, -1); finished <- err }()
	<-paused.read
	_, changed, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, inventory []*item.Item) (bool, error) {
		for _, it := range inventory {
			if it.SlotType == item.SlotTypeEquipped && manager.GetEquipment(it.TemplateID).ReqLevel > 5 {
				return false, pkgerrors.ErrInsufficientLevel
			}
		}
		return char.SetTargetLevel(5)
	})
	if err != nil || !changed {
		close(paused.resume)
		t.Fatalf("downgrade failed: %v", err)
	}
	revision := data.revision
	close(paused.resume)
	if err := <-finished; !pkgerrors.Is(err, pkgerrors.ErrInsufficientLevel) {
		t.Fatalf("stale equip accepted: %v", err)
	}
	if data.GetCharacter().Level != 5 || data.GetItem(1).SlotType != item.SlotTypeBag || data.GetItem(2).SlotType != item.SlotTypeEquipped || data.revision != revision || len(data.dirtyItems) != 0 {
		t.Fatal("rejected equip moved incoming/displaced item or dirtied inventory")
	}
	for _, old := range before {
		if !reflect.DeepEqual(old, data.GetItem(old.ID)) {
			t.Fatal("rejected equip changed item")
		}
	}
}

func TestAtomicEquipDisplacesAndDowngradeSeesNewItem(t *testing.T) {
	cache, data, repo, manager := equipTransitionFixture(t, 10)
	service := appitem.NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)
	revision := data.revision
	displaced, err := service.EquipItem(context.Background(), 7, 1, -1)
	if err != nil || displaced == nil || displaced.ID != 2 || displaced.SlotType != item.SlotTypeBag || data.GetItem(1).SlotType != item.SlotTypeEquipped || data.GetItem(2).SlotType != item.SlotTypeBag || data.revision != revision+1 || len(data.dirtyItems) != 2 {
		t.Fatalf("atomic equip failed: %v", err)
	}
	_, changed, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, inventory []*item.Item) (bool, error) {
		for _, it := range inventory {
			if it.SlotType == item.SlotTypeEquipped && manager.GetEquipment(it.TemplateID).ReqLevel > 5 {
				return false, pkgerrors.ErrInsufficientLevel
			}
		}
		return char.SetTargetLevel(5)
	})
	if err == nil || changed || data.GetCharacter().Level != 10 {
		t.Fatal("downgrade ignored newly equipped item")
	}
}

func TestAtomicEquipFullBagIsAllOrNothing(t *testing.T) {
	_, data, repo, manager := equipTransitionFixture(t, 1)
	for i := 1; i < data.GetCharacter().MaxBagSlots(); i++ {
		data.AddItemWithoutDirty(&item.Item{ID: int64(10 + i), CharacterID: 7, SlotType: item.SlotTypeBag, SlotIndex: i})
	}
	revision := data.revision
	service := appitem.NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)
	if _, err := service.EquipItem(context.Background(), 7, 1, -1); !pkgerrors.Is(err, pkgerrors.ErrInventoryFull) {
		t.Fatalf("full bag accepted: %v", err)
	}
	if data.GetItem(1).SlotType != item.SlotTypeBag || data.GetItem(2).SlotType != item.SlotTypeEquipped || data.revision != revision || data.IsDirty() {
		t.Fatal("failed equip committed partial move")
	}
}

func TestCachedGenericWritesCannotBypassEquipTransition(t *testing.T) {
	_, data, repo, _ := equipTransitionFixture(t, 10)
	if err := repo.MoveItem(context.Background(), 1, item.SlotTypeEquipped, 0); err == nil {
		t.Fatal("generic MoveItem equipped without eligibility")
	}
	stale := data.GetItem(1)
	stale.SlotType = item.SlotTypeEquipped
	if err := repo.Update(context.Background(), stale); err == nil {
		t.Fatal("stale Update equipped without eligibility")
	}
	if data.GetItem(1).SlotType != item.SlotTypeBag || data.GetItem(2).SlotType != item.SlotTypeEquipped {
		t.Fatal("generic equip changed current inventory")
	}
}

func TestAtomicResourcePatchUsesCurrentLevelAndPreservesWealth(t *testing.T) {
	cache, data, repo, manager := equipTransitionFixture(t, 1)
	service := appitem.NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)
	_, _, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, _ []*item.Item) (bool, error) {
		char.Gold = 1234
		_, err := char.SetTargetLevel(5)
		char.CurrentHP = 99999
		char.CurrentMP = 99999
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	before := data.GetCharacter()
	updated, changed, err := service.SyncEquipmentResources(context.Background(), 7)
	if err != nil || !changed || updated == nil {
		t.Fatalf("resource patch failed: %v", err)
	}
	want := *before
	want.CurrentHP = updated.CurrentHP
	want.CurrentMP = updated.CurrentMP
	if !reflect.DeepEqual(&want, updated) || updated.Level != 5 || updated.Gold != 1234 || updated.CurrentHP >= 99999 || updated.CurrentMP >= 99999 {
		t.Fatal("resource patch replaced non-resource state")
	}
}

func TestCachedAutoEquipReturnsCommittedSlot(t *testing.T) {
	_, data, repo, manager := equipTransitionFixture(t, 1)
	service := appitem.NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)
	equipped, result, err := service.UseItem(context.Background(), data.GetCharacter(), 1, nil)
	if err != nil || equipped == nil || equipped.SlotType != item.SlotTypeEquipped || equipped.CalculateSID() != 1 || result["equipped"] != true {
		t.Fatalf("cached autoequip returned stale item: item=%+v result=%v err=%v", equipped, result, err)
	}
	if !reflect.DeepEqual(equipped, data.GetItem(1)) || data.GetItem(2).SlotType != item.SlotTypeBag {
		t.Fatal("autoequip response differs from committed inventory")
	}
}
