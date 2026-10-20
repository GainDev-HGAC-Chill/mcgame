package pet

import (
	"context"
	"math"
	"math/rand"

	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

const petResurrectionItemTemplateID = 1870

// petResurrectionBounds mirrors Flash's PET_GROWRATE_NUM tiers.
// Quality rolls have 0.01 granularity, and never cross the pet's color tier.
// White pets (at most 1.10) are not eligible.
func petResurrectionBounds(growRate float64) (min, max int, ok bool) {
	if math.IsNaN(growRate) || math.IsInf(growRate, 0) {
		return 0, 0, false
	}
	switch domainpet.ColorByGrowRate(growRate) {
	case 1:
		min, max = 111, 130
	case 2:
		min, max = 131, 160
	case 3:
		min, max = 161, 200
	case 4:
		min, max = 201, 250
	default:
		return 0, 0, false
	}
	// Reject malformed quality values even if they fall into the last Flash tier.
	if growRate < float64(min)/100-1e-9 || growRate > float64(max)/100+1e-9 {
		return 0, 0, false
	}
	return min, max, true
}

func defaultPetResurrectionRoll(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// UseCultivationItem implements the petPzXsd RPC (Phuc Sinh).
// The unmodified Flash client does not send an Apply/Cancel RPC. Therefore,
// better results are applied immediately; worse results are discarded.
// The pill is consumed for every valid roll, even if the quality does not rise.
func (s *Service) UseCultivationItem(ctx context.Context, charID, petID, itemID int64) (*PetItemUseResult, error) {
	it, err := s.loadPetFunctionItem(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if it.TemplateID != petResurrectionItemTemplateID || s.gameDataManager == nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetItem(it.TemplateID)
	if tpl == nil || int(tpl.Type) != 507 || int(tpl.PropType) != 4 || int(tpl.ProplNum) != 1 || int(tpl.UseType) != 4 {
		return nil, pkgerrors.ErrInvalidInput
	}

	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	if !p.IsBound() {
		return nil, pkgerrors.ErrInvalidInput
	}
	min, max, ok := petResurrectionBounds(p.GrowRate)
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}

	roll := s.resurrectionRoll
	if roll == nil {
		roll = defaultPetResurrectionRoll
	}
	rolledHundredths := roll(min, max)
	if rolledHundredths < min || rolledHundredths > max {
		return nil, pkgerrors.ErrInvalidInput
	}
	rolled := math.Round(float64(rolledHundredths)) / 100
	original := p.Clone()
	if rolled > original.GrowRate+1e-9 {
		p.GrowRate = rolled
		p.RecalculateStats()
	}

	result, err := s.finishPetItemUse(ctx, original, p, it)
	if err != nil {
		return nil, err
	}
	result.Property = "growRate"
	result.Before = original.GrowRate
	result.After = rolled // For the Flash 'before -> after' dialog, even for a lower roll.
	result.Success = p.GrowRate > original.GrowRate+1e-9
	return result, nil
}
