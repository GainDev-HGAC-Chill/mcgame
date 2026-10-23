// Open-sourced by BaoLT

// Replacement Training balance curve and stat-provider integration.
package skill

import (
	"context"
	"math"

	domainchar "mcgame-server/internal/domain/character"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/domain/stats"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type trainingStat int

const (
	trainingHP trainingStat = iota + 1
	trainingPhysicalDefense
	trainingMagicDefense
	trainingSpeed
)

type trainingFamily struct {
	codeName       string
	stat           trainingStat
	tierMultiplier float64
	skillIDs       [10]int
}

type trainingCoefficients struct {
	hp              float64
	physicalDefense float64
	magicDefense    float64
	speed           float64
}

var trainingFamilies = [...]trainingFamily{
	{codeName: "SKILL493201", stat: trainingHP, tierMultiplier: 1, skillIDs: [10]int{3857, 3873, 3874, 3875, 3876, 3877, 3878, 3879, 3880, 3881}},
	{codeName: "SKILL494301", stat: trainingHP, tierMultiplier: 1.5, skillIDs: [10]int{3883, 3886, 3887, 3888, 3889, 3890, 3891, 3892, 3893, 3894}},
	{codeName: "SKILL494401", stat: trainingHP, tierMultiplier: 2.25, skillIDs: [10]int{3884, 3895, 3896, 3897, 3898, 3899, 3900, 3901, 3902, 3903}},
	{codeName: "SKILL504501", stat: trainingPhysicalDefense, tierMultiplier: 1, skillIDs: [10]int{4071, 4072, 4073, 4074, 4075, 4076, 4077, 4078, 4079, 4080}},
	{codeName: "SKILL504601", stat: trainingPhysicalDefense, tierMultiplier: 1.5, skillIDs: [10]int{4081, 4082, 4083, 4084, 4085, 4086, 4087, 4088, 4089, 4090}},
	{codeName: "SKILL504701", stat: trainingPhysicalDefense, tierMultiplier: 2.25, skillIDs: [10]int{4091, 4092, 4093, 4094, 4095, 4096, 4097, 4098, 4099, 4100}},
	{codeName: "SKILL504801", stat: trainingMagicDefense, tierMultiplier: 1, skillIDs: [10]int{4101, 4102, 4103, 4104, 4105, 4106, 4107, 4108, 4109, 4110}},
	{codeName: "SKILL504901", stat: trainingMagicDefense, tierMultiplier: 1.5, skillIDs: [10]int{4111, 4112, 4113, 4114, 4115, 4116, 4117, 4118, 4119, 4120}},
	{codeName: "SKILL505001", stat: trainingMagicDefense, tierMultiplier: 2.25, skillIDs: [10]int{4121, 4122, 4123, 4124, 4125, 4126, 4127, 4128, 4129, 4130}},
	{codeName: "SKILL505101", stat: trainingSpeed, tierMultiplier: 1, skillIDs: [10]int{4131, 4132, 4133, 4134, 4135, 4136, 4137, 4138, 4139, 4140}},
	{codeName: "SKILL505201", stat: trainingSpeed, tierMultiplier: 1.5, skillIDs: [10]int{4141, 4142, 4143, 4144, 4145, 4146, 4147, 4148, 4149, 4150}},
	{codeName: "SKILL505301", stat: trainingSpeed, tierMultiplier: 2.25, skillIDs: [10]int{4151, 4152, 4153, 4154, 4155, 4156, 4157, 4158, 4159, 4160}},
}

var trainingCharacterCurves = map[trainingStat][10]int{
	trainingHP:              {100, 220, 360, 520, 700, 900, 1120, 1360, 1620, 1900},
	trainingPhysicalDefense: {400, 800, 1200, 1600, 2200, 3000, 4000, 5200, 6800, 8800},
	trainingMagicDefense:    {400, 800, 1200, 1600, 2200, 3000, 4000, 5200, 6800, 8800},
	trainingSpeed:           {20, 40, 60, 120, 180, 240, 320, 480, 700, 950},
}

var trainingPetCurves = map[trainingStat][10]int{
	trainingHP:              {50, 110, 180, 260, 350, 450, 560, 680, 810, 950},
	trainingPhysicalDefense: {200, 400, 600, 800, 1100, 1500, 2000, 2600, 3400, 4400},
	trainingMagicDefense:    {200, 400, 600, 800, 1100, 1500, 2000, 2600, 3400, 4400},
	trainingSpeed:           {10, 20, 30, 60, 90, 120, 160, 240, 350, 480},
}

var trainingPetCoefficients = map[int]trainingCoefficients{
	1: {hp: 1.040, physicalDefense: 1.030, magicDefense: 1.088, speed: 1.000},
	2: {hp: 1.026, physicalDefense: 1.074, magicDefense: 0.750, speed: 1.220},
	3: {hp: 1.250, physicalDefense: 1.136, magicDefense: 1.185, speed: 0.750},
	4: {hp: 1.250, physicalDefense: 1.250, magicDefense: 1.210, speed: 1.150},
	5: {hp: 1.200, physicalDefense: 1.188, magicDefense: 1.250, speed: 1.150},
	6: {hp: 0.750, physicalDefense: 0.812, magicDefense: 1.136, speed: 1.000},
}

func (s *Service) AggregateCharacterStatBonuses(ctx context.Context, charID int64) domainchar.EquipmentStatBonuses {
	bonuses := domainchar.NewEquipmentStatBonuses()
	if s == nil || !s.trainingReady || s.charRepo == nil {
		return bonuses
	}
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil || char == nil {
		s.logTrainingFailure("load character", charID, 0, err)
		return bonuses
	}
	class := s.gameData.GetClass(char.ClassID)
	coefficients, ok := characterTrainingCoefficients(class)
	if !ok {
		s.logTrainingFailure("invalid character class", charID, 0, nil)
		return bonuses
	}
	levels := s.highestTrainingLevels(ctx, charID)
	for index, level := range levels {
		if level == 0 {
			continue
		}
		family := trainingFamilies[index]
		value := trainingValue(trainingCharacterCurves[family.stat][level-1], family.tierMultiplier, coefficientForStat(coefficients, family.stat))
		bonuses.AddFlat(characterTrainingProp(family.stat), value)
	}
	return bonuses
}

func (s *Service) AggregatePetBonuses(ctx context.Context, petID int64) (domainpet.EquipmentStatBonuses, domainpet.SkillStatBonuses) {
	equipment := domainpet.NewEquipmentStatBonuses()
	skills := domainpet.NewSkillStatBonuses()
	if s == nil || !s.trainingReady || s.petRepo == nil {
		return equipment, skills
	}
	pet, err := s.petRepo.FindByID(ctx, petID)
	if err != nil || pet == nil {
		s.logTrainingFailure("load pet", 0, petID, err)
		return equipment, skills
	}
	template := s.gameData.GetCreature(pet.TemplateID)
	if template == nil {
		s.logTrainingFailure("missing pet template", pet.CharacterID, petID, nil)
		return equipment, skills
	}
	coefficients, ok := trainingPetCoefficients[int(template.ClassID)]
	if !ok {
		s.logTrainingFailure("invalid pet family", pet.CharacterID, petID, nil)
		return equipment, skills
	}
	levels := s.highestTrainingLevels(ctx, pet.CharacterID)
	for index, level := range levels {
		if level == 0 {
			continue
		}
		family := trainingFamilies[index]
		value := trainingValue(trainingPetCurves[family.stat][level-1], family.tierMultiplier, coefficientForStat(coefficients, family.stat))
		equipment.AddFlat(petTrainingProp(family.stat), value)
	}
	return equipment, skills
}

func (s *Service) highestTrainingLevels(ctx context.Context, charID int64) [len(trainingFamilies)]int {
	var levels [len(trainingFamilies)]int
	learned, err := s.skillRepo.FindByCharacterID(ctx, charID)
	if err != nil {
		s.logTrainingFailure("load learned skills", charID, 0, err)
		return levels
	}
	for _, entry := range learned {
		if entry == nil {
			continue
		}
		template := s.gameData.GetSkill(entry.SkillID)
		if isTrainingSkillTemplate(template) && !validLearnedTrainingEntry(template, entry.Level) {
			s.logTrainingFailure("invalid learned training skill", charID, 0, nil)
			continue
		}
		for index, family := range trainingFamilies {
			level := exactTrainingLevel(template, family)
			if level == 0 {
				continue
			}
			if level > levels[index] {
				levels[index] = level
			}
		}
	}
	return levels
}

func validLearnedTrainingEntry(template *models.SkillTemplate, persistedLevel int) bool {
	for _, family := range trainingFamilies {
		level := exactTrainingLevel(template, family)
		if level > 0 {
			return persistedLevel == level
		}
	}
	return false
}

func validateTrainingCatalog(manager *gamedata.Manager) bool {
	if manager == nil {
		return false
	}
	count := 0
	for _, template := range manager.GetAllSkills() {
		if !isTrainingSkillTemplate(template) {
			continue
		}
		matched := false
		for _, family := range trainingFamilies {
			if exactTrainingLevel(template, family) > 0 {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
		count++
	}
	if count != len(trainingFamilies)*10 {
		return false
	}
	for _, family := range trainingFamilies {
		for _, skillID := range family.skillIDs {
			if exactTrainingLevel(manager.GetSkill(skillID), family) == 0 {
				return false
			}
		}
	}
	return true
}

func exactTrainingLevel(template *models.SkillTemplate, family trainingFamily) int {
	if template == nil || template.CodeName != family.codeName || int(template.UseEnv) != 3 || int(template.Kind) != 5 || template.ReqClass != "|1|2|3|4|5|6|" {
		return 0
	}
	level := int(template.Level)
	if level < 1 || level > len(family.skillIDs) || int(template.ID) != family.skillIDs[level-1] {
		return 0
	}
	return level
}

func characterTrainingCoefficients(class *models.ClassTemplate) (trainingCoefficients, bool) {
	if class == nil || class.ID < 1 || class.ID > 6 {
		return trainingCoefficients{}, false
	}
	return trainingCoefficients{
		hp:              clampTrainingCoefficient(class.AptStamina / 2000),
		physicalDefense: clampTrainingCoefficient((stats.CoeffDefenseFromStamina*class.AptStamina + stats.CoeffDefenseFromAgility*class.AptAgility) / ((stats.CoeffDefenseFromStamina + stats.CoeffDefenseFromAgility) * 2000)),
		magicDefense:    clampTrainingCoefficient((stats.CoeffMagicDefenseFromInt*class.AptIntelligence + stats.CoeffMagicDefenseFromEnergy*class.AptEnergy) / ((stats.CoeffMagicDefenseFromInt + stats.CoeffMagicDefenseFromEnergy) * 2000)),
		speed:           clampTrainingCoefficient(class.AptAgility / 2000),
	}, true
}

func clampTrainingCoefficient(value float64) float64 {
	return math.Max(0.75, math.Min(1.25, value))
}

func coefficientForStat(coefficients trainingCoefficients, stat trainingStat) float64 {
	switch stat {
	case trainingHP:
		return coefficients.hp
	case trainingPhysicalDefense:
		return coefficients.physicalDefense
	case trainingMagicDefense:
		return coefficients.magicDefense
	case trainingSpeed:
		return coefficients.speed
	default:
		return 0
	}
}

func trainingValue(base int, tierMultiplier, coefficient float64) int {
	return int(math.Round(float64(base) * tierMultiplier * coefficient))
}

func characterTrainingProp(stat trainingStat) int {
	switch stat {
	case trainingHP:
		return domainchar.PropMaxHP
	case trainingPhysicalDefense:
		return domainchar.PropDefense
	case trainingMagicDefense:
		return domainchar.PropMagicDef
	case trainingSpeed:
		return domainchar.PropSpeed
	default:
		return 0
	}
}

func petTrainingProp(stat trainingStat) int {
	switch stat {
	case trainingHP:
		return 1
	case trainingPhysicalDefense:
		return 6
	case trainingMagicDefense:
		return 7
	case trainingSpeed:
		return 11
	default:
		return 0
	}
}

func (s *Service) logTrainingFailure(reason string, charID, petID int64, err error) {
	if s == nil || s.logger == nil {
		return
	}
	fields := []zap.Field{zap.String("reason", reason)}
	if charID > 0 {
		fields = append(fields, zap.Int64("character_id", charID))
	}
	if petID > 0 {
		fields = append(fields, zap.Int64("pet_id", petID))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	s.logger.Warn("Training bonus skipped", fields...)
}
