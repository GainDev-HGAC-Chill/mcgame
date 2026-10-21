package pet

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

const petIllusionLedgerKey = "_pet_illusion_aptitude_v1"

type PetIllusionResult struct {
	Pet               *domainpet.Pet
	Success           bool
	Message           string
	AptitudeType      domainpet.AptitudeType
	PropertyName      string
	Value             int
	ConsumedPetID     int64
	ConsumedLuckItem  int64
	ConsumedLuckSID   int
	ConsumedLuckCount int
	RemainingLuck     int
}

func defaultPetIllusionRoll(rate int) bool {
	return rate >= 100 || rate > 0 && rand.Intn(100) < rate
}

func defaultPetIllusionPercent(minimum, maximum int) int {
	if maximum <= minimum {
		return minimum
	}
	return minimum + rand.Intn(maximum-minimum+1)
}

func (s *Service) PetEat(ctx context.Context, charID, mainPetID, costPetID, luckItemID int64, luckNum int, aptType domainpet.AptitudeType) (*PetIllusionResult, error) {
	if s.gameDataManager == nil || s.itemRepo == nil || mainPetID == costPetID || luckNum < 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	propertyName, ok := petIllusionPropertyName(aptType)
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	mainPet, err := s.getPetAndValidate(ctx, charID, mainPetID)
	if err != nil {
		return nil, err
	}
	costPet, err := s.getPetAndValidate(ctx, charID, costPetID)
	if err != nil {
		return nil, err
	}
	mainTier := domainpet.ColorByGrowRate(mainPet.GrowRate)
	costTier := domainpet.ColorByGrowRate(costPet.GrowRate)
	if !mainPet.IsBound() || mainTier < 3 || costTier < 3 {
		return nil, pkgerrors.ErrInvalidInput
	}

	var luckItem *domainitem.Item
	if luckNum > 0 {
		luckItem, err = s.loadPetFunctionItem(ctx, charID, luckItemID)
		if err != nil {
			return nil, err
		}
		luckTemplate := s.gameDataManager.GetItem(luckItem.TemplateID)
		if luckTemplate == nil || int(luckTemplate.Type) != 507 || int(luckTemplate.UseType) != 4 || int(luckTemplate.PropType) != 7 || luckItem.StackCount < luckNum {
			return nil, pkgerrors.ErrInvalidInput
		}
	} else if luckItemID > 0 {
		return nil, pkgerrors.ErrInvalidInput
	}

	rate := petIllusionSingleRate(mainPet, mainTier) + petIllusionSingleRate(costPet, costTier)
	if petIllusionClassID(mainPet) == petIllusionClassID(costPet) {
		rate += 20
	} else {
		rate /= 2
	}
	rate += luckNum * 20
	if rate > 100 {
		rate = 100
	}

	success := s.illusionRoll(rate)
	value := petIllusionAptitudeValue(mainPet, aptType)
	if success {
		minimum, maximum := 10, 15
		if costTier >= 4 {
			minimum, maximum = 25, 30
		}
		bonus := int(math.Round(float64(petIllusionBaseAptitude(costPet, aptType)) * float64(s.illusionPercent(minimum, maximum)) / 100))
		value, ok = setPetIllusionAptitude(mainPet, aptType, bonus)
		if !ok {
			return nil, pkgerrors.ErrInvalidInput
		}
		mainPet.RecalculateStats()
	}

	remainingLuck, err := s.applyPetIllusionMutation(ctx, mainPet, costPet, luckItem, luckNum, success)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("Ảo hóa thất bại (%d%%)", rate)
	if success {
		message = fmt.Sprintf("Ảo hóa thành công (%d%%)", rate)
	}
	s.logger.Info("Pet illusion completed",
		zap.Int64("character_id", charID),
		zap.Int64("main_pet_id", mainPetID),
		zap.Int64("cost_pet_id", costPetID),
		zap.Int("apt_type", int(aptType)),
		zap.Int("success_rate", rate),
		zap.Bool("success", success))

	result := &PetIllusionResult{
		Pet: mainPet, Success: success, Message: message, AptitudeType: aptType,
		PropertyName: propertyName, Value: value, ConsumedPetID: costPetID,
		ConsumedLuckCount: luckNum, RemainingLuck: remainingLuck,
	}
	if luckItem != nil {
		result.ConsumedLuckItem = luckItem.ID
		result.ConsumedLuckSID = luckItem.CalculateSID()
	}
	return result, nil
}

func (s *Service) applyPetIllusionMutation(ctx context.Context, mainPet, costPet *domainpet.Pet, luckItem *domainitem.Item, luckNum int, updateMain bool) (int, error) {
	luckItemID := int64(0)
	if luckItem != nil {
		luckItemID = luckItem.ID
	}
	if repo, ok := s.petRepo.(domainpet.IllusionRepository); ok {
		return repo.ApplyIllusion(ctx, mainPet, costPet.ID, luckItemID, luckNum, updateMain)
	}

	originalMain := mainPet.Clone()
	if updateMain {
		if err := s.petRepo.Save(ctx, mainPet); err != nil {
			return 0, err
		}
	}
	if err := s.petRepo.Delete(ctx, costPet.ID); err != nil {
		if updateMain {
			_ = s.petRepo.Save(ctx, originalMain)
		}
		return 0, err
	}
	remaining := 0
	if luckItem != nil {
		remaining = luckItem.StackCount - luckNum
		var err error
		if remaining == 0 {
			err = s.itemRepo.Delete(ctx, luckItem.ID)
		} else {
			err = s.itemRepo.UpdateStack(ctx, luckItem.ID, remaining)
		}
		if err != nil {
			_ = s.petRepo.Save(ctx, costPet)
			if updateMain {
				_ = s.petRepo.Save(ctx, originalMain)
			}
			return 0, err
		}
	}
	return remaining, nil
}

func petIllusionSingleRate(p *domainpet.Pet, tier int) int {
	divine := p != nil && p.CreatureData != nil && petMapInt(p.CreatureData["classIds"]) == 10
	if tier >= 4 {
		if divine {
			return 30
		}
		return 15
	}
	if divine {
		return 15
	}
	return 10
}

func petIllusionClassID(p *domainpet.Pet) int {
	if p == nil || p.CreatureData == nil {
		return 0
	}
	return petMapInt(p.CreatureData["classId"])
}

func petMapInt(value interface{}) int {
	parsed, _ := parsePetSkillSlotNumber(value)
	return parsed
}

func petIllusionPropertyName(aptType domainpet.AptitudeType) (string, bool) {
	switch aptType {
	case domainpet.AptTypeStrength:
		return "aptStrengthEx", true
	case domainpet.AptTypeAgility:
		return "aptAgilityEx", true
	case domainpet.AptTypeStamina:
		return "aptStaminaEx", true
	case domainpet.AptTypeIntelligence:
		return "aptIntelligenceEx", true
	case domainpet.AptTypeEnergy:
		return "aptEnergyEx", true
	default:
		return "", false
	}
}

func petIllusionBaseAptitude(p *domainpet.Pet, aptType domainpet.AptitudeType) int {
	switch aptType {
	case domainpet.AptTypeStrength:
		return p.AptStrength
	case domainpet.AptTypeAgility:
		return p.AptAgility
	case domainpet.AptTypeStamina:
		return p.AptStamina
	case domainpet.AptTypeIntelligence:
		return p.AptIntelligence
	case domainpet.AptTypeEnergy:
		return p.AptEnergy
	default:
		return 0
	}
}

func petIllusionAptitudeValue(p *domainpet.Pet, aptType domainpet.AptitudeType) int {
	switch aptType {
	case domainpet.AptTypeStrength:
		return p.AptStrengthEx
	case domainpet.AptTypeAgility:
		return p.AptAgilityEx
	case domainpet.AptTypeStamina:
		return p.AptStaminaEx
	case domainpet.AptTypeIntelligence:
		return p.AptIntelligenceEx
	case domainpet.AptTypeEnergy:
		return p.AptEnergyEx
	default:
		return 0
	}
}

func setPetIllusionAptitude(p *domainpet.Pet, aptType domainpet.AptitudeType, bonus int) (int, bool) {
	if p == nil || bonus < 0 {
		return 0, false
	}
	if p.Property == nil {
		p.Property = make(map[string]interface{})
	}
	ledger, _ := p.Property[petIllusionLedgerKey].(map[string]interface{})
	if ledger == nil {
		ledger = make(map[string]interface{})
	}
	propertyName, ok := petIllusionPropertyName(aptType)
	if !ok {
		return 0, false
	}
	oldBonus := petMapInt(ledger[propertyName])
	current := petIllusionAptitudeValue(p, aptType)
	if oldBonus < 0 || oldBonus > current {
		return 0, false
	}
	next := current - oldBonus + bonus
	switch aptType {
	case domainpet.AptTypeStrength:
		p.AptStrengthEx = next
	case domainpet.AptTypeAgility:
		p.AptAgilityEx = next
	case domainpet.AptTypeStamina:
		p.AptStaminaEx = next
	case domainpet.AptTypeIntelligence:
		p.AptIntelligenceEx = next
	case domainpet.AptTypeEnergy:
		p.AptEnergyEx = next
	}
	ledger[propertyName] = bonus
	p.Property[petIllusionLedgerKey] = ledger
	return next, true
}
