package item

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestShopDeterministicManifestGrantsCatalogContents(t *testing.T) {
	sql, err := os.ReadFile("../../../supabase/data/shop_deterministic_awards.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(sql), "$shop_awards$")
	if len(parts) != 3 {
		t.Fatal("missing manifest")
	}
	var rows [][]json.RawMessage
	if err := json.Unmarshal([]byte(parts[1]), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 13 {
		t.Fatalf("rows=%d want13", len(rows))
	}
	for _, row := range rows {
		var bag, kind, template, count int
		var quality float64
		var name, fragment string
		for index, dest := range []interface{}{&bag, &kind, &template, &count, &quality, &name, &fragment} {
			if err := json.Unmarshal(row[index], dest); err != nil {
				t.Fatal(err)
			}
		}
		t.Run(name, func(t *testing.T) {
			repo := newItemServiceTestRepo()
			repo.nextID = 2000
			svc := NewService(repo, zap.NewNop())
			manager := gamedata.NewManager(nil, zap.NewNop())
			loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: int64(bag), Name: "Box", UseType: 1, Type: 550})
			payload := map[string]interface{}{models.AwardPayloadGuaranteedKey: true}
			if kind == 12 {
				loadCreatureTemplate(t, manager, models.CreatureTemplate{ID: int64(template), Name: name})
			} else {
				tplKind := float64(5)
				if count == 12 {
					tplKind = 6
					payload["materialColorCode"] = float64(3)
				}
				loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: int64(template), Name: name, Type: 600, Kind: tplKind, StackMax: 99999})
			}
			loadAwardTemplate(t, manager, models.ItemAwardTemplate{ID: 1, ItemID: float64(bag), Type: float64(kind), AwardID: float64(template), Count: float64(count), Quality: quality, Rate: 100, Payload: payload})
			svc.SetGameDataManager(manager)
			pets := &awardEffectTestPetService{capacityConfigured: true, hasCapacity: true}
			svc.SetPetService(pets)
			box := &domainitem.Item{ID: 1000, CharacterID: 99, TemplateID: bag, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
			repo.items[box.ID] = box
			if _, _, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 99, Level: 100}, box.ID, nil); err != nil {
				t.Fatal(err)
			}
			if _, exists := repo.items[box.ID]; exists {
				t.Fatal("box not consumed")
			}
			if kind == 12 {
				if len(pets.templateIDs) != 1 || pets.templateIDs[0] != template || pets.awardQuality != 1.5 {
					t.Fatalf("wrong pet reward: ids=%v growth=%v", pets.templateIDs, pets.awardQuality)
				}
			} else {
				if len(repo.items) != 1 {
					t.Fatalf("items=%d want1", len(repo.items))
				}
				for _, item := range repo.items {
					if item.TemplateID != template || item.StackCount != count {
						t.Fatalf("wrong catalog content: %#v", item)
					}
					if count == 12 && item.ColorCode != 3 {
						t.Fatalf("material grade4 color=%d want3", item.ColorCode)
					}
				}
			}
		})
	}
}
