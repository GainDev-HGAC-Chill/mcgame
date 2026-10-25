// Open-sourced by BaoLT

// Pet-roster state reader proves collection, quality, level, and learned-skill milestones.
package achievement

import (
	"context"
	"fmt"
	"math"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type PetRosterStateRepository interface {
	FindByCharacterID(ctx context.Context, characterID int64) ([]*domainpet.Pet, error)
}

type PetRosterCatalog interface {
	GetCreature(id int) *models.CreatureTemplate
}

type PetRosterStateReader struct {
	repo    PetRosterStateRepository
	catalog PetRosterCatalog
}

var petCollectionRequirements = map[int64]struct {
	templateID int
	name       string
}{
	307: {71, "Y Tá MM"},
	308: {87, "Tiểu Lương Tử"},
	309: {735, "Kim Hổ"},
	310: {455, "Kim Ngưu"},
	311: {488, "Dứa Mật"},
	312: {876, "Nhím Xanh"},
	313: {447, "Tinh Linh Hộ Thú"},
	314: {776, "Momo"},
	315: {721, "Hoàng Tử Thiên Sứ"},
	316: {19, "Thiên Sứ Mít Ướt"},
	317: {722, "Công Chúa Ác Ma"},
	318: {486, "Đinh Long"},
	319: {487, "Thần Long Viễn Cổ"},
	343: {903, "Thỏ Baby"},
}

var petOrangeGuardianRequirements = map[int64]struct {
	templateID int
	name       string
}{
	765: {2290, "Bạch Hổ Cuồng Nộ"},
	766: {2281, "Huyền Minh Linh Vũ"},
	767: {2279, "Kim Tước"},
	768: {2277, "Long Tiểu Thanh"},
}

func NewPetRosterStateReader(repo PetRosterStateRepository, catalog PetRosterCatalog) *PetRosterStateReader {
	return &PetRosterStateReader{repo: repo, catalog: catalog}
}

func (r *PetRosterStateReader) Name() string {
	return "pet-roster"
}

func (r *PetRosterStateReader) RequirementIDs() []int64 {
	ids := requirementIDRange(59, 66)
	ids = append(ids, requirementIDRange(71, 79)...)
	ids = append(ids, 81, 82)
	ids = append(ids, requirementIDRange(89, 91)...)
	ids = append(ids, requirementIDRange(97, 99)...)
	ids = append(ids, requirementIDRange(307, 319)...)
	ids = append(ids, 343)
	return append(ids, requirementIDRange(765, 768)...)
}

func (r *PetRosterStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("pet-roster state dependencies not configured")
	}
	if err := validatePetCollectionCatalog(r.catalog); err != nil {
		return nil, err
	}
	pets, err := r.repo.FindByCharacterID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	templateIDs := make(map[int]struct{})
	classTemplates := make(map[int]map[int]struct{})
	classColors := make(map[int]map[int]struct{})
	allColors := make(map[int]struct{})
	orangeDivineByClass := make(map[int]int)
	ownedRequired := make(map[int]struct{})
	orangeOwned := make(map[int]struct{})
	maxDivineLevel := 0
	maxLevelEightySkillCount := 0
	hasOrange := 0
	hasOrangeDivine := 0
	hasOrangeChicken := 0
	hasMaxPurple := 0
	for _, pet := range pets {
		if pet == nil || pet.CharacterID != characterID || pet.ID <= 0 || pet.TemplateID <= 0 {
			continue
		}
		template := r.catalog.GetCreature(pet.TemplateID)
		if template == nil || template.ID != int64(pet.TemplateID) {
			return nil, fmt.Errorf("pet %d has unknown creature template %d", pet.ID, pet.TemplateID)
		}
		classID, ok := exactPetCatalogInt(template.ClassID)
		if !ok || classID < 1 || classID > 7 {
			return nil, fmt.Errorf("pet template %d has invalid class %v", pet.TemplateID, template.ClassID)
		}
		classIDs, ok := exactPetCatalogInt(template.ClassIds)
		if !ok {
			return nil, fmt.Errorf("pet template %d has invalid extended class %v", pet.TemplateID, template.ClassIds)
		}
		color := domainpet.ColorByGrowRate(pet.GrowRate)
		templateIDs[pet.TemplateID] = struct{}{}
		if classTemplates[classID] == nil {
			classTemplates[classID] = make(map[int]struct{})
		}
		classTemplates[classID][pet.TemplateID] = struct{}{}
		if classColors[classID] == nil {
			classColors[classID] = make(map[int]struct{})
		}
		classColors[classID][color] = struct{}{}
		allColors[color] = struct{}{}
		if color == 4 {
			hasOrange = 1
			orangeOwned[pet.TemplateID] = struct{}{}
			if classIDs == 10 {
				hasOrangeDivine = 1
				orangeDivineByClass[classID]++
			}
			if pet.TemplateID == 2 {
				hasOrangeChicken = 1
			}
		}
		if color == 3 && math.Abs(pet.GrowRate-2.0) < 0.000001 {
			hasMaxPurple = 1
		}
		if classIDs == 10 && pet.Level > maxDivineLevel {
			maxDivineLevel = pet.Level
		}
		if pet.Level >= 80 && len(pet.LearnedSkillSlots()) > maxLevelEightySkillCount {
			maxLevelEightySkillCount = len(pet.LearnedSkillSlots())
		}
		ownedRequired[pet.TemplateID] = struct{}{}
	}
	classesWithFiveColors := 0
	for classID := 1; classID <= 6; classID++ {
		if len(classColors[classID]) == 5 {
			classesWithFiveColors++
		}
	}
	proofs := []StateProof{
		stateProof(59, len(templateIDs), 30),
		stateProof(60, len(classTemplates[6]), 10),
		stateProof(61, len(classTemplates[4]), 10),
		stateProof(62, len(classTemplates[1]), 15),
		stateProof(63, len(classTemplates[2]), 15),
		stateProof(64, len(classTemplates[3]), 15),
		stateProof(65, len(classTemplates[5]), 15),
		stateProof(66, len(allColors), 5),
		stateProof(71, classesWithFiveColors, 6),
		stateProof(72, hasOrange, 1),
		stateProof(73, hasOrangeDivine, 1),
		stateProof(74, orangeDivineByClass[6], 1),
		stateProof(75, orangeDivineByClass[4], 1),
		stateProof(76, orangeDivineByClass[1], 1),
		stateProof(77, orangeDivineByClass[2], 1),
		stateProof(78, orangeDivineByClass[3], 1),
		stateProof(79, orangeDivineByClass[5], 1),
		stateProof(81, hasOrangeChicken, 1),
		stateProof(82, hasMaxPurple, 1),
		stateProof(89, maxDivineLevel, 40),
		stateProof(90, maxDivineLevel, 80),
		stateProof(91, maxDivineLevel, 140),
		stateProof(97, maxLevelEightySkillCount, 5),
		stateProof(98, maxLevelEightySkillCount, 10),
		stateProof(99, maxLevelEightySkillCount, 15),
	}
	for requirementID := int64(307); requirementID <= 319; requirementID++ {
		definition := petCollectionRequirements[requirementID]
		_, owned := ownedRequired[definition.templateID]
		progress := 0
		if owned {
			progress = 1
		}
		proofs = append(proofs, stateProof(requirementID, progress, 1))
	}
	definition := petCollectionRequirements[343]
	_, owned := ownedRequired[definition.templateID]
	progress := 0
	if owned {
		progress = 1
	}
	proofs = append(proofs, stateProof(343, progress, 1))
	for requirementID := int64(765); requirementID <= 768; requirementID++ {
		definition := petOrangeGuardianRequirements[requirementID]
		_, ownedOrange := orangeOwned[definition.templateID]
		progress = 0
		if ownedOrange {
			progress = 1
		}
		proofs = append(proofs, stateProof(requirementID, progress, 1))
	}
	return proofs, nil
}

func validatePetCollectionCatalog(catalog PetRosterCatalog) error {
	for requirementID, definition := range petCollectionRequirements {
		template := catalog.GetCreature(definition.templateID)
		if template == nil || template.ID != int64(definition.templateID) || template.Name != definition.name {
			return fmt.Errorf("pet collection requirement %d catalog identity drift", requirementID)
		}
	}
	for requirementID, definition := range petOrangeGuardianRequirements {
		template := catalog.GetCreature(definition.templateID)
		if template == nil || template.ID != int64(definition.templateID) || template.Name != definition.name {
			return fmt.Errorf("pet orange guardian requirement %d catalog identity drift", requirementID)
		}
	}
	chicken := catalog.GetCreature(2)
	if chicken == nil || chicken.ID != 2 || chicken.Name != "Thất Vị Kê" {
		return fmt.Errorf("pet collection requirement 81 catalog identity drift")
	}
	return nil
}

func exactPetCatalogInt(value float64) (int, bool) {
	result := int(value)
	return result, float64(result) == value
}
