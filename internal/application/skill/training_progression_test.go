package skill

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type checkedTrainingCatalog struct {
	Records []checkedTrainingSkill `json:"records"`
}

type checkedTrainingSkill struct {
	ID       string `json:"id"`
	CodeName string `json:"codeName"`
	Kind     string `json:"kind"`
	Level    string `json:"level"`
	ReqClass string `json:"reqClass"`
	UseEnv   string `json:"useEnv"`
}

func TestTrainingSkillCatalogHasTwelveTenLevelFamilies(t *testing.T) {
	content, err := os.ReadFile("../../../docs/database/game_data/TBL_SKILL.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog checkedTrainingCatalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatal(err)
	}
	families := make(map[string]map[int]bool)
	count := 0
	for _, row := range catalog.Records {
		if row.UseEnv != "3" {
			continue
		}
		if row.Kind != "5" || row.ReqClass != "|1|2|3|4|5|6|" || row.CodeName == "" {
			t.Fatalf("invalid training row %+v", row)
		}
		level, err := strconv.Atoi(row.Level)
		if err != nil || level < 1 || level > 10 {
			t.Fatalf("invalid training level %+v", row)
		}
		if families[row.CodeName] == nil {
			families[row.CodeName] = make(map[int]bool)
		}
		if families[row.CodeName][level] {
			t.Fatalf("duplicate family level %s/%d", row.CodeName, level)
		}
		families[row.CodeName][level] = true
		count++
	}
	if count != 120 || len(families) != 12 {
		t.Fatalf("training rows=%d families=%d want 120/12", count, len(families))
	}
	for codeName, levels := range families {
		if len(levels) != 10 {
			t.Fatalf("family %s levels=%d want 10", codeName, len(levels))
		}
	}
}

func TestLearnTrainingSkillAllowsAllSixClassesAndRejectsInvalidStarts(t *testing.T) {
	for classID := 1; classID <= 6; classID++ {
		char := &domainchar.Character{ID: int64(classID), ClassID: classID, ClassRank: 1, Level: 80}
		service := newTrainingProgressionService(t, char, newLifeSkillTestRepo())
		learned, err := service.LearnTrainingSkill(context.Background(), char.ID, 8001)
		if err != nil || learned.SkillID != 8001 || learned.Level != 1 {
			t.Fatalf("class %d learned=%+v error=%v", classID, learned, err)
		}
		if _, err := service.LearnTrainingSkill(context.Background(), char.ID, 8001); !pkgerrors.Is(err, pkgerrors.ErrAlreadyExists) {
			t.Fatalf("class %d duplicate error=%v", classID, err)
		}
	}
	service := newTrainingProgressionService(t, &domainchar.Character{ID: 10, ClassID: 1, ClassRank: 1, Level: 80}, newLifeSkillTestRepo())
	if _, err := service.LearnTrainingSkill(context.Background(), 10, 8002); !errors.Is(err, ErrTrainingSkillInvalid) {
		t.Fatalf("level two start error=%v", err)
	}
	if _, err := service.LearnTrainingSkill(context.Background(), 10, 9001); !errors.Is(err, ErrTrainingSkillInvalid) {
		t.Fatalf("foreign environment error=%v", err)
	}
}

func TestUpgradeTrainingSkillRequiresExactFamilyStepAndSpendsResources(t *testing.T) {
	char := &domainchar.Character{ID: 1, ClassID: 4, ClassRank: 1, Level: 80, Experience: 500, MoneyBind: 300, SelectedMoneyType: 1}
	repo := newLifeSkillTestRepo(&domainskill.CharacterSkill{ID: 10, CharacterID: 1, SkillID: 8001, Level: 1})
	service := newTrainingProgressionService(t, char, repo)
	if _, err := service.UpgradeSkillWithResult(context.Background(), 1, 8003); !errors.Is(err, ErrNormalSkillInvalidStep) {
		t.Fatalf("skip error=%v", err)
	}
	if _, err := service.UpgradeSkillWithResult(context.Background(), 1, 8102); !errors.Is(err, ErrNormalSkillNotLearned) {
		t.Fatalf("cross-tier error=%v", err)
	}
	result, err := service.UpgradeSkillWithResult(context.Background(), 1, 8002)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Skill.SkillID != 8002 || result.Skill.Level != 2 {
		t.Fatalf("result=%+v", result)
	}
	if char.Experience != 400 || char.MoneyBind != 250 {
		t.Fatalf("resources=%d/%d want 400/250", char.Experience, char.MoneyBind)
	}
}

func newTrainingProgressionService(t *testing.T, char *domainchar.Character, repo domainskill.Repository) *Service {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	skills := []json.RawMessage{
		lifeSkillRawJSON(t, map[string]interface{}{"id": 8001, "name": "Training Basic", "kind": 5, "use_env": 3, "level": 1, "req_level": 80, "req_class": "|1|2|3|4|5|6|", "req_c_l": 1, "code_name": "TRAIN-A"}),
		lifeSkillRawJSON(t, map[string]interface{}{"id": 8002, "name": "Training Basic", "kind": 5, "use_env": 3, "level": 2, "req_level": 80, "req_class": "|1|2|3|4|5|6|", "req_c_l": 1, "code_name": "TRAIN-A", "exp_skill": 100, "price": 50}),
		lifeSkillRawJSON(t, map[string]interface{}{"id": 8003, "name": "Training Basic", "kind": 5, "use_env": 3, "level": 3, "req_level": 80, "req_class": "|1|2|3|4|5|6|", "req_c_l": 1, "code_name": "TRAIN-A", "exp_skill": 200, "price": 75}),
		lifeSkillRawJSON(t, map[string]interface{}{"id": 8101, "name": "Training Intermediate", "kind": 5, "use_env": 3, "level": 1, "req_level": 80, "req_class": "|1|2|3|4|5|6|", "req_c_l": 1, "code_name": "TRAIN-B"}),
		lifeSkillRawJSON(t, map[string]interface{}{"id": 8102, "name": "Training Intermediate", "kind": 5, "use_env": 3, "level": 2, "req_level": 80, "req_class": "|1|2|3|4|5|6|", "req_c_l": 1, "code_name": "TRAIN-B", "exp_skill": 100, "price": 50}),
		lifeSkillRawJSON(t, map[string]interface{}{"id": 9001, "name": "Foreign", "kind": 5, "use_env": 4, "level": 1, "req_level": 1, "req_class": "|1|", "code_name": "FOREIGN"}),
	}
	if err := manager.GetCache().LoadTable(models.TableSkill, skills); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, zap.NewNop())
	service.SetCharacterRepository(&lifeSkillTestCharacterRepo{char: char})
	service.SetGameDataManager(manager)
	return service
}
