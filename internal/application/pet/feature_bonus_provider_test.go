// Open-sourced by BaoLT

package pet

import (
	"context"
	"encoding/json"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type additivePetBonusProvider struct {
	hp int
}

func (p additivePetBonusProvider) AggregatePetBonuses(ctx context.Context, petID int64) (domainpet.EquipmentStatBonuses, domainpet.SkillStatBonuses) {
	equipment := domainpet.NewEquipmentStatBonuses()
	equipment.AddFlat(1, p.hp)
	return equipment, domainpet.NewSkillStatBonuses()
}

func TestFeatureBonusProvidersComposeAdditively(t *testing.T) {
	manager := gamedata.NewManager(nil, zap.NewNop())
	raw, err := json.Marshal(map[string]interface{}{"id": 101, "class_id": 1, "apt_stamina": 1000})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.GetCache().LoadTable(models.TableCreature, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, zap.NewNop())
	service.SetGameDataManager(manager)
	service.SetFeatureBonusProvider(additivePetBonusProvider{hp: 10})
	service.AddFeatureBonusProvider(additivePetBonusProvider{hp: 20})
	pet := &domainpet.Pet{ID: 7, TemplateID: 101, Level: 1, AptStamina: 1000, GrowRate: 1}

	service.enrichPet(context.Background(), pet)
	if got := pet.EquipmentBonuses().Flat[1]; got != 30 {
		t.Fatalf("composed HP bonus = %d, want 30", got)
	}
}
