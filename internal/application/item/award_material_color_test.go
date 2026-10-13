package item

import (
	"math"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestMaterialAwardColorUsesClientGradeNotEquipmentQuality(t *testing.T) {
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 16, Name: "Kim Cương", Kind: 6, Type: 600})
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 125, Name: "Potion", Kind: 5, Type: 501})
	h := &awardEffectHandler{gameData: manager}
	got, err := h.awardItemColorCode(29, 16, 0, map[string]interface{}{"materialColorCode": float64(3)})
	if err != nil || got != 3 {
		t.Fatalf("material color=%d err=%v", got, err)
	}
	for _, value := range []interface{}{float64(-1), float64(0), float64(5), 2.5, math.NaN(), math.Inf(1), "3", nil} {
		if _, err := h.awardItemColorCode(29, 16, 0, map[string]interface{}{"materialColorCode": value}); err == nil {
			t.Fatalf("accepted invalid color%v", value)
		}
	}
	if _, err := h.awardItemColorCode(29, 125, 0, map[string]interface{}{"materialColorCode": float64(3)}); err == nil {
		t.Fatal("accepted material override on potion")
	}
	got, err = h.awardItemColorCode(29, 16, 0, nil)
	if err != nil || got != 0 {
		t.Fatal("changed unconfigured default")
	}
	got, err = h.awardItemColorCode(19, 1009, 20, nil)
	if err != nil || got != 4 {
		t.Fatal("changed equipment quality behavior")
	}
}
