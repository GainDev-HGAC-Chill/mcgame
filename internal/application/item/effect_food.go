// Open-sourced by BaoLT

package item

import (
	"context"
	"fmt"
	"time"

	appbuff "mcgame-server/internal/application/buff"
	domainbuff "mcgame-server/internal/domain/buff"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type foodEffectHandler struct {
	itemService *Service
}

type foodBuffConfig struct {
	buffID   int
	duration time.Duration
}

var foodBuffConfigs = map[int]foodBuffConfig{
	1297: {buffID: 772, duration: 15 * time.Minute},
	1298: {buffID: 773, duration: 15 * time.Minute},
	1299: {buffID: 774, duration: 15 * time.Minute},
	1300: {buffID: 775, duration: 15 * time.Minute},
	1301: {buffID: 776, duration: 15 * time.Minute},
	1519: {buffID: 883, duration: 15 * time.Minute},
	1520: {buffID: 884, duration: 15 * time.Minute},
	1521: {buffID: 885, duration: 15 * time.Minute},
	1522: {buffID: 886, duration: 15 * time.Minute},
	1523: {buffID: 887, duration: 15 * time.Minute},
	1676: {buffID: 1169, duration: 15 * time.Minute},
	1677: {buffID: 1170, duration: 15 * time.Minute},
	1678: {buffID: 1171, duration: 15 * time.Minute},
	1679: {buffID: 1172, duration: 15 * time.Minute},
	1680: {buffID: 1173, duration: 15 * time.Minute},
	1856: {buffID: 2709, duration: time.Hour},
	1858: {buffID: 1231, duration: 15 * time.Minute},
	1859: {buffID: 1232, duration: 15 * time.Minute},
	1860: {buffID: 1233, duration: 15 * time.Minute},
	2051: {buffID: 1381, duration: 6 * time.Hour},
	2708: {buffID: 1868, duration: 30 * time.Minute},
	3912: {buffID: 2705, duration: 15 * time.Minute},
	3913: {buffID: 2706, duration: 15 * time.Minute},
	3914: {buffID: 2707, duration: 15 * time.Minute},
	3915: {buffID: 2708, duration: 15 * time.Minute},
	3916: {buffID: 2709, duration: time.Hour},
	4151: {buffID: 2838, duration: time.Hour},
	4153: {buffID: 2840, duration: time.Hour},
	4580: {buffID: 2854, duration: 30 * time.Minute},
	4581: {buffID: 2855, duration: 30 * time.Minute},
	4582: {buffID: 2856, duration: 30 * time.Minute},
	4583: {buffID: 2857, duration: 30 * time.Minute},
	4584: {buffID: 2858, duration: 30 * time.Minute},
}

func (h *foodEffectHandler) CanHandle(tpl *models.ItemTemplateTemplate) bool {
	return int(tpl.Type) == 502
}

func (h *foodEffectHandler) Apply(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
	result := NewEffectResult()
	if config, ok := foodBuffConfigs[int(tpl.ID)]; ok {
		if char == nil || it == nil || int(tpl.Type) != 502 || int(tpl.UseType) != 1 {
			return nil, pkgerrors.ErrCannotUseItem
		}
		if uctx != nil && uctx.TargetType != 0 && uctx.TargetType != 1 {
			return nil, pkgerrors.ErrCannotUseItem
		}
		if h.itemService == nil || h.itemService.buffService == nil {
			return nil, fmt.Errorf("food buff service unavailable")
		}
		buff, err := h.itemService.buffService.AddOrRefresh(ctx, char.ID, appbuff.AddRequest{
			BuffID:     config.buffID,
			BuffType:   domainbuff.TypeTimed,
			Source:     "food_item",
			Duration:   config.duration,
			StackCount: 1,
		})
		if err != nil {
			return nil, err
		}
		result.GrantedBuffs = []*domainbuff.Buff{buff}
		result.Message = fmt.Sprintf("Đã sử dụng %s.", tpl.Name)
		result.RefreshCharacterView = true
		return result, nil
	}

	hpAmount := int(tpl.I1)
	mpAmount := int(tpl.I2)

	if hpAmount > 0 {
		maxHP := char.MaxHP
		if uctx != nil && uctx.EffectiveMaxHP > maxHP {
			maxHP = uctx.EffectiveMaxHP
		}
		missing := maxHP - char.CurrentHP
		if hpAmount > missing {
			hpAmount = missing
		}
		if hpAmount > 0 {
			char.CurrentHP += hpAmount
			result.HPHealed = hpAmount
		}
	}

	if mpAmount > 0 {
		maxMP := char.MaxMP
		if uctx != nil && uctx.EffectiveMaxMP > maxMP {
			maxMP = uctx.EffectiveMaxMP
		}
		missing := maxMP - char.CurrentMP
		if mpAmount > missing {
			mpAmount = missing
		}
		if mpAmount > 0 {
			char.CurrentMP += mpAmount
			result.MPRestored = mpAmount
		}
	}
	if result.HPHealed == 0 && result.MPRestored == 0 {
		return nil, pkgerrors.ErrCannotUseItem
	}

	return result, nil
}
