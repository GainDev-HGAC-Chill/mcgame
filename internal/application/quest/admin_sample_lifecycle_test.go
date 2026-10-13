// Real quest/item/pet application-service integration with isolated in-memory repositories.
// Serialized representative admin-edited catalog rows model restart hydration only;
// this does not execute the admin API, PostgreSQL, RTMP handlers, or Flash client.
// NPC validation is an explicit handler-boundary call, not a CompleteQuest guarantee.
package quest

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	appitem "mcgame-server/internal/application/item"
	apppet "mcgame-server/internal/application/pet"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type adminSampleCharacters struct {
	questTestCharacterRepo
	characters map[int64]*domainchar.Character
}

func (r *adminSampleCharacters) FindByID(_ context.Context, id int64) (*domainchar.Character, error) {
	c := r.characters[id]
	if c == nil {
		return nil, pkgerrors.ErrNotFound
	}
	copy := *c
	return &copy, nil
}

func (r *adminSampleCharacters) Update(_ context.Context, c *domainchar.Character) error {
	copy := *c
	r.characters[c.ID] = &copy
	return nil
}

type adminSamplePets struct {
	*questTestPetRepo
	nextID int64
}

func (r *adminSamplePets) Save(ctx context.Context, p *domainpet.Pet) error {
	if p.ID == 0 {
		p.ID = r.nextID
		r.nextID++
	}
	return r.questTestPetRepo.Save(ctx, p)
}

func adminSampleCatalog(t *testing.T) map[string][]json.RawMessage {
	t.Helper()
	rows := map[string][]interface{}{
		models.TableQuest: {
			models.QuestTemplate{ID: 88000, Name: "Admin edited sample", Type: 2, At: 1, Gender: 2, MinLevel: 10, MaxLevel: 90, StartNPC: 231, FinishNPC: 232, AwardExpRe: 20, MoneyType: 2, MoneyNum: 35},
			models.QuestTemplate{ID: 88001, Name: "Unrelated quest", Type: 2, Gender: 2},
		},
		models.TableQuestRequire: {
			models.QuestRequireTemplate{ID: 1, Qid: 88000, Kind: 2, Type: 12, ItemID: 9001, Num: 2},
			models.QuestRequireTemplate{ID: 2, Qid: 88000, Kind: 1, Type: 29, ItemID: 9002, Num: 2, Q: 5},
			models.QuestRequireTemplate{ID: 3, Qid: 88000, Kind: 3, Type: 12, ItemID: 9003, Num: 1, Q: 12},
			models.QuestRequireTemplate{ID: 4, Qid: 88001, Kind: 2, Type: 12, ItemID: 9004, Num: 1},
		},
		models.TableQuestAward: {
			models.QuestAwardTemplate{ID: 1, Qid: 88000, Kind: 1, B: 1, Type: 29, ItemID: 9005, Num: 3, Q: 0},
			models.QuestAwardTemplate{ID: 2, Qid: 88000, Kind: 1, B: 1, Type: 19, ItemID: 9006, Num: 1, Q: 10},
			models.QuestAwardTemplate{ID: 3, Qid: 88000, Kind: 2, B: 1, Type: 12, ItemID: 9007, Num: 1, Q: 15},
		},
		models.TableCreature: {
			models.CreatureTemplate{ID: 9001, Name: "Monster"},
			models.CreatureTemplate{ID: 9003, Name: "Submit pet", ClassID: 2, GrowBase: 1.2},
			models.CreatureTemplate{ID: 9004, Name: "Other monster"},
			models.CreatureTemplate{ID: 9007, Name: "Reward pet", ClassID: 2, GrowBase: 0.8},
		},
		models.TableItemTemplate: {
			models.ItemTemplateTemplate{ID: 9002, Name: "Collect item", StackMax: 99},
			models.ItemTemplateTemplate{ID: 9005, Name: "Reward item", StackMax: 99},
		},
		models.TableEquiptTemplate: {models.EquiptTemplateTemplate{ID: 9006, Name: "Reward equipment"}},
		models.TableNpc: {
			models.NpcTemplate{ID: 231, Name: "Start NPC", PosMapID: 1, PosX: 1000, PosY: 1000},
			models.NpcTemplate{ID: 232, Name: "Finish NPC", PosMapID: 2, PosX: 100, PosY: 100},
		},
	}
	serialized := make(map[string][]json.RawMessage, len(rows))
	for table, records := range rows {
		for _, record := range records {
			serialized[table] = append(serialized[table], mustQuestJSON(t, record))
		}
	}
	return serialized
}

func adminSampleManager(t *testing.T, rows map[string][]json.RawMessage) *gamedata.Manager {
	t.Helper()
	m := gamedata.NewManager(nil, zap.NewNop())
	for table, records := range rows {
		if err := m.GetCache().LoadTable(table, records); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

func TestAdminSampleLifecycleInMemoryCatalogReload(t *testing.T) {
	ctx := context.Background()
	rows := adminSampleCatalog(t)
	first := adminSampleManager(t, rows)
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	var reloadedRows map[string][]json.RawMessage
	if err := json.Unmarshal(encoded, &reloadedRows); err != nil {
		t.Fatal(err)
	}
	reloaded := adminSampleManager(t, reloadedRows)
	firstRequire, reloadRequire := first.GetQuestRequire(88000), reloaded.GetQuestRequire(88000)
	firstAwards, reloadAwards := first.GetQuestAwards(88000), reloaded.GetQuestAwards(88000)
	sort.Slice(firstRequire, func(i, j int) bool { return firstRequire[i].ID < firstRequire[j].ID })
	sort.Slice(reloadRequire, func(i, j int) bool { return reloadRequire[i].ID < reloadRequire[j].ID })
	sort.Slice(firstAwards, func(i, j int) bool { return firstAwards[i].ID < firstAwards[j].ID })
	sort.Slice(reloadAwards, func(i, j int) bool { return reloadAwards[i].ID < reloadAwards[j].ID })
	if first == reloaded || !reflect.DeepEqual(first.GetQuest(88000), reloaded.GetQuest(88000)) ||
		!reflect.DeepEqual(firstRequire, reloadRequire) || !reflect.DeepEqual(firstAwards, reloadAwards) {
		t.Fatal("fresh catalog hydration changed acceptance/objective/reward rows")
	}
	for name, manager := range map[string]*gamedata.Manager{"edited rows": first, "fresh manager reload": reloaded} {
		t.Run(name, func(t *testing.T) {
			quests := newQuestTestQuestRepo()
			chars := &adminSampleCharacters{characters: map[int64]*domainchar.Character{
				2: {ID: 2, Level: 10, MapID: 2, PosX: 100, PosY: 100, Money: 100},
				3: {ID: 3, Level: 10, MapID: 2, PosX: 100, PosY: 100, Money: 500},
			}}
			items := newQuestTestItemRepo()
			pets := &adminSamplePets{questTestPetRepo: newQuestTestPetRepo(), nextID: 100}
			svc := NewService(quests, zap.NewNop())
			svc.SetGameDataManager(manager)
			svc.SetCharacterRepository(chars)
			itemSvc := appitem.NewService(items, zap.NewNop())
			itemSvc.SetGameDataManager(manager)
			itemSvc.SetQuestService(svc)
			svc.SetItemService(itemSvc)
			petSvc := apppet.NewService(pets, zap.NewNop())
			petSvc.SetGameDataManager(manager)
			svc.SetPetService(petSvc)
			for _, pair := range [][2]int{{2, 88001}, {3, 88000}} {
				if _, err := svc.AcceptQuest(ctx, int64(pair[0]), pair[1], nil); err != nil {
					t.Fatal(err)
				}
			}
			otherQuest, _ := quests.FindByCharacterAndQuest(ctx, 2, 88001)
			otherProgress, _ := quests.FindByCharacterAndQuest(ctx, 3, 88000)
			otherCharacter, _ := chars.FindByID(ctx, 3)
			chars.characters[2].Level = 9
			if _, err := svc.AcceptQuest(ctx, 2, 88000, nil); !errors.Is(err, pkgerrors.ErrInvalidInput) {
				t.Fatalf("edited minLevel ignored: %v", err)
			}
			chars.characters[2].Level = 10
			qp, err := svc.AcceptQuest(ctx, 2, 88000, []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9999, Required: 1, Current: 1}})
			if err != nil {
				t.Fatal(err)
			}
			want := []domainquest.Objective{
				{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 2},
				{Type: domainquest.ObjectiveCollectItem, Target: 9002, Required: 2, ItemTableType: 29, Quality: 5},
				{Type: domainquest.ObjectiveSubmitPet, Target: 9003, Required: 1, Quality: 12},
			}
			if !reflect.DeepEqual(qp.Objectives, want) {
				t.Fatalf("non-authoritative acceptance: %+v", qp.Objectives)
			}
			if _, rewards, err := svc.CompleteQuest(ctx, 2, 88000); !errors.Is(err, pkgerrors.ErrInvalidInput) || rewards != nil || len(quests.histories) != 0 {
				t.Fatalf("early completion granted rewards: %v %v", rewards, err)
			}
			if changed, err := svc.OnMonsterKilled(ctx, 2, 9999); err != nil || len(changed) != 0 {
				t.Fatalf("invalid kill target credited: %v %v", changed, err)
			}
			for i := 0; i < 2; i++ {
				if _, err := svc.OnMonsterKilled(ctx, 2, 9001); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := svc.OnItemCollected(ctx, 2, 9002, 99); err != nil {
				t.Fatal(err)
			}
			if _, err := itemSvc.AddItemWithBindAndColor(ctx, 2, 9002, domainitem.ItemTypeConsumable, 2, false, 0); err != nil {
				t.Fatal(err)
			}
			qp, err = svc.GetActiveQuest(ctx, 2, 88000)
			if err != nil || qp.Objectives[1].Current != 0 || qp.CanComplete() {
				t.Fatalf("forged collection/wrong quality counted: %+v %v", qp, err)
			}
			if _, err := itemSvc.AddItemWithBindAndColor(ctx, 2, 9002, domainitem.ItemTypeConsumable, 2, false, 1); err != nil {
				t.Fatal(err)
			}
			if err := pets.Save(ctx, &domainpet.Pet{ID: 10, CharacterID: 2, TemplateID: 9003, Name: "Submit pet", GrowRate: 1.2}); err != nil {
				t.Fatal(err)
			}
			qp, err = svc.GetActiveQuest(ctx, 2, 88000)
			if err != nil || !qp.CanComplete() || qp.Objectives[0].Current != 2 || qp.Objectives[1].Current != 2 || qp.Objectives[2].Current != 1 {
				t.Fatalf("authoritative inventory/pet progress: %+v %v", qp, err)
			}
			for _, location := range []struct {
				name        string
				mapID, x, y int
			}{
				{"wrongNPC", 1, 1000, 1000}, {"wrong map", 1, 100, 100}, {"too far", 2, 401, 100},
			} {
				t.Run(location.name, func(t *testing.T) {
					c := chars.characters[2]
					c.MapID, c.PosX, c.PosY = location.mapID, location.x, location.y
					if err := svc.ValidateQuestTurnIn(ctx, 2, 88000); !errors.Is(err, pkgerrors.ErrInvalidInput) {
						t.Fatalf("invalid turn-in location allowed: %v", err)
					}
					if len(quests.histories) != 0 || c.Money != 100 || c.Experience != 0 || len(items.items) != 2 || len(pets.pets) != 1 {
						t.Fatal("rejected turn-in changed rewards or inventory")
					}
				})
			}
			chars.characters[2].MapID, chars.characters[2].PosX, chars.characters[2].PosY = 2, 400, 100
			if err := svc.ValidateQuestTurnIn(ctx, 2, 88000); err != nil {
				t.Fatal(err)
			}
			completed, rewards, err := svc.CompleteQuest(ctx, 2, 88000)
			if err != nil || completed.Status != domainquest.QuestStatusCompleted || len(quests.histories) != 1 {
				t.Fatalf("completion failed: %+v %v", completed, err)
			}
			c := chars.characters[2]
			if c.Experience != 20 || c.Money != 135 || c.QuestN != 1 || rewards["exp"] != int64(20) || rewards["currencyChanges"].(map[string]int64)["money"] != 35 {
				t.Fatalf("catalog EXP/money not granted: %+v %v", c, rewards)
			}
			if len(rewards["items"].([]map[string]interface{})) != 2 || len(rewards["pets"].([]map[string]interface{})) != 1 || !reflect.DeepEqual(rewards["consumedPets"], []int64{10}) {
				t.Fatalf("missing item/pet/consumption payload: %v", rewards)
			}
			for _, it := range items.items {
				switch it.TemplateID {
				case 9002:
					if it.ColorCode != 0 || it.StackCount != 2 {
						t.Fatalf("wrong-quality item consumed: %+v", it)
					}
				case 9005:
					if !it.IsBound || it.StackCount != 3 || it.ItemType != domainitem.ItemTypeConsumable {
						t.Fatalf("item reward binding/count: %+v", it)
					}
				case 9006:
					if !it.IsBound || it.ColorCode != 3 || it.ItemType != domainitem.ItemTypeEquipment {
						t.Fatalf("equipment reward quality/binding: %+v", it)
					}
				default:
					t.Fatalf("unexpected inventory: %+v", it)
				}
			}
			if len(items.items) != 3 || len(pets.pets) != 1 {
				t.Fatal("unexpected post-completion inventory")
			}
			for _, p := range pets.pets {
				if p.TemplateID != 9007 || p.GrowRate != 1.5 || !p.IsBound() {
					t.Fatalf("quest pet quality/binding: %+v", p)
				}
			}
			beforeRetry := mustQuestJSON(t, []interface{}{chars.characters, items.items, pets.pets, quests.progress, quests.histories})
			if _, rewards, err := svc.CompleteQuest(ctx, 2, 88000); !errors.Is(err, pkgerrors.ErrInvalidInput) || rewards != nil {
				t.Fatalf("retry rewarded: %v %v", rewards, err)
			}
			if _, err := svc.AcceptQuest(ctx, 2, 88000, nil); !errors.Is(err, pkgerrors.ErrInvalidInput) {
				t.Fatalf("completed quest accepted: %v", err)
			}
			if !reflect.DeepEqual(beforeRetry, mustQuestJSON(t, []interface{}{chars.characters, items.items, pets.pets, quests.progress, quests.histories})) {
				t.Fatal("retry changed persisted state")
			}
			gotOther, _ := quests.FindByCharacterAndQuest(ctx, 2, 88001)
			gotProgress, _ := quests.FindByCharacterAndQuest(ctx, 3, 88000)
			gotChar, _ := chars.FindByID(ctx, 3)
			if !reflect.DeepEqual(otherQuest, gotOther) || !reflect.DeepEqual(otherProgress, gotProgress) || !reflect.DeepEqual(otherCharacter, gotChar) {
				t.Fatal("another quest/character changed")
			}
		})
	}
}

func TestAdminSampleInvalidTargetFailsClosed(t *testing.T) {
	rows := adminSampleCatalog(t)
	rows[models.TableQuestRequire][0] = mustQuestJSON(t, models.QuestRequireTemplate{ID: 1, Qid: 88000, Kind: 2, Type: 12, ItemID: 9999, Num: 2})
	svc := NewService(newQuestTestQuestRepo(), zap.NewNop())
	svc.SetGameDataManager(adminSampleManager(t, rows))
	svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, Level: 10}})
	svc.SetItemService(appitem.NewService(newQuestTestItemRepo(), zap.NewNop()))
	svc.SetPetService(apppet.NewService(newQuestTestPetRepo(), zap.NewNop()))
	qp, err := svc.AcceptQuest(context.Background(), 2, 88000, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !qp.CompletionBlocked || svc.catalogTargetsAvailable(qp) {
		t.Fatal("missing catalog target did not fail closed")
	}
	authorityBlocked(t, svc, qp)
}
