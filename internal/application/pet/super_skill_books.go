// Pet Super-skill books replace exact level-three prerequisite trios.
package pet

import (
	"sort"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type petSuperSkillBookSpec struct {
	BookTemplateID int
	Family         string
	PropType       int
	ReqClass       string
	Prerequisites  [3]int
	SuperChain     [3]int
}

type PetSkillSlotUpdate struct {
	Slot    int
	SlotKey string
	SkillID int
}

var petSuperSkillBooks = map[int]petSuperSkillBookSpec{
	3216: {BookTemplateID: 3216, Family: "Thụ Bì Ngoại Sáo", PropType: 237, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2830, 3345, 5156}, SuperChain: [3]int{5527, 5528, 5529}},
	3217: {BookTemplateID: 3217, Family: "Trí Tuệ Thiên Nhiên", PropType: 238, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2749, 3348, 5159}, SuperChain: [3]int{5530, 5531, 5532}},
	3218: {BookTemplateID: 3218, Family: "Thú Tính Tỉnh Ngộ", PropType: 239, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2743, 3351, 5162}, SuperChain: [3]int{5533, 5534, 5535}},
	3219: {BookTemplateID: 3219, Family: "Cổ Long Chi Hồn", PropType: 240, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2821, 3354, 5165}, SuperChain: [3]int{5536, 5537, 5538}},
	3220: {BookTemplateID: 3220, Family: "Ngạnh Xác Hóa", PropType: 241, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2699, 3357, 5168}, SuperChain: [3]int{5539, 5540, 5541}},
	3221: {BookTemplateID: 3221, Family: "Vẻ Ngoài Yếu Ớt", PropType: 242, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2842, 3360, 5171}, SuperChain: [3]int{5542, 5543, 5544}},
	3222: {BookTemplateID: 3222, Family: "Tinh Thần Tập Trung", PropType: 243, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{3329, 3363, 5174}, SuperChain: [3]int{5545, 5546, 5547}},
	3223: {BookTemplateID: 3223, Family: "Trực Giác Dã Thú", PropType: 244, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2746, 3366, 5177}, SuperChain: [3]int{5548, 5549, 5550}},
	3224: {BookTemplateID: 3224, Family: "Báo Chi Tấn Tiệp", PropType: 245, ReqClass: "|1|2|3|4|5|6|", Prerequisites: [3]int{2748, 3369, 5180}, SuperChain: [3]int{5551, 5552, 5553}},
	3225: {BookTemplateID: 3225, Family: "Vỏ Ngoài Phóng Điện", PropType: 246, ReqClass: "|1|2|3|4|5|", Prerequisites: [3]int{2711, 3372, 5183}, SuperChain: [3]int{5554, 5555, 5556}},
	3226: {BookTemplateID: 3226, Family: "Long Lân Chi Giáp", PropType: 247, ReqClass: "|1|2|3|4|5|", Prerequisites: [3]int{2818, 3375, 5186}, SuperChain: [3]int{5557, 5558, 5559}},
}

func (s *Service) validatePetSuperSkillBookCatalog(book *models.ItemTemplateTemplate, spec petSuperSkillBookSpec) *models.SkillTemplate {
	if s == nil || s.gameDataManager == nil || book == nil || book.GetID() != spec.BookTemplateID {
		return nil
	}
	if book.Name != spec.Family+" Siêu Cấp" || !isPetSkillBookTemplate(book) || int(book.PropType) != spec.PropType || int(book.ProplNum) != 3 || book.ReqClass != spec.ReqClass {
		return nil
	}

	prerequisiteNames := [3]string{spec.Family, spec.Family + " Trung Cấp", spec.Family + " Cao Cấp"}
	for index, skillID := range spec.Prerequisites {
		skill := s.gameDataManager.GetSkill(skillID)
		if !validPetSuperCatalogSkill(skill, prerequisiteNames[index], 3) {
			return nil
		}
	}
	var superCodeName string
	for index, skillID := range spec.SuperChain {
		skill := s.gameDataManager.GetSkill(skillID)
		if !validPetSuperCatalogSkill(skill, spec.Family+" Siêu Cấp", index+1) {
			return nil
		}
		if index == 0 {
			superCodeName = skill.CodeName
		}
		if superCodeName == "" || skill.CodeName != superCodeName {
			return nil
		}
	}
	return s.gameDataManager.GetSkill(spec.SuperChain[0])
}

func validPetSuperCatalogSkill(skill *models.SkillTemplate, name string, level int) bool {
	return skill != nil && skill.Name == name && int(skill.Level) == level && isPetSkillUseEnvironment(int(skill.UseEnv))
}

func (s *Service) petCanUseSuperSkillBook(p *domainpet.Pet, book *models.ItemTemplateTemplate) bool {
	if p == nil || book == nil || s.petSkillQualityLevel(p) < int(book.ProplNum) {
		return false
	}
	allowedClasses := book.ParsePipeDelimitedInts(book.ReqClass)
	if len(allowedClasses) == 0 {
		return true
	}
	classID := s.petCreatureClassID(p)
	for _, allowed := range allowedClasses {
		if allowed == classID {
			return true
		}
	}
	return false
}

func (s *Service) petCreatureClassID(p *domainpet.Pet) int {
	if p == nil {
		return 0
	}
	if p.CreatureData != nil {
		if classID, ok := parsePetSkillSlotNumber(p.CreatureData["classId"]); ok && classID > 0 {
			return classID
		}
		if classID, ok := parsePetSkillSlotNumber(p.CreatureData["class_id"]); ok && classID > 0 {
			return classID
		}
	}
	if s.gameDataManager != nil {
		if template := s.gameDataManager.GetCreature(p.TemplateID); template != nil {
			return int(template.ClassID)
		}
	}
	return 0
}

func petSuperSkillSlotUpdates(p *domainpet.Pet, spec petSuperSkillBookSpec) ([]PetSkillSlotUpdate, bool) {
	if p == nil {
		return nil, false
	}
	prerequisiteSlots := make(map[int][]int, len(spec.Prerequisites))
	for slot := 1; slot <= domainpet.TotalPetSkillSlots; slot++ {
		skillID := petSkillSlotValue(p, slot)
		if petSuperChainContains(spec, skillID) {
			return nil, false
		}
		for _, prerequisiteID := range spec.Prerequisites {
			if skillID == prerequisiteID {
				prerequisiteSlots[prerequisiteID] = append(prerequisiteSlots[prerequisiteID], slot)
			}
		}
	}

	slots := make([]int, 0, len(spec.Prerequisites))
	for _, prerequisiteID := range spec.Prerequisites {
		matched := prerequisiteSlots[prerequisiteID]
		if len(matched) != 1 {
			return nil, false
		}
		slots = append(slots, matched[0])
	}
	sort.Ints(slots)
	updates := make([]PetSkillSlotUpdate, 0, len(slots))
	for index, slot := range slots {
		skillID := 0
		if index == 0 {
			skillID = spec.SuperChain[0]
		}
		updates = append(updates, PetSkillSlotUpdate{Slot: slot, SlotKey: petSkillSlotKey(slot), SkillID: skillID})
	}
	return updates, true
}

func petSuperChainContains(spec petSuperSkillBookSpec, skillID int) bool {
	for _, candidate := range spec.SuperChain {
		if skillID == candidate {
			return true
		}
	}
	return false
}

func (s *Service) petHasSuperSkillForOrdinary(p *domainpet.Pet, learnedSkill *models.SkillTemplate) bool {
	if p == nil || learnedSkill == nil {
		return false
	}
	for _, spec := range petSuperSkillBooks {
		if !petSkillNameBelongsToOrdinaryFamily(learnedSkill.Name, spec.Family) {
			continue
		}
		for slot := 1; slot <= domainpet.TotalPetSkillSlots; slot++ {
			if petSuperChainContains(spec, petSkillSlotValue(p, slot)) {
				return true
			}
		}
	}
	return false
}

func petSkillNameBelongsToOrdinaryFamily(name string, family string) bool {
	if name == family {
		return true
	}
	return name == family+" Trung Cấp" || name == family+" Cao Cấp"
}
