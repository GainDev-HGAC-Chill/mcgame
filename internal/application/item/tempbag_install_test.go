package item

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestInstallTempBagPersistsCapacityExpiryAndReplacement(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[1] = &domainitem.Item{ID: 1, CharacterID: 1, TemplateID: 20, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1, Properties: map[string]interface{}{}}
	repo.items[2] = &domainitem.Item{ID: 2, CharacterID: 1, TemplateID: 50, SlotType: domainitem.SlotTypeBag, SlotIndex: 1, StackCount: 1, Properties: map[string]interface{}{}}
	service := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	records := make([]json.RawMessage, 0, 2)
	for _, tpl := range []models.ItemTemplateTemplate{{ID: 20, Type: 517, ProplNum: 20, T: -1}, {ID: 50, Type: 517, ProplNum: 50, T: 43200}} {
		raw, err := json.Marshal(tpl)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, raw)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, records); err != nil {
		t.Fatal(err)
	}
	service.SetGameDataManager(manager)
	now := time.Now().UTC()
	first, err := service.InstallTempBag(context.Background(), 1, 2101, 80, 1, now)
	if err != nil || first.Replaced != nil || first.Installed.SlotIndex != tempBagInstalledIndex(1) {
		t.Fatalf("first install=%+v err=%v", first, err)
	}
	second, err := service.InstallTempBag(context.Background(), 1, 2102, 80, 1, now)
	if err != nil || second.Replaced == nil || second.Replaced.ID != 1 || second.Replaced.SlotIndex != 1 {
		t.Fatalf("replacement=%+v err=%v", second, err)
	}
	expiresAt, ok := tempBagExpiry(second.Installed)
	if !ok || !expiresAt.Equal(now.Add(30*24*time.Hour)) {
		t.Fatalf("expiry=%v ok=%v", expiresAt, ok)
	}
	items, err := service.GetAllItems(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	payload := service.BuildTBagPayload(&domainchar.Character{TempBagSlots: 1}, items)
	if payload["tempSlotNum"] != 50 {
		t.Fatalf("tempSlotNum=%v, want 50", payload["tempSlotNum"])
	}
	installed := payload["tempBag"].(map[string]interface{})["1"].(map[string]interface{})
	if installed["tid"] != 50 || installed["ot"] != false {
		t.Fatalf("installed payload=%v", installed)
	}
	if _, err := service.SortTempBag(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	storedInstalled := repo.items[2]
	if storedInstalled.SlotIndex != tempBagInstalledIndex(1) {
		t.Fatalf("sort moved installed bag to content slot %d", storedInstalled.SlotIndex)
	}
}

func TestInstallTempBagBlocksReplacementThatWouldHideContents(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[1] = &domainitem.Item{ID: 1, CharacterID: 1, TemplateID: 50, SlotType: domainitem.SlotTypeTempBag, SlotIndex: tempBagInstalledIndex(1), StackCount: 1, Properties: map[string]interface{}{}}
	repo.items[2] = &domainitem.Item{ID: 2, CharacterID: 1, TemplateID: 20, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1, Properties: map[string]interface{}{}}
	repo.items[3] = &domainitem.Item{ID: 3, CharacterID: 1, TemplateID: 900, SlotType: domainitem.SlotTypeTempBag, SlotIndex: 30, StackCount: 1, Properties: map[string]interface{}{}}
	service := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	records := make([]json.RawMessage, 0, 2)
	for _, tpl := range []models.ItemTemplateTemplate{{ID: 20, Type: 517, ProplNum: 20, T: -1}, {ID: 50, Type: 517, ProplNum: 50, T: -1}} {
		raw, _ := json.Marshal(tpl)
		records = append(records, raw)
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, records); err != nil {
		t.Fatal(err)
	}
	service.SetGameDataManager(manager)
	if _, err := service.InstallTempBag(context.Background(), 1, 2101, 80, 1, time.Now()); err == nil {
		t.Fatal("replacement reducing capacity below occupied slot succeeded")
	}
	if repo.items[1].SlotType != domainitem.SlotTypeTempBag || repo.items[2].SlotType != domainitem.SlotTypeBag || repo.items[3].SlotIndex != 30 {
		t.Fatalf("items changed after rejected replacement: %+v", repo.items)
	}
}
