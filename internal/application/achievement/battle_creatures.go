// Open-sourced by BaoLT

// Battle creature achievement matching uses exact catalog IDs and names only.
package achievement

import (
	"fmt"
)

type battleCreatureRequirement struct {
	requirementID int64
	achievementID int64
	creatures     map[int]string
}

var battleCreatureRequirements = []battleCreatureRequirement{
	{190, 5001, map[int]string{144: "Tà Thần"}},
	{191, 5002, map[int]string{142: "Tử Thần"}},
	{192, 5003, map[int]string{141: "Chiến Thần"}},
	{193, 5004, map[int]string{143: "Ma Thần"}},
	{194, 5005, map[int]string{144: "Tà Thần"}},
	{195, 5005, map[int]string{142: "Tử Thần"}},
	{196, 5005, map[int]string{141: "Chiến Thần"}},
	{197, 5005, map[int]string{143: "Ma Thần"}},
	{198, 5006, map[int]string{592: "Thạch Yêu"}},
	{199, 5007, map[int]string{127: "Bách Thảo Tinh"}},
	{200, 5008, map[int]string{130: "Phủ Ma"}},
	{201, 5009, map[int]string{131: "Ma Chiến"}},
	{202, 5010, map[int]string{592: "Thạch Yêu"}},
	{203, 5010, map[int]string{127: "Bách Thảo Tinh"}},
	{204, 5010, map[int]string{130: "Phủ Ma"}},
	{205, 5010, map[int]string{131: "Ma Chiến"}},
	{206, 5011, map[int]string{542: "Ác Thần"}},
	{207, 5012, map[int]string{536: "Quỷ Vương"}},
	{209, 5014, map[int]string{786: "Atula Vương"}},
	{210, 5015, map[int]string{806: "Solomon"}},
	{211, 5016, map[int]string{753: "Avatar"}},
	{231, 5034, map[int]string{22: "Thủ Lĩnh Hải Tặc"}},
	{232, 5035, map[int]string{489: "Ẩn Phong Long"}},
	{233, 5035, map[int]string{490: "Hám Địa Long"}},
	{234, 5035, map[int]string{491: "Đinh Thủy Long"}},
	{235, 5035, map[int]string{492: "Tinh Hoả Long"}},
	{236, 5035, map[int]string{493: "Quang Dực Long"}},
	{237, 5035, map[int]string{494: "Ám Dạ Long"}},
	{238, 5035, map[int]string{495: "Thánh Long"}},
	{240, 5037, map[int]string{277: "Nữ Yêu Ngàn Năm"}},
	{241, 5037, map[int]string{279: "Lão Quái Bạch Cốt"}},
	{242, 5037, map[int]string{280: "Chướng Thi Vương"}},
	{243, 5037, map[int]string{278: "Hung Linh Ngàn Năm"}},
	{326, 5013, map[int]string{542: "Ác Thần"}},
	{514, 6154, map[int]string{2012: "Mehdi Urboss【Sổ Tay Ma Thú】"}},
	{526, 6166, map[int]string{1272: "Ma Chiến【Kiếp Nạn Vô Ưu - Ma Chiến】"}},
	{527, 6167, map[int]string{1274: "Quỷ Vương【Kiếp Nạn Vô Ưu - Quỷ Vương】"}},
	{528, 6168, map[int]string{1284: "Chu Ma Vương【Kiếp Nạn Vô Ưu - Chu Ma】"}},
	{529, 6169, map[int]string{1269: "Ma Hồn【Kiếp Nạn Vô Ưu - Ma Hồn】"}},
	{530, 6170, map[int]string{1303: "Mehdi Em【Kiếp Nạn Vô Ưu - Mehdi 1】", 1304: "Mehdi【Kiếp Nạn Vô Ưu - Mehdi 2】"}},
	{531, 6171, map[int]string{1300: "Hydra【Kiếp Nạn Vô Ưu - Atula】"}},
	{532, 6172, map[int]string{1306: "Atula Vương【Kiếp Nạn Vô Ưu - Atula】"}},
	{533, 6173, map[int]string{1307: "Avatar【Kiếp Nạn Vô Ưu - Avatar】"}},
	{534, 6174, map[int]string{1312: "Solomon【Kiếp Nạn Vô Ưu - Solomon】"}},
	{570, 6208, map[int]string{1803: "Thánh Kiếm Orlando【Boss 1-2 VDRL】"}},
	{571, 6209, map[int]string{1846: "Deeven"}},
	{572, 6210, map[int]string{1822: "Pollux【Boss VDRL3 - VL】"}},
	{573, 6210, map[int]string{1821: "Catos【Boss VDRL3 - MP】"}},
	{574, 6211, map[int]string{1814: "Thanatos【Quái Chính 4 VDRL 1】"}},
	{575, 6212, map[int]string{1833: " Tà Thần Khakass【Boss VDRL 5】"}},
}

func (s *Service) defeatedCreatureRequirements(defeatedCreatureIDs []int) ([]requirementTarget, error) {
	if len(defeatedCreatureIDs) == 0 {
		return nil, nil
	}
	if s == nil || s.gameDataManager == nil {
		return nil, fmt.Errorf("achievement creature catalog not configured")
	}
	defeated := make(map[int]struct{}, len(defeatedCreatureIDs))
	for _, creatureID := range defeatedCreatureIDs {
		if creatureID <= 0 {
			return nil, fmt.Errorf("invalid defeated creature id %d", creatureID)
		}
		defeated[creatureID] = struct{}{}
	}

	updates := make([]requirementTarget, 0)
	for _, definition := range battleCreatureRequirements {
		matched := true
		for creatureID, expectedName := range definition.creatures {
			if _, ok := defeated[creatureID]; !ok {
				matched = false
				break
			}
			creature := s.gameDataManager.GetCreature(creatureID)
			if creature == nil || creature.Name != expectedName {
				return nil, fmt.Errorf("creature authority for requirement %d is invalid", definition.requirementID)
			}
		}
		if matched {
			updates = append(updates, requirementTarget{RequirementID: definition.requirementID, AchievementID: definition.achievementID, Target: 1})
		}
	}
	return updates, nil
}
