package item

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestUsePreferredTimedOrConsumableItemKeepsTimedItemAndDoesNotExtendExpiry(t *testing.T) {
	repo := newItemServiceTestRepo()
	service := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, _ := json.Marshal(models.ItemTemplateTemplate{ID: 1515, T: 10080})
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	service.SetGameDataManager(manager)
	it := domainitem.NewItem(1, 1515, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 0)
	if err := repo.Create(context.Background(), it); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	used, templateID, consumed, err := service.UsePreferredTimedOrConsumableItem(context.Background(), 1, []int{1515}, now)
	if err != nil || used == nil || templateID != 1515 || consumed {
		t.Fatalf("unexpected activation: item=%+v template=%d consumed=%v err=%v", used, templateID, consumed, err)
	}
	firstExpiry, ok := reusableItemExpiry(used)
	if !ok || !firstExpiry.Equal(now.Add(7*24*time.Hour)) {
		t.Fatalf("expiry=%v active=%v", firstExpiry, ok)
	}
	used, _, consumed, err = service.UsePreferredTimedOrConsumableItem(context.Background(), 1, []int{1515}, now.Add(time.Hour))
	secondExpiry, ok := reusableItemExpiry(used)
	if err != nil || consumed || !ok || !secondExpiry.Equal(firstExpiry) {
		t.Fatalf("reuse extended or consumed item: expiry=%v consumed=%v err=%v", secondExpiry, consumed, err)
	}
	if len(repo.items) != 1 {
		t.Fatalf("timed item count=%d, want 1", len(repo.items))
	}
}

func TestUsePreferredTimedOrConsumableItemConsumesPermanentSingleUseItem(t *testing.T) {
	repo := newItemServiceTestRepo()
	service := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, _ := json.Marshal(models.ItemTemplateTemplate{ID: 1036, T: -1})
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	service.SetGameDataManager(manager)
	it := domainitem.NewItem(1, 1036, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 0)
	if err := repo.Create(context.Background(), it); err != nil {
		t.Fatal(err)
	}
	_, _, consumed, err := service.UsePreferredTimedOrConsumableItem(context.Background(), 1, []int{1036}, time.Now())
	if err != nil || !consumed || len(repo.items) != 0 {
		t.Fatalf("single-use item not consumed: consumed=%v items=%d err=%v", consumed, len(repo.items), err)
	}
}

func TestUsePreferredTimedOrConsumableItemPrefersTimedPassAndKeepsSingleUseItem(t *testing.T) {
	repo := newItemServiceTestRepo()
	service := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	rawSingle, _ := json.Marshal(models.ItemTemplateTemplate{ID: 1036, T: -1})
	rawTimed, _ := json.Marshal(models.ItemTemplateTemplate{ID: 1363, T: 43200})
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{rawSingle, rawTimed}); err != nil {
		t.Fatal(err)
	}
	service.SetGameDataManager(manager)
	single := domainitem.NewItem(1, 1036, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 0)
	timed := domainitem.NewItem(1, 1363, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 1)
	if err := repo.Create(context.Background(), single); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), timed); err != nil {
		t.Fatal(err)
	}
	used, templateID, consumed, err := service.UsePreferredTimedOrConsumableItem(context.Background(), 1, []int{1363, 1036}, time.Now())
	if err != nil || used == nil || templateID != 1363 || consumed || len(repo.items) != 2 {
		t.Fatalf("item=%+v template=%d consumed=%v items=%d err=%v", used, templateID, consumed, len(repo.items), err)
	}
}

func TestUsePreferredTimedOrConsumableItemDoesNotReactivateExpiredPass(t *testing.T) {
	repo := newItemServiceTestRepo()
	service := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, _ := json.Marshal(models.ItemTemplateTemplate{ID: 1515, T: 10080})
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	service.SetGameDataManager(manager)
	it := domainitem.NewItem(1, 1515, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 0)
	expiredAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	it.Properties[reusableItemExpiresAtKey] = expiredAt.Format(time.RFC3339Nano)
	if err := repo.Create(context.Background(), it); err != nil {
		t.Fatal(err)
	}
	used, templateID, consumed, err := service.UsePreferredTimedOrConsumableItem(context.Background(), 1, []int{1515}, expiredAt.Add(time.Second))
	if err != nil || used != nil || templateID != 0 || consumed {
		t.Fatalf("expired item reactivated: item=%+v template=%d consumed=%v err=%v", used, templateID, consumed, err)
	}
	storedExpiry, ok := reusableItemExpiry(repo.items[it.ID])
	if !ok || !storedExpiry.Equal(expiredAt) {
		t.Fatalf("expiry changed: %v active=%v", storedExpiry, ok)
	}
}
