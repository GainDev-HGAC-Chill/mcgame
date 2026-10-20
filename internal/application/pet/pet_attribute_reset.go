package pet

import (
	"context"

	pkgerrors "mcgame-server/pkg/errors"
)

const petAttributeResetItemTemplateID = 1526

// ResetPetAttributesWithItem consumes Quy Nguyen Don only if a safe reset is possible.
func (s *Service) ResetPetAttributesWithItem(ctx context.Context, charID, petID, itemID int64) (*PetItemUseResult, error) {
	it, err := s.loadPetFunctionItem(ctx, charID, itemID)
	if err != nil {
		return nil, err
	}
	if s.gameDataManager == nil || it.TemplateID != petAttributeResetItemTemplateID {
		return nil, pkgerrors.ErrInvalidInput
	}
	itemTemplate := s.gameDataManager.GetItem(it.TemplateID)
	if itemTemplate == nil || int(itemTemplate.Type) != 507 || int(itemTemplate.PropType) != 1 || int(itemTemplate.UseType) != 4 {
		return nil, pkgerrors.ErrInvalidInput
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	creature := s.gameDataManager.GetCreature(p.TemplateID)
	// Flash PetFuncPanel.setXdInfo only permits classIds == 10.
	if creature == nil || int(creature.ClassIds) != 10 {
		return nil, pkgerrors.ErrInvalidInput
	}
	original := p.Clone()
	refunded, ok := p.ResetAllocatedAttributePoints()
	if !ok || refunded <= 0 {
		// No points to reset, or legacy Apt*Ex contains other untracked bonuses.
		// Never consume an item if we cannot safely distinguish them.
		return nil, pkgerrors.ErrInvalidInput
	}
	return s.finishPetItemUse(ctx, original, p, it)
}
