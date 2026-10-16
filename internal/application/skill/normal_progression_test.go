// Open-sourced by BaoLT

package skill

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func TestResolveOrdinarySkillBookUsesClassFingerprint(t *testing.T) {
	service := newOrdinaryProgressionService(t, &domainchar.Character{ID: 1, ClassID: 6, Level: 100}, newLifeSkillTestRepo())

	template, err := service.ResolveOrdinarySkillBook(1419)
	if err != nil {
		t.Fatalf("ResolveOrdinarySkillBook() error = %v", err)
	}
	if template.ID != 3424 {
		t.Fatalf("skill id = %d, want 3424", int(template.ID))
	}
}

func TestLearnOrdinarySkillByBookRejectsDuplicateFamily(t *testing.T) {
	existing := &domainskill.CharacterSkill{ID: 10, CharacterID: 1, SkillID: 3425, Level: 2}
	service := newOrdinaryProgressionService(t, &domainchar.Character{ID: 1, ClassID: 6, Level: 100}, newLifeSkillTestRepo(existing))

	_, _, err := service.LearnOrdinarySkillByBook(context.Background(), 1, 1419)
	if !pkgerrors.Is(err, pkgerrors.ErrAlreadyExists) {
		t.Fatalf("error = %v, want %v", err, pkgerrors.ErrAlreadyExists)
	}
}

func TestUpgradeSkillRequiresExactNextLevelAndSpendsResources(t *testing.T) {
	char := &domainchar.Character{ID: 1, ClassID: 6, Level: 100, Experience: 500, MoneyBind: 300, SelectedMoneyType: 1}
	existing := &domainskill.CharacterSkill{ID: 10, CharacterID: 1, SkillID: 3424, Level: 1}
	repo := newLifeSkillTestRepo(existing)
	service := newOrdinaryProgressionService(t, char, repo)

	if _, err := service.UpgradeSkillWithResult(context.Background(), char.ID, 3426); !errors.Is(err, ErrNormalSkillInvalidStep) {
		t.Fatalf("skip error = %v, want %v", err, ErrNormalSkillInvalidStep)
	}
	result, err := service.UpgradeSkillWithResult(context.Background(), char.ID, 3425)
	if err != nil {
		t.Fatalf("UpgradeSkillWithResult() error = %v", err)
	}
	if !result.Changed || result.Skill.SkillID != 3425 || result.Skill.Level != 2 {
		t.Fatalf("result = %#v", result)
	}
	if char.Experience != 400 || char.MoneyBind != 250 {
		t.Fatalf("resources = exp %d moneyBind %d, want 400/250", char.Experience, char.MoneyBind)
	}
}

func TestUpgradeSkillRejectsUnlearnedAndForeignEnvironment(t *testing.T) {
	service := newOrdinaryProgressionService(t, &domainchar.Character{ID: 1, ClassID: 6, Level: 100, Experience: 500, MoneyBind: 300}, newLifeSkillTestRepo())

	if _, err := service.UpgradeSkillWithResult(context.Background(), 1, 3425); !errors.Is(err, ErrNormalSkillNotLearned) {
		t.Fatalf("unlearned error = %v, want %v", err, ErrNormalSkillNotLearned)
	}
	if _, err := service.UpgradeSkillWithResult(context.Background(), 1, 7001); !pkgerrors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatalf("foreign error = %v, want %v", err, pkgerrors.ErrInvalidInput)
	}
}

func newOrdinaryProgressionService(t *testing.T, char *domainchar.Character, repo domainskill.Repository) *Service {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	items := []json.RawMessage{
		lifeSkillRawJSON(t, map[string]interface{}{
			"id": 1419, "name": "Sách Kỹ Năng - Nộ Lôi Chiến", "type": 505,
			"req_class": "|6|", "req_level": 1,
		}),
	}
	skills := []json.RawMessage{
		lifeSkillRawJSON(t, map[string]interface{}{
			"id": 3424, "name": "Nộ Lôi Chiến", "type": 1, "kind": 1, "use_env": 0,
			"level": 1, "req_level": 1, "req_class": "|6|", "code_name": "SKILL467301",
		}),
		lifeSkillRawJSON(t, map[string]interface{}{
			"id": 3425, "name": "Nộ Lôi Chiến", "type": 1, "kind": 1, "use_env": 0,
			"level": 2, "req_level": 1, "req_class": "|6|", "code_name": "SKILL467301",
			"exp_skill": 100, "price": 50,
		}),
		lifeSkillRawJSON(t, map[string]interface{}{
			"id": 3426, "name": "Nộ Lôi Chiến", "type": 1, "kind": 1, "use_env": 0,
			"level": 3, "req_level": 1, "req_class": "|6|", "code_name": "SKILL467301",
			"exp_skill": 200, "price": 75,
		}),
		lifeSkillRawJSON(t, map[string]interface{}{
			"id": 60601, "name": "Nộ Lôi Chiến", "type": 1, "kind": 1, "use_env": 0,
			"level": 1, "req_level": 1, "req_class": "|6@|", "code_name": "SKILL1000606",
		}),
		lifeSkillRawJSON(t, map[string]interface{}{
			"id": 7001, "name": "Foreign", "type": 1, "kind": 1, "use_env": 4,
			"level": 2, "req_level": 1, "req_class": "|6|", "code_name": "foreign",
		}),
	}
	if err := manager.GetCache().LoadTable(models.TableItemTemplate, items); err != nil {
		t.Fatalf("LoadTable(ItemTemplate) error = %v", err)
	}
	if err := manager.GetCache().LoadTable(models.TableSkill, skills); err != nil {
		t.Fatalf("LoadTable(Skill) error = %v", err)
	}
	service := NewService(repo, zap.NewNop())
	service.SetCharacterRepository(&lifeSkillTestCharacterRepo{char: char})
	service.SetGameDataManager(manager)
	return service
}
