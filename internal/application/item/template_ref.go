
package item

import (
    "context"
    "fmt"

    domainitem "mcgame-server/internal/domain/item"
)

// TableType là bảng template của Flash: 19 hoặc 29.
type TemplateRef struct {
    TableType  int
    TemplateID int
}

func (s *Service) ResolveTemplateType(ref TemplateRef) (domainitem.ItemType, error) {
    if s == nil || s.gameDataRec == nil {
        return 0, fmt.Errorf("game data not initialized")
    }

    if ref.TemplateID <= 0 {
        return 0, fmt.Errorf("invalid template ID: %d", ref.TemplateID)
    }

    switch ref.TableType {
    case 19:
        if s.gameDataRec.GetEquipment(ref.TemplateID) == nil {
            return 0, fmt.Errorf("equipment %d not found", ref.TemplateID)
        }
        return domainitem.ItemTypeEquipment, nil

    case 29:
        tpl := s.gameDataRec.GetItem(ref.TemplateID)
        if tpl == nil {
            return 0, fmt.Errorf("item %d not found", ref.TemplateID)
        }

        switch int(tpl.Kind) {
        case 6, 14:
            return domainitem.ItemTypeMaterial, nil
        default:
            return domainitem.ItemTypeConsumable, nil
        }

    default:
        return 0, fmt.Errorf("unsupported table type: %d", ref.TableType)
    }
}

func (s *Service) AddTemplateItem(
    ctx context.Context,
    charID int64,
    ref TemplateRef,
    count int,
    isBound bool,
    colorCode int,
) (*domainitem.Item, error) {
    if count <= 0 {
        return nil, fmt.Errorf("invalid item count")
    }

    itemType, err := s.ResolveTemplateType(ref)
    if err != nil {
        return nil, err
    }

    return s.AddItemWithBindAndColor(
        ctx,
        charID,
        ref.TemplateID,
        itemType,
        count,
        isBound,
        colorCode,
    )
}
