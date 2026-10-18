package pet

import (
	"strings"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

// These are the innate level-1 skills for the five named pet templates.
// Other pets continue to use TBL_CREATURE_SKILL without any override.
var petNamedBaseSkillNames = map[string][]string{
	"Bạch Hổ Cuồng Nộ": {
		"Hổ Khiếu Sơn Lâm",
		"Chiến Đấu Máu Lửa",
		"Thú Vương Đột Kích",
		"Đột Kích Bí Mật",
	},
	"Kim Tước": {
		"Thánh Quang Thanh Tẩy",
		"Kiên Bàn Tâm",
		"Mãnh Lực Trùng Kích",
	},
	"Long Tiểu Thanh": {
		"Thanh Long Cửu Biến",
		"Uy Nghiêm Của Rồng",
		"Hủy Diệt Phún Xạ",
	},
	"Võ Sĩ Shiba": {
		"Trảm Linh",
		"Tàn Tâm",
	},
	"Đại Thánh Chí Tôn": {
		"Di Hình Hoán Vị",
		"Vô Song Loạn Vũ",
		"Bách Biến Trùng Kích",
	},
}

// seedNamedPetDefaultSkills handles only the five special pet templates above.
// It resolves IDs from TBL_SKILL at level 1, instead of hardcoding IDs or
// requiring every mapping to already exist in TBL_CREATURE_SKILL.
// It returns false when the pet is not configured or any skill is missing;
// the caller then uses the existing TBL_CREATURE_SKILL behavior unchanged.
func (s *Service) seedNamedPetDefaultSkills(p *domainpet.Pet) bool {
	if s == nil || p == nil || s.gameDataManager == nil {
		return false
	}
	template := s.gameDataManager.GetCreature(int(p.TemplateID))
	if template == nil {
		return false
	}
	// Match Vietnamese creature names regardless of letter case or outer spaces.
	var required []string
	for petName, skillNames := range petNamedBaseSkillNames {
		if strings.EqualFold(strings.TrimSpace(template.Name), petName) {
			required = skillNames
			break
		}
	}
	if len(required) == 0 {
		return false
	}

	wanted := make(map[string]bool, len(required))
	for _, name := range required {
		wanted[name] = true
	}

	matched := make(map[string]*models.SkillTemplate, len(required))
	for _, skill := range s.gameDataManager.GetAllSkills() {
		if skill == nil || skill.Level != 1 {
			continue
		}
		name := strings.TrimSpace(skill.Name)
		if !wanted[name] {
			continue
		}
		if current := matched[name]; current == nil || skill.ID < current.ID {
			matched[name] = skill
		}
	}

	missing := make([]string, 0)
	for _, name := range required {
		if matched[name] == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		if s.logger != nil {
			s.logger.Error("Pet base skill names not found in TBL_SKILL; using original creature skill mapping",
				zap.Int("creature_id", p.TemplateID),
				zap.String("creature_name", template.Name),
				zap.Strings("missing_skill_names", missing))
		}
		return false
	}

	// This function is called only while creating a NEW pet, before saving it.
	// Requested innate skills take priority, in the order declared above.
	usedIDs := make(map[int]bool, petSkillDefaultOpenSlots)
	usedNames := make(map[string]bool, petSkillDefaultOpenSlots)
	usedCodes := make(map[string]bool, petSkillDefaultOpenSlots)
	slot := 1
	for _, name := range required {
		skill := matched[name]
		skillID := int(skill.ID)
		if usedIDs[skillID] {
			continue
		}
		setPetSkillSlotValue(p, slot, skillID)
		usedIDs[skillID] = true
		usedNames[strings.TrimSpace(skill.Name)] = true
		if skill.CodeName != "" {
			usedCodes[skill.CodeName] = true
		}
		slot++
	}

	// Preserve any other template-defined skills in the remaining default slots.
	for _, creatureSkill := range s.gameDataManager.GetCreatureSkills(int(p.TemplateID)) {
		if slot > petSkillDefaultOpenSlots {
			break
		}
		skillID := int(creatureSkill.Sid)
		if skillID <= 0 || usedIDs[skillID] {
			continue
		}
		if skill := s.gameDataManager.GetSkill(skillID); skill != nil {
			if usedNames[strings.TrimSpace(skill.Name)] ||
				(skill.CodeName != "" && usedCodes[skill.CodeName]) {
				continue
			}
			usedNames[strings.TrimSpace(skill.Name)] = true
			if skill.CodeName != "" {
				usedCodes[skill.CodeName] = true
			}
		}
		setPetSkillSlotValue(p, slot, skillID)
		usedIDs[skillID] = true
		slot++
	}
	return true
}
