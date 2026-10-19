package pet

import (
	"context"
	"math"
	"math/rand"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

// petWashRoll picks an integer aptitude in the inclusive interval [min, max].
func petWashRoll(min, max int) int {
	if max <= min {
		return min
	}
	return min + rand.Intn(max-min+1)
}

func petWashSlot(p *domainpet.Pet, creature *models.CreatureTemplate, slot int) (string, *int, float64) {
	switch slot {
	case 1:
		return "aptStrength", &p.AptStrength, creature.AptStrength
	case 2:
		return "aptAgility", &p.AptAgility, creature.AptAgility
	case 3:
		return "aptStamina", &p.AptStamina, creature.AptStamina
	case 4:
		return "aptIntelligence", &p.AptIntelligence, creature.AptIntelligence
	case 5:
		return "aptEnergy", &p.AptEnergy, creature.AptEnergy
	default:
		return "", nil, 0
	}
}

type petWashPillKind struct {
	slot     int
	advanced bool
}

// These 10 item IDs and their propl_num slots are from
// supabase/seeds/data_tbl_item_template.sql. Reject other 507/2 items.
var petWashPills = map[int]petWashPillKind{
	275: {1, false}, 276: {2, false}, 277: {3, false},
	278: {4, false}, 279: {5, false},
	1652: {1, true}, 1653: {2, true}, 1654: {3, true},
	1655: {4, true}, 1656: {5, true},
}

// PetXsd implements the existing Flash client RPC. Unlike the older
// implementation, it changes the selected BASIC aptitude and never GrowRateAdd.
// The current Flash onXsd callback has no confirmation RPC: pressing the
// dialog's Apply/Discard text only dismisses a one-button Alert. Therefore this
// compatibility implementation persists improvements immediately and discards
// lower/equal candidates. Proper pending/confirm/cancel requires rebuilding
// the Flash client.
func (s *Service) PetXsd(ctx context.Context, charID, petID, itemID int64) (*PetItemUseResult, error) {
	it, err := s.loadPetFunctionItem(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if s.gameDataManager == nil {
		return nil, pkgerrors.ErrInvalidInput
	}

	itemTemplate := s.gameDataManager.GetItem(it.TemplateID)
	pill, recognized := petWashPills[it.TemplateID]
	if !recognized || itemTemplate == nil || int(itemTemplate.Type) != 507 || int(itemTemplate.UseType) != 4 ||
		int(itemTemplate.PropType) != 2 || itemTemplate.ProplNum != float64(pill.slot) {
		return nil, pkgerrors.ErrInvalidInput
	}

	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	if !p.IsBound() {
		return nil, pkgerrors.ErrInvalidInput
	}

	creature := s.gameDataManager.GetCreature(int(p.TemplateID))
	if creature == nil {
		return nil, pkgerrors.ErrInvalidInput
	}

	name, aptitude, base := petWashSlot(p, creature, pill.slot)
	if aptitude == nil || math.IsNaN(base) || math.IsInf(base, 0) || base <= 0 || base > 1_000_000 {
		return nil, pkgerrors.ErrInvalidInput
	}
	max := int(math.Floor(base*1.2 + 0.5)) // AS3 Math.round(base * 1.2)
	min := int(math.Floor(base*0.8 + 0.5)) // 80% - 120% (item description)
	if min < 1 {
		min = 1
	}
	if max < min {
		return nil, pkgerrors.ErrInvalidInput
	}

	before := *aptitude
	// The client has a dedicated "aptHigh" branch. Reaching the cap must not
	// consume another pill or change any property.
	if before >= max {
		return &PetItemUseResult{
			Pet: p, ItemID: it.ID, ItemSID: it.CalculateSID(),
			RemainingStack: it.StackCount, Property: name,
			Before: float64(before), After: float64(before), AppliedValue: float64(before),
			Success: true, AptHigh: true,
		}, nil
	}

	if pill.advanced {
		// Advanced pills may not decrease aptitude. Choose an integer between
		// current+1 and the same client-visible aptitude cap.
		min = before + 1
		if min < 1 {
			min = 1
		}
	}
	roll := s.washRoll
	if roll == nil {
		roll = petWashRoll
	}
	after := roll(min, max)
	if after < min || after > max {
		return nil, pkgerrors.ErrInvalidInput
	}

	original := p.Clone()
	// This unmodified Flash client has no confirmation RPC. Its single-button
	// dialog labels a lower/equal roll as "Hủy bỏ". Consume the normal pill,
	// show the candidate roll, but keep the original aptitude in persistence.
	// Improvements are accepted immediately to match the "Ứng dụng" dialog.
	persisted := before
	if after > before {
		*aptitude = after
		p.RecalculateStats()
		persisted = after
	}
	result, err := s.finishPetItemUse(ctx, original, p, it)
	if err != nil {
		return nil, err
	}
	result.Property = name
	result.Before = float64(before)
	result.After = float64(after)
	result.AppliedValue = float64(persisted)
	result.Success = true
	result.AptHigh = false // only preexisting max suppresses the roll dialog

	s.logger.Info("Pet aptitude washed",
		zap.Int64("character_id", charID),
		zap.Int64("pet_id", petID),
		zap.Int64("item_id", itemID),
		zap.String("aptitude", name),
		zap.Int("before", before),
		zap.Int("after", after),
		zap.Int("applied", persisted),
		zap.Bool("advanced", pill.advanced))
	return result, nil
}
