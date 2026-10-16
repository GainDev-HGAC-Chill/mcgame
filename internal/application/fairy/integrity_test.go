// Open-sourced by BaoLT

package fairy

import (
	"context"
	"errors"
	"sync"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

func TestOnSkillValidatesBookAndGrowthLevel(t *testing.T) {
	tests := []struct {
		name      string
		bookType  float64
		propType  float64
		reqLevel  float64
		growthExp int64
	}{
		{name: "wrong type", bookType: 505, propType: 5917},
		{name: "wrong prop type", bookType: 521, propType: 5920},
		{name: "under growth level", bookType: 521, propType: 5917, reqLevel: 11, growthExp: FairyGrowExp[9]},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, repo, _, items, _ := newTestService(&domainchar.Character{ID: 1})
			f := newFairy(7)
			f.Gexp = test.growthExp
			seedFairy(t, repo, 1, f)
			items.items[600] = &domainitem.Item{ID: 600, TemplateID: 3577, StackCount: 1}
			sd := newFakeSkillData(chainSkills("fairy_skill_a", 5917, 5918, 5919))
			sd.items[3577] = &models.ItemTemplateTemplate{ID: 3577, Type: test.bookType, PropType: test.propType, ReqLevel: test.reqLevel}
			svc.SetSkillData(sd)

			if _, err := svc.OnSkill(context.Background(), 1, 7, 600); err == nil {
				t.Fatal("OnSkill accepted invalid book request")
			}
			if len(items.consumed) != 0 {
				t.Fatalf("invalid book was consumed: %+v", items.consumed)
			}
		})
	}
}

func TestOnSkillRejectsDuplicateFamilyAfterUpgrade(t *testing.T) {
	svc, repo, _, items, _ := newTestService(&domainchar.Character{ID: 1})
	f := newFairy(7)
	f.SkillFlag["s1"] = 5918
	seedFairy(t, repo, 1, f)
	items.items[600] = &domainitem.Item{ID: 600, TemplateID: 3577, StackCount: 1}
	configureFairyBook(svc, 3577, chainSkills("fairy_skill_a", 5917, 5918, 5919), 0)

	if _, err := svc.OnSkill(context.Background(), 1, 7, 600); !errors.Is(err, ErrSkillDuplicate) {
		t.Fatalf("want ErrSkillDuplicate, got %v", err)
	}
	if len(items.consumed) != 0 {
		t.Fatalf("duplicate-family book was consumed: %+v", items.consumed)
	}
}

func TestUpSkillUsesCanonicalChainDespiteCodeNameCollision(t *testing.T) {
	svc, repo, _, items, _ := newTestService(&domainchar.Character{ID: 1})
	f := newFairy(7)
	f.SkillFlag["s1"] = 6266
	seedFairy(t, repo, 1, f)
	rows := append([]*models.SkillTemplate{{ID: 6407, CodeName: "skill_elf_13", Level: 2}}, chainSkills("skill_elf_13", 6266, 6267, 6268)...)
	svc.SetSkillData(newFakeSkillData(rows))
	items.byTemplate[ItemFairySkillUpItem] = &domainitem.Item{TemplateID: ItemFairySkillUpItem, StackCount: 2}

	res, err := svc.UpSkill(context.Background(), 1, 7, 1)
	if err != nil {
		t.Fatalf("UpSkill: %v", err)
	}
	if res.Skill != 6267 {
		t.Fatalf("upgraded skill = %d, want canonical 6267", res.Skill)
	}
}

func TestSkillBookCompensationOnSaveFailure(t *testing.T) {
	svc, repo, _, items, _ := newTestService(&domainchar.Character{ID: 1})
	f := newFairy(7)
	seedFairy(t, repo, 1, f)
	items.items[600] = &domainitem.Item{ID: 600, TemplateID: 3577, ItemType: domainitem.ItemTypeConsumable, StackCount: 1, IsBound: false}
	configureFairyBook(svc, 3577, chainSkills("fairy_skill_a", 5917, 5918, 5919), 0)
	repo.upsertErr = errors.New("save failed")

	if _, err := svc.OnSkill(context.Background(), 1, 7, 600); err == nil {
		t.Fatal("OnSkill succeeded despite save failure")
	}
	if restored := items.byTemplate[3577]; restored == nil || restored.StackCount != 1 || restored.IsBound {
		t.Fatalf("learn book was not restored faithfully: %+v", restored)
	}
}

func TestUpgradeBookCompensationOnSaveFailure(t *testing.T) {
	svc, repo, _, items, _ := newTestService(&domainchar.Character{ID: 1})
	f := newFairy(7)
	f.SkillFlag["s1"] = 5918
	seedFairy(t, repo, 1, f)
	svc.SetSkillData(newFakeSkillData(chainSkills("fairy_skill_a", 5917, 5918, 5919)))
	items.byTemplate[ItemFairySkillUpItem] = &domainitem.Item{TemplateID: ItemFairySkillUpItem, StackCount: 3}
	repo.upsertErr = errors.New("save failed")

	if _, err := svc.UpSkill(context.Background(), 1, 7, 1); err == nil {
		t.Fatal("UpSkill succeeded despite save failure")
	}
	if restored := items.byTemplate[ItemFairySkillUpItem]; restored == nil || restored.StackCount != 3 || !restored.IsBound {
		t.Fatalf("upgrade books were not safely restored: %+v", restored)
	}
}

func TestSetSkillConfigValidatesShapeOwnershipActivityAndFamilies(t *testing.T) {
	svc, repo, _, _, _ := newTestService(&domainchar.Character{ID: 1})
	f := newFairy(7)
	f.SkillFlag["s1"] = 6227
	f.SkillFlag["s2"] = 6230
	f.SkillFlag["s3"] = 6228
	f.SkillFlag["s4"] = 5917
	seedFairy(t, repo, 1, f)
	svc.SetSkillData(newFakeSkillData([]*models.SkillTemplate{
		{ID: 6227, Level: 1, Kind: 1},
		{ID: 6230, Level: 1, Kind: 1},
		{ID: 6228, Level: 2, Kind: 1},
		{ID: 5917, Level: 1, Kind: 0},
	}))
	valid := validSkillConfig(6227, 6230)
	if err := svc.SetSkillConfig(context.Background(), 1, 7, valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(map[string]interface{})
	}{
		{name: "bad select", mutate: func(c map[string]interface{}) { c["select"] = 3 }},
		{name: "missing group", mutate: func(c map[string]interface{}) { delete(c, "2") }},
		{name: "missing slot", mutate: func(c map[string]interface{}) { delete(c["0"].(map[string]interface{}), "c4") }},
		{name: "unlearned", mutate: func(c map[string]interface{}) { c["0"].(map[string]interface{})["c0"] = 6231 }},
		{name: "passive", mutate: func(c map[string]interface{}) { c["0"].(map[string]interface{})["c0"] = 5917 }},
		{name: "duplicate family", mutate: func(c map[string]interface{}) { c["0"].(map[string]interface{})["c1"] = 6228 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validSkillConfig(6227, 6230)
			test.mutate(config)
			if err := svc.SetSkillConfig(context.Background(), 1, 7, config); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}

func TestChangeStateRestsOtherGuardingFairiesAndDeleteReportsActive(t *testing.T) {
	svc, repo, _, _, _ := newTestService(&domainchar.Character{ID: 1})
	state := defaultState()
	first := newFairy(7)
	first.State = fairyStateGuard
	second := newFairy(8)
	second.State = fairyStateGuard
	state.Fairies[first.ID] = first
	state.Fairies[second.ID] = second
	repo.charStates[1] = []*domainfeature.CharacterFeatureState{{CharacterID: 1, FeatureKey: rosterFeatureKey, State: state.toPersist()}}

	change, err := svc.ChangeState(context.Background(), 1, 8, fairyStateGuard)
	if err != nil {
		t.Fatalf("ChangeState: %v", err)
	}
	if len(change.RestedFairyIDs) != 1 || change.RestedFairyIDs[0] != 7 {
		t.Fatalf("rested ids = %+v, want [7]", change.RestedFairyIDs)
	}
	persisted := stateFromMap(repo.charStates[1][0].State)
	if persisted.Fairies[7].State != fairyStateRest || persisted.Fairies[8].State != fairyStateGuard {
		t.Fatalf("guard states not canonical: first=%d second=%d", persisted.Fairies[7].State, persisted.Fairies[8].State)
	}
	deleted, err := svc.Delete(context.Background(), 1, 8)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted.WasGuarding {
		t.Fatal("deleting active fairy did not report guarding state")
	}
}

func TestConcurrentLearnRequestsPreserveBothSkills(t *testing.T) {
	svc, repo, _, items, _ := newTestService(&domainchar.Character{ID: 1})
	seedFairy(t, repo, 1, newFairy(7))
	items.items[600] = &domainitem.Item{ID: 600, TemplateID: 3577, StackCount: 1}
	items.items[601] = &domainitem.Item{ID: 601, TemplateID: 3578, StackCount: 1}
	sd := newFakeSkillData(append(chainSkills("a", 5917, 5918, 5919), chainSkills("b", 5920, 5921, 5922)...))
	sd.items[3577] = &models.ItemTemplateTemplate{ID: 3577, Type: 521, PropType: 5917}
	sd.items[3578] = &models.ItemTemplateTemplate{ID: 3578, Type: 521, PropType: 5920}
	svc.SetSkillData(sd)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, itemID := range []int64{600, 601} {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			_, err := svc.OnSkill(context.Background(), 1, 7, id)
			errs <- err
		}(itemID)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent OnSkill: %v", err)
		}
	}
	persisted := stateFromMap(repo.charStates[1][0].State)
	learned := persisted.Fairies[7].SkillFlag
	if len(learned) != 2 {
		t.Fatalf("learned skills = %+v, want two preserved entries", learned)
	}
}

func TestBattleSkillRotationSnapshotsSelectedValidatedGroup(t *testing.T) {
	svc, repo, _, _, _ := newTestService(&domainchar.Character{ID: 1})
	f := newFairy(7)
	f.State = fairyStateGuard
	f.SkillFlag["s1"] = 6230
	f.SkillFlag["s2"] = 6260
	f.Config = validSkillConfig(6230, 6260)
	f.Config["select"] = 1
	seedFairy(t, repo, 1, f)
	svc.SetSkillData(newFakeSkillData([]*models.SkillTemplate{
		{ID: 6230, Level: 1, Kind: 1},
		{ID: 6260, Level: 1, Kind: 1},
	}))

	rotation, err := svc.BattleSkillRotation(context.Background(), 1)
	if err != nil {
		t.Fatalf("BattleSkillRotation: %v", err)
	}
	want := []int{6230, 6260, -1, -1, -1}
	if len(rotation) != len(want) {
		t.Fatalf("rotation = %+v", rotation)
	}
	for index := range want {
		if rotation[index] != want[index] {
			t.Fatalf("rotation[%d] = %d, want %d", index, rotation[index], want[index])
		}
	}
}

func TestBattleSkillRotationFailsClosedForMultipleGuards(t *testing.T) {
	svc, repo, _, _, _ := newTestService(&domainchar.Character{ID: 1})
	state := defaultState()
	state.Fairies[7] = newFairy(7)
	state.Fairies[8] = newFairy(8)
	state.Fairies[7].State = fairyStateGuard
	state.Fairies[8].State = fairyStateGuard
	repo.charStates[1] = []*domainfeature.CharacterFeatureState{{CharacterID: 1, FeatureKey: rosterFeatureKey, State: state.toPersist()}}

	rotation, err := svc.BattleSkillRotation(context.Background(), 1)
	if err != nil {
		t.Fatalf("BattleSkillRotation: %v", err)
	}
	if rotation != nil {
		t.Fatalf("multiple guards produced rotation %+v", rotation)
	}
}

func validSkillConfig(first, second int) map[string]interface{} {
	config := map[string]interface{}{"select": 0}
	for group := 0; group < 3; group++ {
		config[string(rune('0'+group))] = map[string]interface{}{
			"c0": first,
			"c1": second,
			"c2": -1,
			"c3": -1,
			"c4": -1,
		}
	}
	return config
}
