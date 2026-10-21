package pet

import (
	"context"
	"errors"
	"fmt"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type atomicSkillBookPetRepo struct {
	*openSkillPetRepo
	err    error
	called bool
}

func (r *atomicSkillBookPetRepo) ApplySkillBook(ctx context.Context, mutation domainpet.SkillBookMutation) (int, error) {
	r.called = true
	if r.err != nil {
		return 0, r.err
	}
	if err := r.Save(ctx, mutation.Pet); err != nil {
		return 0, err
	}
	return mutation.ExpectedBookStack - 1, nil
}

func TestPetSuperSkillMappingsReplaceExactLevelThreePrerequisites(t *testing.T) {
	for bookID, spec := range petSuperSkillBooks {
		t.Run(fmt.Sprintf("book_%d", bookID), func(t *testing.T) {
			property := make(map[string]interface{}, domainpet.TotalPetSkillSlots)
			for slot := 1; slot <= domainpet.TotalPetSkillSlots; slot++ {
				property[petSkillSlotKey(slot)] = 7000 + slot
			}
			property["skill8"] = spec.Prerequisites[0]
			property["skill3"] = spec.Prerequisites[1]
			property["skill11"] = spec.Prerequisites[2]
			p := &domainpet.Pet{Property: property}

			updates, ok := petSuperSkillSlotUpdates(p, spec)
			if !ok {
				t.Fatal("petSuperSkillSlotUpdates() rejected exact prerequisites")
			}
			if len(updates) != 3 || updates[0].Slot != 3 || updates[0].SkillID != spec.SuperChain[0] || updates[1].Slot != 8 || updates[1].SkillID != 0 || updates[2].Slot != 11 || updates[2].SkillID != 0 {
				t.Fatalf("updates = %#v", updates)
			}
		})
	}
}

func TestPetSuperSkillRequirementsFailClosed(t *testing.T) {
	spec := petSuperSkillBooks[3216]
	base := &domainpet.Pet{Property: map[string]interface{}{"skill1": spec.Prerequisites[0], "skill2": spec.Prerequisites[1], "skill3": spec.Prerequisites[2]}}

	missing := clonePetSnapshot(base)
	setPetSkillSlotValue(missing, 2, 3344)
	if _, ok := petSuperSkillSlotUpdates(missing, spec); ok {
		t.Fatal("level-two prerequisite was accepted")
	}

	ownedSuper := clonePetSnapshot(base)
	setPetSkillSlotValue(ownedSuper, 4, spec.SuperChain[2])
	if _, ok := petSuperSkillSlotUpdates(ownedSuper, spec); ok {
		t.Fatal("existing Super chain was accepted")
	}

	duplicate := clonePetSnapshot(base)
	setPetSkillSlotValue(duplicate, 4, spec.Prerequisites[0])
	if _, ok := petSuperSkillSlotUpdates(duplicate, spec); ok {
		t.Fatal("duplicate prerequisite was accepted")
	}
}

func TestPetSuperSkillBlocksMatchingOrdinaryFamily(t *testing.T) {
	service := NewService(nil, zap.NewNop())
	p := &domainpet.Pet{Property: map[string]interface{}{"skill1": 5528}}
	for _, name := range []string{"Thụ Bì Ngoại Sáo", "Thụ Bì Ngoại Sáo Trung Cấp", "Thụ Bì Ngoại Sáo Cao Cấp"} {
		if !service.petHasSuperSkillForOrdinary(p, &models.SkillTemplate{Name: name}) {
			t.Fatalf("matching ordinary skill %q was not blocked", name)
		}
	}
	if service.petHasSuperSkillForOrdinary(p, &models.SkillTemplate{Name: "Cổ Long Chi Hồn"}) {
		t.Fatal("unrelated ordinary family was blocked")
	}
}

func TestPetSuperSkillBookUsesQualityAndCreatureClass(t *testing.T) {
	service := NewService(nil, zap.NewNop())
	book := &models.ItemTemplateTemplate{ProplNum: 3, ReqClass: "|1|2|3|4|5|"}
	p := &domainpet.Pet{CreatureData: map[string]interface{}{"classId": 5, "classIds": 99, "qLevel": 3}}
	if !service.petCanUseSuperSkillBook(p, book) {
		t.Fatal("matching qLevel and classId were rejected")
	}
	p.CreatureData["qLevel"] = 2
	if service.petCanUseSuperSkillBook(p, book) {
		t.Fatal("Pet below required qLevel was accepted")
	}
	p.CreatureData["qLevel"] = 3
	p.CreatureData["classId"] = 6
	if service.petCanUseSuperSkillBook(p, book) {
		t.Fatal("classId excluded by item reqClass was accepted")
	}
}

func TestLearnSuperSkillBookAtomicFailureLeavesPetAndBookUnchanged(t *testing.T) {
	mutationErr := errors.New("atomic skill-book failure")
	baseRepo := &openSkillPetRepo{pets: map[int64]*domainpet.Pet{
		1: {
			ID: 1, CharacterID: 7, TemplateID: 1001, Level: 1, GrowRate: 1,
			CreatureData: map[string]interface{}{"classId": 3, "classIds": 10, "qLevel": 3},
			Property:     map[string]interface{}{"skill1": 2830, "skill2": 3345, "skill3": 5156},
		},
	}}
	petRepo := &atomicSkillBookPetRepo{openSkillPetRepo: baseRepo, err: mutationErr}
	itemRepo := &bookSkillItemRepo{items: map[int64]*domainitem.Item{9: {ID: 9, CharacterID: 7, TemplateID: 3216, StackCount: 1}}}
	service := NewService(petRepo, zap.NewNop())
	service.SetItemRepository(itemRepo)
	service.SetGameDataManager(newPetSkillBookGameData(t))

	if _, err := service.LearnSkillFromBook(context.Background(), 7, 1, 9); !errors.Is(err, mutationErr) {
		t.Fatalf("error = %v, want %v", err, mutationErr)
	}
	if !petRepo.called {
		t.Fatal("atomic repository was not called")
	}
	stored := baseRepo.pets[1]
	if petSkillSlotValue(stored, 1) != 2830 || petSkillSlotValue(stored, 2) != 3345 || petSkillSlotValue(stored, 3) != 5156 {
		t.Fatalf("stored prerequisites changed: %#v", stored.Property)
	}
	if itemRepo.items[9].StackCount != 1 {
		t.Fatalf("book stack = %d, want 1", itemRepo.items[9].StackCount)
	}
}
