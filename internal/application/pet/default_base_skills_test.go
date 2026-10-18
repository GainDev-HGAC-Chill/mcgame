package pet

import (
	"encoding/json"
	"fmt"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestSeedNamedPetDefaultSkills_AllFiveTemplates(t *testing.T) {
	tests := []struct {
		name   string
		skills []string
	}{
		{"Bạch Hổ Cuồng Nộ", []string{"Hổ Khiếu Sơn Lâm", "Chiến Đấu Máu Lửa", "Thú Vương Đột Kích", "Đột Kích Bí Mật"}},
		{"Kim Tước", []string{"Thánh Quang Thanh Tẩy", "Kiên Bàn Tâm", "Mãnh Lực Trùng Kích"}},
		{"Long Tiểu Thanh", []string{"Thanh Long Cửu Biến", "Uy Nghiêm Của Rồng", "Hủy Diệt Phún Xạ"}},
		{"võ sĩ shiba", []string{"Trảm Linh", "Tàn Tâm"}},
		{"Đại Thánh Chí Tôn", []string{"Di Hình Hoán Vị", "Vô Song Loạn Vũ", "Bách Biến Trùng Kích"}},
	}

	for testIndex, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			creatureID := int64(70000 + testIndex)
			template := models.CreatureTemplate{ID: creatureID, Name: tc.name}
			manager := newContractPetGameDataWithSkills(t,
				[]models.CreatureTemplate{template},
				[]models.CreatureSkillTemplate{{ID: 1, Cid: float64(creatureID), Position: 1, Sid: 9999}},
			)
			rows := make([]json.RawMessage, 0, len(tc.skills))
			for i, name := range tc.skills {
				skill := models.SkillTemplate{
					ID: int64(80000 + i), Name: "  " + name + "  ", Level: 1,
					CodeName: fmt.Sprintf("BASE_TEST_%d", i),
				}
				raw, err := json.Marshal(skill)
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, raw)
			}
			if err := manager.GetCache().LoadTable(models.TableSkill, rows); err != nil {
				t.Fatal(err)
			}
			p := domainpet.NewPetFromCreature(123, &template)
			ensurePetSkillSlots(p)
			service := NewService(nil, zap.NewNop())
			service.SetGameDataManager(manager)
			service.seedDefaultPetSkills(p)

			for i := range tc.skills {
				if got, want := petSkillSlotValue(p, i+1), 80000+i; got != want {
					t.Fatalf("skill%d = %d, want %d", i+1, got, want)
				}
			}
			if len(tc.skills) < petSkillDefaultOpenSlots {
				if got := petSkillSlotValue(p, len(tc.skills)+1); got != 9999 {
					t.Fatalf("remaining template skill = %d, want 9999", got)
				}
			}
			if got := petSkillSlotValue(p, 6); got != -1 {
				t.Fatalf("skill6 = %d, want locked (-1)", got)
			}
		})
	}
}

func TestSeedNamedPetDefaultSkills_MissingSkillFallsBack(t *testing.T) {
	template := models.CreatureTemplate{ID: 71001, Name: "Võ Sĩ Shiba"}
	manager := newContractPetGameDataWithSkills(t,
		[]models.CreatureTemplate{template},
		[]models.CreatureSkillTemplate{{ID: 1, Cid: float64(template.ID), Position: 1, Sid: 7777}},
	)
	p := domainpet.NewPetFromCreature(123, &template)
	ensurePetSkillSlots(p)
	service := NewService(nil, zap.NewNop())
	service.SetGameDataManager(manager)
	service.seedDefaultPetSkills(p)
	if got := petSkillSlotValue(p, 1); got != 7777 {
		t.Fatalf("fallback skill1 = %d, want 7777", got)
	}
	if got := petSkillSlotValue(p, 2); got != 0 {
		t.Fatalf("fallback skill2 = %d, want 0", got)
	}
}

func TestSeedNamedPetDefaultSkills_DoesNotModifyOtherPetTemplates(t *testing.T) {
	for _, name := range []string{"Huyền Minh Linh Vũ", "Lôi Lân Ảnh", "Đại Thánh Chí Tôn [Tiến Hóa]"} {
		t.Run(name, func(t *testing.T) {
			template := models.CreatureTemplate{ID: 72001, Name: name}
			manager := newContractPetGameDataWithSkills(t,
				[]models.CreatureTemplate{template},
				[]models.CreatureSkillTemplate{{ID: 1, Cid: float64(template.ID), Position: 1, Sid: 7001}},
			)
			p := domainpet.NewPetFromCreature(123, &template)
			ensurePetSkillSlots(p)
			service := NewService(nil, zap.NewNop())
			service.SetGameDataManager(manager)
			service.seedDefaultPetSkills(p)
			if got := petSkillSlotValue(p, 1); got != 7001 {
				t.Fatalf("other pet skill1 = %d, want 7001", got)
			}
		})
	}
}
