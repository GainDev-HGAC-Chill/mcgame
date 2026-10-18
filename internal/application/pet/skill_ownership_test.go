package pet

import (
	"context"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"

	"go.uber.org/zap"
)

func TestOwnedSkillLevelUsesCanonicalPetSkillSlots(t *testing.T) {
	repo := &openSkillPetRepo{pets: map[int64]*domainpet.Pet{
		2484410: {
			ID:          2484410,
			CharacterID: 1,
			TemplateID:  1001,
			GrowRate:    1,
			Property: map[string]interface{}{
				"skill1": 5528,
			},
		},
	}}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(newPetSkillBookGameData(t))

	level, owned, err := service.OwnedSkillLevel(context.Background(), 1, 2484410, 5528)
	if err != nil {
		t.Fatal(err)
	}
	if !owned || level != 2 {
		t.Fatalf("OwnedSkillLevel() = (%d, %v), want (2, true)", level, owned)
	}

	if _, owned, err := service.OwnedSkillLevel(context.Background(), 1, 2484410, 5527); err != nil || owned {
		t.Fatalf("OwnedSkillLevel(unowned) = (_, %v, %v), want (_, false, nil)", owned, err)
	}
}
