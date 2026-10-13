package quest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	appitem "mcgame-server/internal/application/item"
	apppet "mcgame-server/internal/application/pet"
	domainchar "mcgame-server/internal/domain/character"
	domainpet "mcgame-server/internal/domain/pet"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func authoritySave(t *testing.T, svc *Service, qp *domainquest.QuestProgress) {
	t.Helper()
	if err := svc.questRepo.Save(context.Background(), qp); err != nil {
		t.Fatal(err)
	}
}

func authorityBlocked(t *testing.T, svc *Service, qp *domainquest.QuestProgress) {
	t.Helper()
	if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 103 || qp.CanComplete() {
		t.Fatalf("blocked quest reported complete: dto=%v progress=%+v", dto, qp)
	}
	authoritySave(t, svc, qp)
	if _, rewards, err := svc.CompleteQuest(context.Background(), qp.CharacterID, qp.QuestID); err == nil || rewards != nil {
		t.Fatalf("blocked quest granted rewards: rewards=%v err=%v", rewards, err)
	}
	if repo := svc.questRepo.(*questTestQuestRepo); len(repo.histories) != 0 {
		t.Fatal("blocked quest wrote completion history")
	}
}

func TestRequirementAuthorityAcceptIgnoresClientObjectives(t *testing.T) {
	for _, objectives := range [][]domainquest.Objective{
		nil,
		{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 1, Current: 1}},
		{{Type: domainquest.ObjectiveTalkToNPC, Target: 1, Required: 1, Current: 1}},
	} {
		svc := specialQuestService(t, 8000, "Catalog kill", 1)
		loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{Qid: 8000, ItemID: 9001, Kind: 2, Type: 12, Num: 2})
		svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, Level: 90}})
		qp, err := svc.AcceptQuest(context.Background(), 2, 8000, objectives)
		if err != nil {
			t.Fatal(err)
		}
		want := []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 2}}
		if !reflect.DeepEqual(qp.Objectives, want) || qp.CanComplete() {
			t.Fatalf("accepted client objectives: %+v", qp)
		}
	}
}

func TestRequirementAuthorityCatalogRebuildPreservesOnlyExactProgress(t *testing.T) {
	for _, tc := range []struct {
		name    string
		stored  []domainquest.Objective
		current int
	}{
		{"empty", nil, 0},
		{"truncated", []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 2, Current: 1}}, 1},
		{"forged target", []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9999, Required: 2, Current: 2}}, 0},
		{"forged count", []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 1, Current: 1}}, 0},
		{"forged type", []domainquest.Objective{{Type: domainquest.ObjectiveTalkToNPC, Target: 9001, Required: 2, Current: 2}}, 0},
		{"clamped", []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 2, Current: 99}}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := specialQuestService(t, 8000, "Catalog kill", 1)
			rows := []models.QuestRequireTemplate{
				{ID: 1, Qid: 8000, ItemID: 9001, Kind: 2, Type: 12, Num: 2},
				{ID: 2, Qid: 8000, ItemID: 9002, Kind: 2, Type: 12, Num: 1},
			}
			for _, row := range rows {
				loadTestQuestRequirement(t, svc.gameDataManager, row)
			}
			qp := domainquest.NewQuestProgress(2, 8000, tc.stored)
			if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 103 {
				t.Fatalf("incomplete catalog reported complete: %v", dto)
			}
			if len(qp.Objectives) != 2 {
				t.Fatalf("catalog objectives missing: %+v", qp.Objectives)
			}
			for _, obj := range qp.Objectives {
				want := 0
				if obj.Target == 9001 {
					want = tc.current
				}
				if obj.Current != want {
					t.Fatalf("incorrect retained progress: %+v", obj)
				}
			}
		})
	}
}

func TestRequirementAuthorityMissingCatalogTargets(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		questID, target, kind, table int
	}{
		{"missing creature 666", 3983, 666, 2, 12},
		{"missing creature 838", 5005, 838, 2, 12},
		{"missing item 3831", 50170, 3831, 1, 29},
		{"missing item 3832", 50171, 3832, 1, 29},
		{"missing equipment", 8000, 9001, 1, 19},
		{"missing pet creature", 8000, 9001, 3, 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := specialQuestService(t, tc.questID, "Missing catalog target", 1)
			row := models.QuestRequireTemplate{ID: 1, Qid: float64(tc.questID), ItemID: float64(tc.target), Kind: float64(tc.kind), Type: float64(tc.table), Num: 1}
			if err := svc.gameDataManager.GetCache().LoadTable(models.TableQuestRequire, []json.RawMessage{mustQuestJSON(t, row)}); err != nil {
				t.Fatal(err)
			}
			svc.SetItemService(appitem.NewService(newQuestTestItemRepo(), zap.NewNop()))
			svc.SetPetService(apppet.NewService(newQuestTestPetRepo(), zap.NewNop()))
			qp := domainquest.NewQuestProgress(2, tc.questID, svc.GetInitialObjectives(tc.questID))
			qp.Objectives[0].Current = 1
			if svc.catalogTargetsAvailable(qp) {
				t.Fatal("nonexistent catalog target reported available")
			}
			authorityBlocked(t, svc, qp)
			if !qp.CompletionBlocked {
				t.Fatal("missing catalog target did not block verification")
			}
		})
	}
}

func TestRequirementAuthorityEmptyKillNeedsRealEvents(t *testing.T) {
	svc := specialQuestService(t, 8000, "Catalog kill", 1)
	loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{Qid: 8000, ItemID: 9001, Kind: 2, Type: 12, Num: 2})
	qp := domainquest.NewQuestProgress(2, 8000, nil)
	authorityBlocked(t, svc, qp)
	for _, target := range []int{9999, 9001, 9001} {
		if _, err := svc.OnMonsterKilled(context.Background(), 2, target); err != nil {
			t.Fatal(err)
		}
		active, err := svc.GetActiveQuest(context.Background(), 2, 8000)
		if err != nil {
			t.Fatal(err)
		}
		if target == 9999 && active.Objectives[0].Current != 0 {
			t.Fatal("unrelated kill counted")
		}
	}
	if _, _, err := svc.CompleteQuest(context.Background(), 2, 8000); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.CompleteQuest(context.Background(), 2, 8000); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("duplicate completion err=%v", err)
	}
}

func TestRequirementAuthorityUnknownForgedObjectivesFailClosed(t *testing.T) {
	for _, kind := range []int{1, 2, 3, 6, 7, 9, 10, 20} {
		t.Run(fmt.Sprintf("type_%d", kind), func(t *testing.T) {
			svc := specialQuestService(t, 8000, "Unreviewed", kind)
			qp := domainquest.NewQuestProgress(2, 8000, []domainquest.Objective{{Type: domainquest.ObjectiveKillMonster, Target: 9001, Required: 1, Current: 1}})
			authorityBlocked(t, svc, qp)
		})
	}
}

func TestRequirementAuthoritySpecialDependenciesFailClosed(t *testing.T) {
	for _, tc := range []struct {
		id   int
		name string
		kind int
	}{
		{2590, "Cấp Độ Lên Đến 10", 10},
		{2629, "Cấp độ pet tăng đến 10", 10},
		{2663, "Tăng số lượng pet lên 2", 10},
		{2695, "Kết giao 1 người bạn", 10},
		{1, "Reviewed conversation", 1},
	} {
		t.Run(fmt.Sprint(tc.id), func(t *testing.T) {
			svc := specialQuestService(t, tc.id, tc.name, tc.kind)
			if tc.id == 1 {
				if err := svc.gameDataManager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, models.QuestTemplate{ID: 1, Type: 1, FinishNPC: 231})}); err != nil {
					t.Fatal(err)
				}
			}
			authorityBlocked(t, svc, domainquest.NewQuestProgress(2, tc.id, nil))
		})
	}
	svc := specialQuestService(t, 2590, "Cấp Độ Lên Đến 10", 10)
	svc.SetCharacterRepository(&questTestCharacterRepo{})
	authorityBlocked(t, svc, domainquest.NewQuestProgress(2, 2590, nil))
}

func TestRequirementAuthorityExactDefinitionMatching(t *testing.T) {
	collect := domainquest.Objective{Type: domainquest.ObjectiveCollectItem, Target: 9001, Required: 2, ItemTableType: 29, Quality: 5}
	pet := domainquest.Objective{Type: domainquest.ObjectiveSubmitPet, Target: 9001, Required: 2, Quality: 12}
	for _, tc := range []struct {
		name      string
		canonical domainquest.Objective
		change    func(*domainquest.Objective)
		matches   bool
	}{
		{"collect progress", collect, func(o *domainquest.Objective) { o.Current = 1 }, true},
		{"collect table", collect, func(o *domainquest.Objective) { o.ItemTableType = 19 }, false},
		{"collect quality", collect, func(o *domainquest.Objective) { o.Quality = -1 }, false},
		{"pet progress", pet, func(o *domainquest.Objective) { o.Current = 1 }, true},
		{"pet quality", pet, func(o *domainquest.Objective) { o.Quality = 1 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stored := tc.canonical
			tc.change(&stored)
			if got := sameObjectiveDefinition(tc.canonical, stored); got != tc.matches {
				t.Fatalf("definition match=%v want=%v", got, tc.matches)
			}
		})
	}
}

func TestRequirementAuthorityMissingDependenciesAndCurrency(t *testing.T) {
	for _, tc := range []struct {
		name           string
		kind, table    int
		rn             float64
		missingManager bool
	}{
		{"item service", 1, 29, 0, false},
		{"pet service", 3, 12, 0, false},
		{"manager", 2, 12, 0, true},
		{"positive currency", 2, 12, 1, false},
		{"negative currency", 2, 12, -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := specialQuestService(t, 8000, "Catalog", 1)
			if err := svc.gameDataManager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, models.QuestTemplate{ID: 8000, Type: 1, Rn: tc.rn})}); err != nil {
				t.Fatal(err)
			}
			loadTestQuestRequirement(t, svc.gameDataManager, models.QuestRequireTemplate{Qid: 8000, ItemID: 9001, Kind: float64(tc.kind), Type: float64(tc.table), Num: 1})
			qp := domainquest.NewQuestProgress(2, 8000, svc.GetInitialObjectives(8000))
			qp.Objectives[0].Current = 1
			if tc.missingManager {
				svc.SetGameDataManager(nil)
			}
			authorityBlocked(t, svc, qp)
		})
	}
}

func TestRequirementAuthorityLevelAndPetThresholds(t *testing.T) {
	for _, tc := range []struct {
		name         string
		id           int
		title        string
		level, count int
		complete     bool
	}{
		{"character below", 2590, "Cấp Độ Lên Đến 10", 9, 0, false},
		{"character exact", 2618, "Cấp Độ Lên Đến 10", 10, 0, true},
		{"pet below", 2629, "Cấp độ pet tăng đến 10", 9, 1, false},
		{"pet exact", 2657, "Cấp độ pet tăng đến 10", 10, 1, true},
		{"pet count below", 2663, "Tăng số lượng pet lên 2", 10, 1, false},
		{"pet count exact", 2685, "Tăng số lượng pet lên 2", 10, 2, true},
		{"wrong name", 2590, "Level 10", 90, 0, false},
		{"outside reviewed range", 2589, "Cấp Độ Lên Đến 10", 90, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := specialQuestService(t, tc.id, tc.title, 10)
			svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, Level: tc.level}})
			pets := []*domainpet.Pet{{ID: 99, CharacterID: 3, Level: 100}}
			for i := 0; i < tc.count; i++ {
				pets = append(pets, &domainpet.Pet{ID: int64(i + 1), CharacterID: 2, Level: tc.level})
			}
			svc.SetPetService(apppet.NewService(newQuestTestPetRepo(pets...), zap.NewNop()))
			qp := domainquest.NewQuestProgress(2, tc.id, nil)
			if !tc.complete {
				authorityBlocked(t, svc, qp)
				return
			}
			if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 104 || !qp.CanComplete() {
				t.Fatalf("threshold not met: %v", dto)
			}
			authoritySave(t, svc, qp)
			if _, _, err := svc.CompleteQuest(context.Background(), 2, tc.id); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequirementAuthorityConversationLocation(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mapID, x, y int
		complete    bool
	}{
		{"same location", 1, 100, 100, true},
		{"radius boundary", 1, 400, 100, true},
		{"beyond radius", 1, 401, 100, false},
		{"diagonal remote", 1, 400, 400, false},
		{"wrong map", 2, 100, 100, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := specialQuestService(t, 1, "Reviewed conversation", 1)
			if err := svc.gameDataManager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, models.QuestTemplate{ID: 1, Type: 1, FinishNPC: 231})}); err != nil {
				t.Fatal(err)
			}
			if err := svc.gameDataManager.GetCache().LoadTable(models.TableNpc, []json.RawMessage{mustQuestJSON(t, models.NpcTemplate{ID: 231, PosMapID: 1, PosX: 100, PosY: 100})}); err != nil {
				t.Fatal(err)
			}
			svc.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, MapID: tc.mapID, PosX: tc.x, PosY: tc.y}})
			qp := domainquest.NewQuestProgress(2, 1, nil)
			if !tc.complete {
				authorityBlocked(t, svc, qp)
				return
			}
			if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 104 {
				t.Fatalf("nearby conversation blocked: %v", dto)
			}
			authoritySave(t, svc, qp)
			if _, _, err := svc.CompleteQuest(context.Background(), 2, 1); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequirementAuthorityExpiredAndConcurrentCompletion(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		svc := specialQuestService(t, 8000, "Catalog kill", 1)
		qp := domainquest.NewQuestProgress(2, 8000, loadTestCompletedKill(t, svc.gameDataManager, 8000))
		expired := time.Now().Add(-time.Minute)
		qp.ExpiresAt = &expired
		authoritySave(t, svc, qp)
		if _, _, err := svc.CompleteQuest(context.Background(), 2, 8000); !errors.Is(err, pkgerrors.ErrInvalidInput) {
			t.Fatalf("expired completion err=%v", err)
		}
		if len(svc.questRepo.(*questTestQuestRepo).histories) != 0 {
			t.Fatal("expired quest wrote history")
		}
	})
	t.Run("concurrent", func(t *testing.T) {
		svc := specialQuestService(t, 8000, "Catalog kill", 1)
		authoritySave(t, svc, domainquest.NewQuestProgress(2, 8000, loadTestCompletedKill(t, svc.gameDataManager, 8000)))
		start := make(chan struct{})
		results := make(chan error, 8)
		var wg sync.WaitGroup
		for i := 0; i < cap(results); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, _, err := svc.CompleteQuest(context.Background(), 2, 8000)
				results <- err
			}()
		}
		close(start)
		wg.Wait()
		close(results)
		successes := 0
		for err := range results {
			if err == nil {
				successes++
			} else if !errors.Is(err, pkgerrors.ErrInvalidInput) {
				t.Fatalf("unexpected completion error: %v", err)
			}
		}
		if successes != 1 || len(svc.questRepo.(*questTestQuestRepo).histories) != 1 {
			t.Fatalf("duplicate successful completion: %d", successes)
		}
	})
}
