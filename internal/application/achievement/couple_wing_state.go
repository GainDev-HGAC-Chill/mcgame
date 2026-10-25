// Open-sourced by BaoLT

// Couple-wing state reader proves matching equipped wing identity and color for current spouses.
package achievement

import (
	"context"
	"errors"
	"fmt"

	domainitem "mcgame-server/internal/domain/item"
	domainmarriage "mcgame-server/internal/domain/marriage"
	pkgerrors "mcgame-server/pkg/errors"
)

type CoupleWingMarriageRepository interface {
	FindByPartner(ctx context.Context, characterID int64) (*domainmarriage.MarriageRecord, error)
}

type CoupleWingStateReader struct {
	marriages CoupleWingMarriageRepository
	items     CharacterLoadoutStateRepository
	catalog   CharacterLoadoutStateCatalog
}

func NewCoupleWingStateReader(marriages CoupleWingMarriageRepository, items CharacterLoadoutStateRepository, catalog CharacterLoadoutStateCatalog) *CoupleWingStateReader {
	return &CoupleWingStateReader{marriages: marriages, items: items, catalog: catalog}
}

func (r *CoupleWingStateReader) Name() string {
	return "couple-wing"
}

func (r *CoupleWingStateReader) RequirementIDs() []int64 {
	return []int64{373, 374, 375}
}

func (r *CoupleWingStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.marriages == nil || r.items == nil || r.catalog == nil {
		return nil, fmt.Errorf("couple-wing state dependencies not configured")
	}
	marriage, err := r.marriages.FindByPartner(ctx, characterID)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			return coupleWingProofs(0), nil
		}
		return nil, err
	}
	if marriage == nil || marriage.ID <= 0 || (marriage.Partner1ID != characterID && marriage.Partner2ID != characterID) {
		return nil, fmt.Errorf("invalid marriage state for character %d", characterID)
	}
	partnerID := marriage.Partner1ID
	if partnerID == characterID {
		partnerID = marriage.Partner2ID
	}
	if partnerID <= 0 || partnerID == characterID {
		return nil, fmt.Errorf("invalid marriage partner for character %d", characterID)
	}
	characterWing, err := r.equippedWing(ctx, characterID)
	if err != nil {
		return nil, err
	}
	partnerWing, err := r.equippedWing(ctx, partnerID)
	if err != nil {
		return nil, err
	}
	matchingColor := 0
	if characterWing != nil && partnerWing != nil && characterWing.TemplateID == partnerWing.TemplateID && characterWing.ColorCode == partnerWing.ColorCode {
		matchingColor = characterWing.ColorCode
	}
	return coupleWingProofs(matchingColor), nil
}

func (r *CoupleWingStateReader) equippedWing(ctx context.Context, characterID int64) (*domainitem.Item, error) {
	items, err := r.items.FindEquipped(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var wing *domainitem.Item
	for _, item := range items {
		if item == nil || item.CharacterID != characterID || item.ItemType != domainitem.ItemTypeEquipment || item.SlotType != domainitem.SlotTypeEquipped {
			continue
		}
		template := r.catalog.GetEquipment(item.TemplateID)
		if template == nil || template.ID != int64(item.TemplateID) {
			return nil, fmt.Errorf("character %d has unknown equipped template %d", characterID, item.TemplateID)
		}
		position, positionOK := exactPetCatalogInt(template.Position)
		kind, kindOK := exactPetCatalogInt(template.Kind)
		if !positionOK || !kindOK {
			return nil, fmt.Errorf("equipped template %d has invalid topology", item.TemplateID)
		}
		if position != 22 || kind != 13 {
			continue
		}
		if wing != nil {
			return nil, fmt.Errorf("character %d has multiple equipped wings", characterID)
		}
		color, colorErr := petEquipmentStateColor(item)
		if colorErr != nil || color < 3 || color > 5 {
			return nil, fmt.Errorf("character %d has invalid equipped wing color", characterID)
		}
		copy := *item
		copy.ColorCode = color
		wing = &copy
	}
	return wing, nil
}

func coupleWingProofs(color int) []StateProof {
	return []StateProof{
		stateProof(373, boolStateProgress(color == 3), 1),
		stateProof(374, boolStateProgress(color == 4), 1),
		stateProof(375, boolStateProgress(color == 5), 1),
	}
}

func boolStateProgress(value bool) int {
	if value {
		return 1
	}
	return 0
}
