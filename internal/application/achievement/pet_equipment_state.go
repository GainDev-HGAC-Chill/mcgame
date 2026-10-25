// Open-sourced by BaoLT

// Pet-equipment state reader proves complete same-color sets and orange gear ownership.
package achievement

import (
	"context"
	"fmt"
	"strconv"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type PetEquipmentStateItemRepository interface {
	FindByCharacterID(ctx context.Context, characterID int64) ([]*domainitem.Item, error)
}

type PetEquipmentStateCatalog interface {
	GetEquipment(id int) *models.EquiptTemplateTemplate
}

type PetEquipmentStateReader struct {
	petRepo  PetRosterStateRepository
	itemRepo PetEquipmentStateItemRepository
	catalog  PetEquipmentStateCatalog
}

func NewPetEquipmentStateReader(petRepo PetRosterStateRepository, itemRepo PetEquipmentStateItemRepository, catalog PetEquipmentStateCatalog) *PetEquipmentStateReader {
	return &PetEquipmentStateReader{petRepo: petRepo, itemRepo: itemRepo, catalog: catalog}
}

func (r *PetEquipmentStateReader) Name() string {
	return "pet-equipment"
}

func (r *PetEquipmentStateReader) RequirementIDs() []int64 {
	return []int64{85, 86, 87, 88, 513, 644, 645, 646, 647, 648, 649, 650, 651}
}

func (r *PetEquipmentStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.petRepo == nil || r.itemRepo == nil || r.catalog == nil {
		return nil, fmt.Errorf("pet-equipment state dependencies not configured")
	}
	pets, err := r.petRepo.FindByCharacterID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	items, err := r.itemRepo.FindByCharacterID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	itemsByID := make(map[int64]*domainitem.Item, len(items))
	hasOrange := 0
	for _, item := range items {
		if item == nil || item.ID <= 0 || item.CharacterID != characterID || item.ItemType != domainitem.ItemTypeEquipment {
			continue
		}
		itemsByID[item.ID] = item
		template := r.catalog.GetEquipment(item.TemplateID)
		if !isPetEquipmentStateTemplate(template) {
			continue
		}
		color, err := petEquipmentStateColor(item)
		if err != nil {
			return nil, fmt.Errorf("pet equipment %d: %w", item.ID, err)
		}
		if color == 5 {
			hasOrange = 1
		}
	}
	completeColors := make(map[int]struct{})
	maxSublimationElements := make(map[int]int)
	maxSublimationTwentyFive := 0
	maxSublimationFifty := 0
	for _, pet := range pets {
		if pet == nil || pet.ID <= 0 || pet.CharacterID != characterID {
			continue
		}
		color, complete, err := r.completePetEquipmentColor(pet, itemsByID)
		if err != nil {
			return nil, err
		}
		if complete {
			completeColors[color] = struct{}{}
		}
		elements, twentyFive, fifty, fullSet, err := r.petEquipmentSublimation(pet, itemsByID)
		if err != nil {
			return nil, err
		}
		if fullSet {
			for element, count := range elements {
				if count > maxSublimationElements[element] {
					maxSublimationElements[element] = count
				}
			}
			if twentyFive > maxSublimationTwentyFive {
				maxSublimationTwentyFive = twentyFive
			}
			if fifty > maxSublimationFifty {
				maxSublimationFifty = fifty
			}
		}
	}
	proofs := make([]StateProof, 0, 13)
	for requirementID, color := range map[int64]int{85: 1, 86: 2, 87: 3, 88: 4} {
		_, exists := completeColors[color]
		progress := 0
		if exists {
			progress = 1
		}
		proofs = append(proofs, stateProof(requirementID, progress, 1))
	}
	return append(proofs,
		stateProof(513, hasOrange, 1),
		stateProof(644, maxSublimationElements[1], 8),
		stateProof(645, maxSublimationElements[2], 8),
		stateProof(646, maxSublimationElements[3], 8),
		stateProof(647, maxSublimationElements[4], 8),
		stateProof(648, maxSublimationElements[5], 8),
		stateProof(649, maxSublimationElements[6], 8),
		stateProof(650, maxSublimationTwentyFive, 8),
		stateProof(651, maxSublimationFifty, 8),
	), nil
}

func (r *PetEquipmentStateReader) petEquipmentSublimation(pet *domainpet.Pet, itemsByID map[int64]*domainitem.Item) (map[int]int, int, int, bool, error) {
	elements := make(map[int]int)
	twentyFive := 0
	fifty := 0
	group := -1
	seen := make(map[int64]struct{}, domainpet.TotalPetEquipmentSlots)
	for slot := 1; slot <= domainpet.TotalPetEquipmentSlots; slot++ {
		itemID := pet.EquipmentItemID(slot)
		if itemID <= 0 {
			return elements, 0, 0, false, nil
		}
		if _, duplicate := seen[itemID]; duplicate {
			return nil, 0, 0, false, fmt.Errorf("pet %d repeats equipment %d", pet.ID, itemID)
		}
		seen[itemID] = struct{}{}
		item := itemsByID[itemID]
		if item == nil || item.SlotType != domainitem.SlotTypePetEquipped || item.SlotIndex < 0 {
			return nil, 0, 0, false, fmt.Errorf("pet %d references invalid equipment %d", pet.ID, itemID)
		}
		template := r.catalog.GetEquipment(item.TemplateID)
		position, positionOK := exactPetCatalogInt(templatePosition(template))
		if !isPetEquipmentStateTemplate(template) || !positionOK || position != 49+slot {
			return nil, 0, 0, false, fmt.Errorf("pet %d equipment %d has invalid slot template", pet.ID, itemID)
		}
		itemGroup := item.SlotIndex / domainpet.TotalPetEquipmentSlots
		if item.SlotIndex%domainpet.TotalPetEquipmentSlots != slot-1 || itemGroup < 0 || itemGroup >= domainpet.MaxPetEquipmentGroups {
			return nil, 0, 0, false, fmt.Errorf("pet %d equipment %d has invalid persisted slot", pet.ID, itemID)
		}
		if group < 0 {
			group = itemGroup
		} else if group != itemGroup {
			return nil, 0, 0, false, fmt.Errorf("pet %d equipment spans multiple groups", pet.ID)
		}
		state, err := decodeSublimationState(item.Properties["flag"])
		if err != nil {
			return nil, 0, 0, false, fmt.Errorf("pet equipment %d: %w", item.ID, err)
		}
		if state.Element > 0 {
			elements[state.Element]++
		}
		if state.Level >= 25 {
			twentyFive++
		}
		if state.Level >= 50 {
			fifty++
		}
	}
	if persistedGroup, ok := pet.EquipmentGroup(); ok && persistedGroup != group {
		return nil, 0, 0, false, fmt.Errorf("pet %d equipment group mismatch", pet.ID)
	}
	return elements, twentyFive, fifty, true, nil
}

func (r *PetEquipmentStateReader) completePetEquipmentColor(pet *domainpet.Pet, itemsByID map[int64]*domainitem.Item) (int, bool, error) {
	color := 0
	group := -1
	seen := make(map[int64]struct{}, domainpet.TotalPetEquipmentSlots)
	for slot := 1; slot <= domainpet.TotalPetEquipmentSlots; slot++ {
		itemID := pet.EquipmentItemID(slot)
		if itemID <= 0 {
			return 0, false, nil
		}
		if _, duplicate := seen[itemID]; duplicate {
			return 0, false, fmt.Errorf("pet %d repeats equipment %d", pet.ID, itemID)
		}
		seen[itemID] = struct{}{}
		item := itemsByID[itemID]
		if item == nil || item.SlotType != domainitem.SlotTypePetEquipped || item.SlotIndex < 0 {
			return 0, false, fmt.Errorf("pet %d references invalid equipment %d", pet.ID, itemID)
		}
		template := r.catalog.GetEquipment(item.TemplateID)
		position, ok := exactPetCatalogInt(templatePosition(template))
		if !isPetEquipmentStateTemplate(template) || !ok || position != 49+slot {
			return 0, false, fmt.Errorf("pet %d equipment %d has invalid slot template", pet.ID, itemID)
		}
		itemGroup := item.SlotIndex / domainpet.TotalPetEquipmentSlots
		if item.SlotIndex%domainpet.TotalPetEquipmentSlots != slot-1 || itemGroup < 0 || itemGroup >= domainpet.MaxPetEquipmentGroups {
			return 0, false, fmt.Errorf("pet %d equipment %d has invalid persisted slot", pet.ID, itemID)
		}
		if group < 0 {
			group = itemGroup
		} else if group != itemGroup {
			return 0, false, fmt.Errorf("pet %d equipment spans multiple groups", pet.ID)
		}
		itemColor, err := petEquipmentStateColor(item)
		if err != nil {
			return 0, false, fmt.Errorf("pet equipment %d: %w", item.ID, err)
		}
		if color == 0 {
			color = itemColor
		} else if color != itemColor {
			return 0, false, nil
		}
	}
	if persistedGroup, ok := pet.EquipmentGroup(); ok && persistedGroup != group {
		return 0, false, fmt.Errorf("pet %d equipment group mismatch", pet.ID)
	}
	return color, color >= 1 && color <= 4, nil
}

func isPetEquipmentStateTemplate(template *models.EquiptTemplateTemplate) bool {
	if template == nil || template.ID <= 0 {
		return false
	}
	kind, kindOK := exactPetCatalogInt(template.Kind)
	position, positionOK := exactPetCatalogInt(template.Position)
	return kindOK && positionOK && kind == 9 && position >= 50 && position <= 57
}

func templatePosition(template *models.EquiptTemplateTemplate) float64 {
	if template == nil {
		return 0
	}
	return template.Position
}

func petEquipmentStateColor(item *domainitem.Item) (int, error) {
	if item.ColorCode < 0 || item.ColorCode > 5 {
		return 0, fmt.Errorf("invalid color code %d", item.ColorCode)
	}
	if item.ColorCode > 0 {
		return item.ColorCode, nil
	}
	if displayColor, ok := exactPetEquipmentStateInt(item.Properties["color"]); ok && displayColor >= 0 && displayColor <= 4 {
		return domainitem.EquipmentColorCodeFromDisplayColor(displayColor), nil
	}
	if colorCode, ok := exactPetEquipmentStateInt(item.Properties["colorCode"]); ok && colorCode >= 1 && colorCode <= 5 {
		return colorCode, nil
	}
	return 0, fmt.Errorf("missing canonical color")
}

func exactPetEquipmentStateInt(value interface{}) (int, bool) {
	if stringValue, ok := value.(string); ok {
		parsed, err := strconv.Atoi(stringValue)
		return parsed, err == nil
	}
	return stateInt(value)
}
