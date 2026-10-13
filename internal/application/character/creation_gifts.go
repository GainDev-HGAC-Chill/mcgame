package character

import (
	"fmt"
	"math"

	appitem "mcgame-server/internal/application/item"
	apppet "mcgame-server/internal/application/pet"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

func (s *Service) buildCreationPackage(char *character.Character, defaults character.NewCharacterDefaults) (*character.CreationPackage, error) {
	if (len(defaults.Items) > 0 || len(defaults.Pets) > 0) && s.gameData == nil {
		return nil, fmt.Errorf("starter catalogs are unavailable")
	}
	hash, err := defaults.SnapshotHash()
	if err != nil {
		return nil, err
	}
	bundle := &character.CreationPackage{Defaults: defaults, Hash: hash, Items: []*item.Item{}, Pets: []*domainpet.Pet{}}
	used := map[item.SlotType]int{}
	limits := map[item.SlotType]int{item.SlotTypeBag: char.BagSlots, item.SlotTypeQuestBag: item.DefaultQuestBagSlots, item.SlotTypePetItemBag: item.DefaultPetItemBagSlots}
	for index, entry := range defaults.Items {
		prototype, stackMax, err := s.creationItem(entry)
		if err != nil {
			return nil, fmt.Errorf("starter item %d: %w", index+1, err)
		}
		remaining := entry.Quantity
		for _, existing := range bundle.Items {
			if existing.TemplateID != prototype.TemplateID || existing.ItemType != prototype.ItemType || existing.SlotType != prototype.SlotType || existing.IsBound != prototype.IsBound || existing.ColorCode != prototype.ColorCode {
				continue
			}
			added := min(remaining, stackMax-existing.StackCount)
			existing.StackCount += added
			remaining -= added
		}
		for remaining > 0 {
			if used[prototype.SlotType] >= limits[prototype.SlotType] {
				return nil, fmt.Errorf("starter item package exceeds bag capacity for slot type %d", prototype.SlotType)
			}
			it, _, err := s.creationItem(entry)
			if err != nil {
				return nil, err
			}
			it.SlotIndex = used[it.SlotType]
			it.StackCount = min(remaining, stackMax)
			remaining -= it.StackCount
			used[it.SlotType]++
			bundle.Items = append(bundle.Items, it)
		}
	}
	for index, entry := range defaults.Pets {
		template := s.gameData.GetCreature(entry.TemplateID)
		if template == nil || template.ClassID != 2 {
			return nil, fmt.Errorf("starter pet %d must reference a pet creature (class_id=2)", index+1)
		}
		if template.IsBind > 0 && !entry.Bound {
			return nil, fmt.Errorf("starter pet %d requires binding", index+1)
		}
		if len(bundle.Pets)+entry.Quantity > char.PetMaxNum {
			return nil, fmt.Errorf("starter pets exceed carried pet capacity %d", char.PetMaxNum)
		}
		for count := 0; count < entry.Quantity; count++ {
			bundle.Pets = append(bundle.Pets, apppet.PrepareCreationPet(s.gameData, template, entry.Growth, entry.Bound))
		}
	}
	return bundle, nil
}

func (s *Service) creationItem(entry character.StarterItem) (*item.Item, int, error) {
	var bindType, stack, single float64
	var equipment *models.EquiptTemplateTemplate
	it := item.NewItem(0, entry.TemplateID, item.ItemTypeConsumable, item.SlotTypeBag, 0)
	it.IsBound = entry.Bound
	it.ColorCode = entry.Color
	if entry.TemplateTableID == 19 {
		equipment = s.gameData.GetEquipment(entry.TemplateID)
		if equipment == nil {
			return nil, 0, fmt.Errorf("equipment template %d not found", entry.TemplateID)
		}
		bindType, stack, single = equipment.BindType, equipment.StackMax, equipment.SingleFlag
		it.ItemType = item.ItemTypeEquipment
		if equipment.Kind == 9 {
			it.SlotType = item.SlotTypePetItemBag
		}
		if entry.Color == 0 {
			it.ColorCode = item.NormalizeEquipmentColorCode(int(equipment.ColorCode))
			if it.ColorCode == 0 {
				it.ColorCode = item.NormalizeEquipmentColorCode(int(equipment.Color))
			}
		}
		appitem.PrepareCreationEquipment(it, equipment)
	} else {
		template := s.gameData.GetItem(entry.TemplateID)
		if template == nil {
			return nil, 0, fmt.Errorf("item template %d not found", entry.TemplateID)
		}
		bindType, stack, single = template.BindType, template.StackMax, template.SingleFlag
		if template.Kind == 6 {
			it.ItemType = item.ItemTypeMaterial
			if entry.Color > 4 {
				return nil, 0, fmt.Errorf("material color must be 0..4")
			}
		} else if entry.Color != 0 {
			return nil, 0, fmt.Errorf("non-material item color must be 0")
		}
		switch int(template.Type) {
		case 508, 509, 550:
			it.SlotType = item.SlotTypeQuestBag
		case 507, 512, 513:
			it.SlotType = item.SlotTypePetItemBag
		}
	}
	if bindType > 0 && !entry.Bound {
		return nil, 0, fmt.Errorf("template requires binding")
	}
	if math.IsNaN(stack) || math.IsInf(stack, 0) || math.Trunc(stack) != stack || stack > math.MaxInt32 {
		return nil, 0, fmt.Errorf("invalid catalog stack maximum")
	}
	stackMax := max(1, int(stack))
	if single > 0 || equipment != nil {
		stackMax = 1
	}
	return it, stackMax, nil
}
