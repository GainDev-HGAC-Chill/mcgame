
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

type scrollEffectHandler struct {
    service *Service
}

func expBookConfig(id int) (int, time.Duration, bool) {
    switch id {
    case 298:
        return 281, time.Hour, true
    case 368:
        return 281, 3 * time.Hour, true
    case 369:
        return 281, 12 * time.Hour, true
    case 1140:
        return 762, time.Hour, true
    case 1141:
        return 762, 3 * time.Hour, true
    case 1142:
        return 764, time.Hour, true
    case 1143:
        return 764, 3 * time.Hour, true
    case 2274:
        return 1538, time.Hour, true
    case 2275:
        return 1538, 3 * time.Hour, true
    }
    return 0, 0, false
}

func (h *scrollEffectHandler) CanHandle(
    tpl *models.ItemTemplateTemplate,
) bool {
    return int(tpl.Type) == 500
}

func (h *scrollEffectHandler) Apply(
    ctx context.Context,
    uctx *UseContext,
    char *character.Character,
    it *domainitem.Item,
    tpl *models.ItemTemplateTemplate,
) (*ItemEffectResult, error) {
    result := NewEffectResult()

    buffID, duration, ok := expBookConfig(int(tpl.ID))
    if !ok {
        result.Message = fmt.Sprintf("Đã sử dụng %s.", tpl.Name)
        return result, nil
    }

    if char == nil || it == nil || it.StackCount < 1 {
        return nil, pkgerrors.ErrCannotUseItem
    }

    if int(tpl.Type) != 500 || int(tpl.UseType) != 1 {
        return nil, pkgerrors.ErrCannotUseItem
    }

    if uctx != nil &&
        uctx.TargetType != 0 &&
        uctx.TargetType != 1 {
        return nil, pkgerrors.ErrCannotUseItem
    }

    if h.service == nil || h.service.buffService == nil {
        return nil, fmt.Errorf("EXP buff service unavailable")
    }

    buff, err := h.service.buffService.AddOrRefresh(
        ctx,
        char.ID,
        appbuff.AddRequest{
            BuffID:     buffID,
            BuffType:   domainbuff.TypeTimed,
            Source:     "exp_book",
            Duration:   duration,
            StackCount: 1,
        },
    )
    if err != nil {
        return nil, err
    }

    result.GrantedBuffs = append(result.GrantedBuffs, buff)
    result.Message = fmt.Sprintf(
        "Đã sử dụng %s.", tpl.Name,
    )

    return result, nil
}
