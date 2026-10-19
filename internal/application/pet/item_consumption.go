package pet

import (
	"context"
	"math/rand"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

type PetItemUseResult struct {
	Pet            *domainpet.Pet
	ItemID         int64
	ItemSID        int
	RemainingStack int
	Property       string
	Before         float64
	After          float64
	Success        bool
	SafeItemID     int64
	SafeItemSID    int
	SafeRemaining  int
	AptHigh        bool
	AppliedValue   float64
}

var petStarSuccessRates = [...]int{0, 100, 100, 90, 80, 70, 60, 50, 30, 30, 30, 30, 20}

func (s *Service) loadPetFunctionItem(ctx context.Context, charID, itemID int64) (*domainitem.Item, error) {
	if s.itemRepo == nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	it, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if it == nil || it.CharacterID != charID || it.SlotType != domainitem.SlotTypeBag || it.StackCount <= 0 {
		return nil, pkgerrors.ErrItemNotOwned
	}
	return it, nil
}

func (s *Service) finishPetItemUse(ctx context.Context, originalPet *domainpet.Pet, updatedPet *domainpet.Pet, it *domainitem.Item) (*PetItemUseResult, error) {
	if err := s.petRepo.Save(ctx, updatedPet); err != nil {
		return nil, err
	}
	remaining := it.StackCount - 1
	var err error
	if remaining == 0 {
		err = s.itemRepo.Delete(ctx, it.ID)
	} else {
		err = s.itemRepo.UpdateStack(ctx, it.ID, remaining)
	}
	if err != nil {
		_ = s.petRepo.Save(ctx, originalPet)
		return nil, err
	}
	return &PetItemUseResult{Pet: updatedPet, ItemID: it.ID, ItemSID: it.CalculateSID(), RemainingStack: remaining}, nil
}

func (s *Service) UpgradeStarWithItem(ctx context.Context, charID, petID, itemID int64, safeItemIDs ...int64) (*PetItemUseResult, error) {
	it, err := s.loadPetFunctionItem(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if s.gameDataManager == nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetItem(it.TemplateID)
	if tpl == nil || int(tpl.Type) != 507 || int(tpl.UseType) != 4 || int(tpl.PropType) != 3 || tpl.ProplNum <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	var safeItem *domainitem.Item
	if len(safeItemIDs) > 0 && safeItemIDs[0] > 0 {
		if safeItemIDs[0] == itemID {
			return nil, pkgerrors.ErrInvalidInput
		}
		safeItem, err = s.loadPetFunctionItem(ctx, charID, safeItemIDs[0])
		if err != nil {
			return nil, err
		}
		safeTemplate := s.gameDataManager.GetItem(safeItem.TemplateID)
		if safeTemplate == nil || int(safeTemplate.Type) != 507 || int(safeTemplate.UseType) != 4 || int(safeTemplate.PropType) != 6 {
			return nil, pkgerrors.ErrInvalidInput
		}
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	original := p.Clone()
	targetStar := p.UpgradeNum + 1
	if targetStar > domainpet.MaxPetStars {
		return nil, pkgerrors.ErrInvalidInput
	}
	success := s.starRoll(petStarSuccessRates[targetStar])
	if success {
		p.AddStar()
		p.AddGrowthRate(tpl.ProplNum / 100)
	} else if p.UpgradeNum >= 8 && safeItem == nil {
		p.ResetStarEnhancement()
	}
	result, err := s.finishPetItemUse(ctx, original, p, it)
	if err != nil {
		return nil, err
	}
	result.Success = success
	if safeItem == nil {
		return result, nil
	}
	remaining := safeItem.StackCount - 1
	if remaining == 0 {
		err = s.itemRepo.Delete(ctx, safeItem.ID)
	} else {
		err = s.itemRepo.UpdateStack(ctx, safeItem.ID, remaining)
	}
	if err != nil {
		_ = s.petRepo.Save(ctx, original)
		return nil, err
	}
	result.SafeItemID = safeItem.ID
	result.SafeItemSID = safeItem.CalculateSID()
	result.SafeRemaining = remaining
	return result, nil
}

func defaultPetStarRoll(rate int) bool {
	return rate >= 100 || rate > 0 && rand.Intn(100) < rate
}

func (s *Service) ClearStarsWithItem(ctx context.Context, charID, petID, itemID int64) (*PetItemUseResult, error) {
	it, err := s.loadPetFunctionItem(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	original := p.Clone()
	p.ClearStars()
	return s.finishPetItemUse(ctx, original, p, it)
}

func (s *Service) CultivateWithItem(ctx context.Context, charID, petID, itemID int64) (*PetItemUseResult, error) {
	it, err := s.loadPetFunctionItem(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if s.gameDataManager == nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetItem(it.TemplateID)
	if tpl == nil || int(tpl.PropType) < 1 || int(tpl.PropType) > 5 {
		return nil, pkgerrors.ErrInvalidInput
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	original := p.Clone()
	applyPetCultivation(p, int(tpl.PropType))
	return s.finishPetItemUse(ctx, original, p, it)
}
