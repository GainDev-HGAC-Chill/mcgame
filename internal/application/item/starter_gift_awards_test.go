// Isolated catalog fixtures validate the approved SQL manifest without using real players.
package item

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type starterReward struct {
	Bag, Type, Template, Count int
	Quality                    float64
	Prefix                     int
	Name                       string
}

func starterManifest(t *testing.T) []starterReward {
	t.Helper()
	sql, err := os.ReadFile("../../../supabase/data/starter_gift_awards.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(sql), "$starter_awards$")
	if !ok {
		t.Fatal("missing manifest")
	}
	data, _, ok := strings.Cut(rest, "$starter_awards$")
	if !ok {
		t.Fatal("unterminated manifest")
	}
	var rows [][]json.RawMessage
	if err := json.Unmarshal([]byte(data), &rows); err != nil {
		t.Fatal(err)
	}
	var rewards []starterReward
	for _, row := range rows {
		if len(row) != 7 {
			t.Fatal("invalid manifest row")
		}
		var r starterReward
		for i, dest := range []interface{}{&r.Bag, &r.Type, &r.Template, &r.Count, &r.Quality, &r.Prefix, &r.Name} {
			if err := json.Unmarshal(row[i], dest); err != nil {
				t.Fatal(err)
			}
		}
		rewards = append(rewards, r)
	}
	if len(rewards) != 54 {
		t.Fatalf("manifest rows=%d want54", len(rewards))
	}
	return rewards
}

func starterFixture(t *testing.T, bag, level int) (*Service, *itemServiceTestRepo, *domainitem.Item, *awardEffectTestPetService, []starterReward) {
	t.Helper()
	log := zap.NewNop()
	repo := newItemServiceTestRepo()
	repo.nextID = 6000
	svc := NewService(repo, log)
	manager := gamedata.NewManager(nil, log)
	var selected []starterReward
	for index, r := range starterManifest(t) {
		if r.Bag != bag {
			continue
		}
		selected = append(selected, r)
		switch r.Type {
		case 19:
			loadEquipmentTemplate(t, manager, models.EquiptTemplateTemplate{ID: int64(r.Template), Name: r.Name, Kind: 4, Type: 400, ReqLevel: 1})
		case 12:
			loadCreatureTemplate(t, manager, models.CreatureTemplate{ID: int64(r.Template), Name: r.Name})
		default:
			kind := float64(5)
			if r.Template == 16 {
				kind = 6
			}
			loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: int64(r.Template), Name: r.Name, Type: 501, Kind: kind, StackMax: 99999, UseType: 1})
		}
		payload := map[string]interface{}{models.AwardPayloadGuaranteedKey: true}
		if r.Template == 16 && r.Type == 29 {
			payload["materialColorCode"] = float64(3)
		}
		loadAwardTemplate(t, manager, models.ItemAwardTemplate{ID: int64(810000001 + index), ItemID: float64(bag), AwardID: float64(r.Template), Type: float64(r.Type), Count: float64(r.Count), Rate: 100, Quality: r.Quality, PreNameType: float64(r.Prefix), Payload: payload})
	}
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: int64(bag), Name: "Starter bag", Type: 550, UseType: 1, ReqLevel: float64(level)})
	svc.SetGameDataManager(manager)
	petSvc := &awardEffectTestPetService{capacityConfigured: true, hasCapacity: true}
	svc.SetPetService(petSvc)
	box := &domainitem.Item{ID: 5000, CharacterID: 99, TemplateID: bag, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
	repo.items[box.ID] = box
	return svc, repo, box, petSvc, selected
}

func TestStarterGiftManifestGrantsEveryApprovedReward(t *testing.T) {
	for _, spec := range []struct{ bag, level, rows int }{{2918, 1, 9}, {2919, 10, 9}, {2920, 20, 9}, {2921, 30, 11}, {2922, 40, 8}, {2923, 50, 8}} {
		t.Run(fmt.Sprint(spec.level), func(t *testing.T) {
			svc, repo, box, petSvc, rewards := starterFixture(t, spec.bag, spec.level)
			if len(rewards) != spec.rows {
				t.Fatalf("reward rows=%d want%d", len(rewards), spec.rows)
			}
			_, _, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 99, Level: spec.level}, box.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := repo.items[box.ID]; ok {
				t.Fatal("box not consumed")
			}
			for _, r := range rewards {
				if r.Type == 12 {
					if len(petSvc.templateIDs) != 1 || petSvc.templateIDs[0] != 91 || petSvc.awardQuality != 1.9 {
						t.Fatalf("pet reward=%v quality=%v", petSvc.templateIDs, petSvc.awardQuality)
					}
					continue
				}
				count := 0
				for _, it := range repo.items {
					if it.TemplateID != r.Template {
						continue
					}
					count += it.StackCount
					if r.Type == 19 && (it.ColorCode != 4 || it.Properties["preNameType"] != 5 || it.Properties["quality"] != 20) {
						t.Fatalf("jewelry quality incorrect: %#v", it)
					}
					if r.Template == 16 && it.ColorCode != 3 {
						t.Fatalf("diamond color=%d want3", it.ColorCode)
					}
				}
				if count != r.Count {
					t.Fatalf("reward%d count=%d want%d", r.Template, count, r.Count)
				}
			}
			itemCount, petCalls := len(repo.items), petSvc.contractCalls
			if _, _, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 99, Level: spec.level}, box.ID, nil); err == nil {
				t.Fatal("reused a consumed bag")
			}
			if len(repo.items) != itemCount || petSvc.contractCalls != petCalls {
				t.Fatal("consumed bag replay granted more rewards")
			}
		})
	}
}

func TestStarterGiftLevelGatePreservesBoxAndAwards(t *testing.T) {
	for _, spec := range []struct{ bag, level int }{{2918, 1}, {2919, 10}, {2920, 20}, {2921, 30}, {2922, 40}, {2923, 50}} {
		t.Run(fmt.Sprint(spec.level), func(t *testing.T) {
			svc, repo, box, petSvc, _ := starterFixture(t, spec.bag, spec.level)
			_, _, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 99, Level: spec.level - 1}, box.ID, nil)
			if !pkgerrors.Is(err, pkgerrors.ErrInsufficientLevel) {
				t.Fatalf("error=%v", err)
			}
			if len(repo.items) != 1 || box.StackCount != 1 || petSvc.contractCalls != 0 {
				t.Fatal("blocked use changed rewards/inventory")
			}
		})
	}
}

func TestStarterGiftFullBagPreservesAllRewards(t *testing.T) {
	for _, spec := range []struct{ bag, level int }{{2918, 1}, {2919, 10}, {2920, 20}, {2921, 30}, {2922, 40}, {2923, 50}} {
		t.Run(fmt.Sprint(spec.level), func(t *testing.T) {
			svc, repo, box, petSvc, _ := starterFixture(t, spec.bag, spec.level)
			for slot := 1; slot < 210; slot++ {
				id := int64(100000 + slot)
				repo.items[id] = &domainitem.Item{ID: id, CharacterID: 99, TemplateID: 100000 + slot, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: slot, StackCount: 1}
			}
			_, _, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 99, Level: spec.level}, box.ID, nil)
			if !pkgerrors.Is(err, pkgerrors.ErrInventoryFull) {
				t.Fatalf("error=%v", err)
			}
			if len(repo.items) != 210 || box.StackCount != 1 || petSvc.contractCalls != 0 {
				t.Fatal("full bag changed rewards/inventory")
			}
		})
	}
}
