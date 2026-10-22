// Open-sourced by BaoLT

package item

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type monsterHeartGrantRecorder struct {
	heartID int
	count   int
}

func (r *monsterHeartGrantRecorder) AddHeart(_ context.Context, _ int64, heartID, count int) (map[string]interface{}, error) {
	r.heartID = heartID
	r.count += count
	return map[string]interface{}{"bag": map[string]interface{}{}}, nil
}

type failingMonsterHeartKeyRepo struct {
	*itemServiceTestRepo
	failUpdate bool
}

func (r *failingMonsterHeartKeyRepo) UpdateStack(ctx context.Context, id int64, stackCount int) error {
	if r.failUpdate {
		return errors.New("forced stack failure")
	}
	return r.itemServiceTestRepo.UpdateStack(ctx, id, stackCount)
}

type monsterHeartCatalogFile[T any] struct {
	Records []T `json:"records"`
}

type monsterHeartKeyCatalogItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	UseType  string `json:"useType"`
	ProplNum string `json:"proplNum"`
}

type monsterHeartCatalogHeart struct {
	ID string `json:"id"`
}

func TestMonsterHeartKeyCatalogFingerprintAndReferences(t *testing.T) {
	var items monsterHeartCatalogFile[monsterHeartKeyCatalogItem]
	readCatalogFile(t, "../../../docs/database/game_data/TBL_ITEM_TEMPLATE.json", &items)
	var hearts monsterHeartCatalogFile[monsterHeartCatalogHeart]
	readCatalogFile(t, "../../../docs/database/game_data/TBL_CREATUREH_HEART.json", &hearts)

	heartIDs := make(map[int]struct{}, len(hearts.Records))
	for _, heart := range hearts.Records {
		id, _ := strconv.Atoi(heart.ID)
		heartIDs[id] = struct{}{}
	}
	matched := 0
	for i := range items.Records {
		item := &items.Records[i]
		id, _ := strconv.Atoi(item.ID)
		typeID, _ := strconv.Atoi(item.Type)
		useType, _ := strconv.Atoi(item.UseType)
		heartID, _ := strconv.Atoi(item.ProplNum)
		tpl := &models.ItemTemplateTemplate{ID: int64(id), Name: item.Name, Type: float64(typeID), UseType: float64(useType), ProplNum: float64(heartID)}
		if !isMonsterHeartKey(tpl) {
			continue
		}
		matched++
		if _, ok := heartIDs[heartID]; !ok {
			t.Fatalf("key %d references missing heart %d", int(tpl.ID), int(tpl.ProplNum))
		}
	}
	if matched != 395 {
		t.Fatalf("monster-heart key count = %d, want 395", matched)
	}
}

func TestUseMultiItemReturnsActualUsedItemAndSpiritualityGrant(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[88] = &domainitem.Item{ID: 88, CharacterID: 7, TemplateID: 2336, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 4, StackCount: 3}
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTestRecord(t, manager, models.TableItemTemplate, models.ItemTemplateTemplate{ID: 2336, Name: "Kết Tinh Thần Tu", Type: 550, UseType: 1, Kind: 5, StackMax: 99999})
	loadItemTestRecord(t, manager, models.TableItemAward, models.ItemAwardTemplate{ID: 1, ItemID: 2336, AwardID: 241, Type: 35, Count: 400000, Rate: 1})
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)
	char := &domainchar.Character{ID: 7, Spirituality: "0"}

	result, err := service.UseMultiItem(context.Background(), char, 88, 2, 1, -1)
	if err != nil {
		t.Fatal(err)
	}
	used, ok := result["used_item"].(*domainitem.Item)
	if !ok || used.ID != 88 || used.StackCount != 1 || used.CalculateSID() != repo.items[88].CalculateSID() {
		t.Fatalf("used_item = %#v", result["used_item"])
	}
	if char.Spirituality != "800000" {
		t.Fatalf("spirituality = %q, want 800000", char.Spirituality)
	}
	grants := result["granted_currencies"].([]GrantedCurrencyResult)
	if len(grants) != 1 || grants[0].ClientKey != "spirituality" || grants[0].Delta != 800000 || grants[0].Total != 800000 {
		t.Fatalf("grants = %#v", grants)
	}

	result, err = service.UseMultiItem(context.Background(), char, 88, 1, 1, -1)
	if err != nil {
		t.Fatal(err)
	}
	used = result["used_item"].(*domainitem.Item)
	if used.StackCount != 0 || result["remaining"] != 0 {
		t.Fatalf("exhausted result = %#v", result)
	}
}

func TestMonsterHeartKeyConsumesBeforeSingleAtomicGrant(t *testing.T) {
	baseRepo := newItemServiceTestRepo()
	baseRepo.items[90] = &domainitem.Item{ID: 90, CharacterID: 7, TemplateID: 5375, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 5, StackCount: 3}
	repo := &failingMonsterHeartKeyRepo{itemServiceTestRepo: baseRepo, failUpdate: true}
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTestRecord(t, manager, models.TableItemTemplate, models.ItemTemplateTemplate{ID: 5375, Name: "Ma Tâm (Hình Xăm Thanh Long)", Type: 550, UseType: 1, Kind: 5, ProplNum: 1, StackMax: 99999})
	loadItemTestRecord(t, manager, models.TableCreaturehHeart, models.CreaturehHeartTemplate{ID: 1, Type: 1})
	recorder := &monsterHeartGrantRecorder{}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(manager)
	service.SetMonsterHeartGranter(recorder)
	char := &domainchar.Character{ID: 7}

	if _, err := service.UseMultiItem(context.Background(), char, 90, 2, 1, -1); err == nil {
		t.Fatal("expected forced inventory update failure")
	}
	if recorder.count != 0 {
		t.Fatalf("heart granted before consumption: %#v", recorder)
	}

	repo.failUpdate = false
	result, err := service.UseMultiItem(context.Background(), char, 90, 2, 1, -1)
	if err != nil {
		t.Fatal(err)
	}
	if recorder.heartID != 1 || recorder.count != 2 {
		t.Fatalf("heart grant = %#v, want heart 1 x2", recorder)
	}
	if _, ok := result["monster_heart_panel"].(map[string]interface{}); !ok {
		t.Fatalf("missing monster-heart panel result: %#v", result)
	}
}

func readCatalogFile(t *testing.T, path string, target interface{}) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatal(err)
	}
}

func loadItemTestRecord(t *testing.T, manager *gamedata.Manager, table string, record interface{}) {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(table, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
}
