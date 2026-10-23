// Open-sourced by BaoLT

package skill

import (
	"context"
	"encoding/json"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainpet "mcgame-server/internal/domain/pet"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type trainingPetTestRepo struct {
	pet *domainpet.Pet
}

func (r *trainingPetTestRepo) FindByID(ctx context.Context, id int64) (*domainpet.Pet, error) {
	if r.pet == nil || r.pet.ID != id {
		return nil, nil
	}
	copy := *r.pet
	return &copy, nil
}

func TestTrainingBonusesStackFamiliesAndKeepHighestFamilyLevel(t *testing.T) {
	manager := newTrainingBonusGameData(t)
	repo := newLifeSkillTestRepo(
		&domainskill.CharacterSkill{CharacterID: 5, SkillID: 3857, Level: 1},
		&domainskill.CharacterSkill{CharacterID: 5, SkillID: 3881, Level: 10},
		&domainskill.CharacterSkill{CharacterID: 5, SkillID: 3886, Level: 2},
		&domainskill.CharacterSkill{CharacterID: 5, SkillID: 4140, Level: 10},
	)
	service := NewService(repo, zap.NewNop())
	service.SetCharacterRepository(&lifeSkillTestCharacterRepo{char: &domainchar.Character{ID: 5, ClassID: 5}})
	service.SetGameDataManager(manager)

	bonuses := service.AggregateCharacterStatBonuses(context.Background(), 5)
	if got := bonuses.Flat[domainchar.PropMaxHP]; got != 2230 {
		t.Fatalf("max HP bonus = %d, want 2230", got)
	}
	if got := bonuses.Flat[domainchar.PropSpeed]; got != 950 {
		t.Fatalf("speed bonus = %d, want 950", got)
	}
}

func TestTrainingPetBonusesUseOwnerSkillsAndFrozenFamilyCoefficient(t *testing.T) {
	manager := newTrainingBonusGameData(t)
	repo := newLifeSkillTestRepo(
		&domainskill.CharacterSkill{CharacterID: 5, SkillID: 3881, Level: 10},
		&domainskill.CharacterSkill{CharacterID: 5, SkillID: 3886, Level: 2},
		&domainskill.CharacterSkill{CharacterID: 5, SkillID: 4140, Level: 10},
	)
	service := NewService(repo, zap.NewNop())
	service.SetPetRepository(&trainingPetTestRepo{pet: &domainpet.Pet{ID: 9, CharacterID: 5, TemplateID: 101}})
	service.SetGameDataManager(manager)

	equipment, skills := service.AggregatePetBonuses(context.Background(), 9)
	if got := equipment.Flat[1]; got != 1144 {
		t.Fatalf("pet HP bonus = %d, want 1144", got)
	}
	if got := equipment.Flat[11]; got != 586 {
		t.Fatalf("pet speed bonus = %d, want 586", got)
	}
	if !skills.IsZero() {
		t.Fatalf("training unexpectedly produced skill bonuses: %+v", skills)
	}
}

func TestTrainingBonusesFailClosedOnCatalogOrPersistedLevelDrift(t *testing.T) {
	manager := newTrainingBonusGameData(t)
	repo := newLifeSkillTestRepo(&domainskill.CharacterSkill{CharacterID: 5, SkillID: 3881, Level: 9})
	service := NewService(repo, zap.NewNop())
	service.SetCharacterRepository(&lifeSkillTestCharacterRepo{char: &domainchar.Character{ID: 5, ClassID: 5}})
	service.SetGameDataManager(manager)
	if got := service.AggregateCharacterStatBonuses(context.Background(), 5).Flat[domainchar.PropMaxHP]; got != 0 {
		t.Fatalf("drifted persisted level bonus = %d, want 0", got)
	}

	badManager := newTrainingBonusGameDataRows(t, false)
	service.SetGameDataManager(badManager)
	repo.skills[1].Level = 10
	if got := service.AggregateCharacterStatBonuses(context.Background(), 5).Flat[domainchar.PropMaxHP]; got != 0 {
		t.Fatalf("incomplete catalog bonus = %d, want 0", got)
	}
}

func newTrainingBonusGameData(t *testing.T) *gamedata.Manager {
	t.Helper()
	return newTrainingBonusGameDataRows(t, true)
}

func newTrainingBonusGameDataRows(t *testing.T, complete bool) *gamedata.Manager {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	skillRows := make([]json.RawMessage, 0, len(trainingFamilies)*10)
	for familyIndex, family := range trainingFamilies {
		for level, skillID := range family.skillIDs {
			if !complete && familyIndex == len(trainingFamilies)-1 && level == 9 {
				continue
			}
			skillRows = append(skillRows, trainingRawJSON(t, map[string]interface{}{
				"id": skillID, "code_name": family.codeName, "kind": 5, "level": level + 1,
				"req_class": "|1|2|3|4|5|6|", "use_env": 3,
			}))
		}
	}
	if err := manager.GetCache().LoadTable(models.TableSkill, skillRows); err != nil {
		t.Fatal(err)
	}
	classRow := trainingRawJSON(t, map[string]interface{}{
		"id": 5, "apt_agility": 2000, "apt_energy": 2000, "apt_intelligence": 2000,
		"apt_stamina": 2000, "apt_strength": 2000,
	})
	if err := manager.GetCache().LoadTable(models.TableClass, []json.RawMessage{classRow}); err != nil {
		t.Fatal(err)
	}
	creatureRow := trainingRawJSON(t, map[string]interface{}{"id": 101, "class_id": 2})
	if err := manager.GetCache().LoadTable(models.TableCreature, []json.RawMessage{creatureRow}); err != nil {
		t.Fatal(err)
	}
	return manager
}

func trainingRawJSON(t *testing.T, value map[string]interface{}) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
