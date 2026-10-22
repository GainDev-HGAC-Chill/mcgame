// Open-sourced by BaoLT

// Shop config handlers and currency helpers.
package shop

import (
	"strconv"
	"strings"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func getCurrencyName(currencyType int) string {
	switch currencyType {
	case 1:
		return "Bạc khóa"
	case 2:
		return "Bạc"
	case 3:
		return "Vàng khóa"
	case 4:
		return "Vàng"
	case 7:
		return "Huy chương Liên Minh"
	case 10:
		return "Điểm Năm Mới"
	case 11:
		return "Điểm Tết"
	case 12:
		return "Điểm Valentine"
	case 13:
		return "Điểm Lồng Đèn"
	case 14:
		return "Điểm Quốc Tế Lao Động"
	case 15:
		return "Điểm Câu Cá"
	case 16:
		return "Điểm Thất Tịch"
	case 17:
		return "Điểm Mùa Hè"
	case 18:
		return "Điểm Kỷ Niệm"
	case 19:
		return "Điểm Đấu Trường Thú"
	case 20:
		return "Cống hiến Bang Hội (Thường)"
	case 21:
		return "Cống hiến Bang Hội (Quyên góp)"
	case 22:
		return "Điểm Giáng Sinh"
	case 25:
		return "Điểm Quốc Khánh"
	case 26:
		return "Mảnh Thú Cưỡi"
	case 29:
		return "Điểm World Cup"
	case 30:
		return "Vàng World Cup"
	case 31:
		return "Điểm Olympic"
	case 49:
		return "Điểm Chu Niên"
	case 58:
		return "Điểm Ngày Độc Thân"
	case 60:
		return "Điểm ShowTime"
	case 61:
		return "Vàng Shop"
	case 62:
		return "Điểm Tiêu Hao Chu Niên"
	case 64:
		return "MC Beans"
	case 65:
		return "Điểm Hoạt Động Đấu Trường Thú"
	case 69:
		return "Điểm ShowTime 2"
	default:
		return "Điểm"
	}
}

func getCurrencyKey(currencyType int) string {
	switch currencyType {
	case 1:
		return "moneyBind"
	case 2:
		return "money"
	case 3:
		return "goldBind"
	case 4:
		return "gold"
	case 20:
		return "guildContrib"
	case 61:
		return "shopGold"
	case 64:
		return "mcbeans"
	case 0:
		return "exPoint"
	default:
		return "money"
	}
}

func (h *Handler) GetShopConfig(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	rows := []string{}
	if h.gameData != nil {
		slots := h.gameData.GetAllShopSlots()
		renderableSlots := make([]*models.ShopSlotTemplate, 0, len(slots))
		for _, slot := range slots {
			if shopSlotHasTemplate(h.gameData, slot) {
				renderableSlots = append(renderableSlots, slot)
				continue
			}
			if slot != nil {
				h.logger.Warn("GetShopConfig: skipped slot with missing template", zap.Int64("slot_id", slot.ID), zap.Int("table_type", int(slot.Type)), zap.Int("template_id", int(slot.ItemID)))
			}
		}
		rows = buildShopConfigRows(renderableSlots)
	}
	result := []interface{}{0, "", rows}
	if err := ctx.Connection.SendCallback("onGetShopConfig", result); err != nil {
		h.logger.Error("Failed to send onGetShopConfig callback", zap.Uint32("conn_id", ctx.ConnID), zap.Error(err))
	}

	return nil, nil
}

func shopSlotHasTemplate(gameData *gamedata.Manager, slot *models.ShopSlotTemplate) bool {
	if gameData == nil || slot == nil || slot.ItemID <= 0 {
		return false
	}
	templateID := int(slot.ItemID)
	switch int(slot.Type) {
	case 12:
		return gameData.GetCreature(templateID) != nil
	case 19:
		return gameData.GetEquipment(templateID) != nil
	case 29:
		return gameData.GetItem(templateID) != nil
	default:
		return false
	}
}

func buildShopConfigRows(slots []*models.ShopSlotTemplate) []string {
	rows := make([]string, 0, len(slots))
	for _, slot := range slots {
		if slot == nil {
			continue
		}
		fields := []struct {
			name  string
			value float64
		}{
			{"id", float64(slot.ID)},
			{"amount", slot.Amount},
			{"gold", slot.Gold},
			{"gt", slot.Gt},
			{"itemId", slot.ItemID},
			{"money", slot.Money},
			{"pNum1", slot.PNum1},
			{"pNum2", slot.PNum2},
			{"pType1", slot.PType1},
			{"pType2", slot.PType2},
			{"point", slot.Point},
			{"position", slot.Position},
			{"priceAll", slot.PriceAll},
			{"quality", slot.Quality},
			{"r", slot.R},
			{"sale", slot.Sale},
			{"sid", slot.Sid},
			{"st", slot.St},
			{"type", slot.Type},
		}
		var row strings.Builder
		row.WriteString("var obj={};")
		for _, field := range fields {
			row.WriteString("obj.")
			row.WriteString(field.name)
			row.WriteString("='")
			row.WriteString(strconv.FormatFloat(field.value, 'f', -1, 64))
			row.WriteString("';")
		}
		rows = append(rows, row.String())
	}
	return rows
}

func (h *Handler) GetLimitShopConfig(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {

	result := []interface{}{0, "", []interface{}{}}
	if err := ctx.Connection.SendCallback("onGetLimitShopConfig", result); err != nil {
		h.logger.Error("Failed to send onGetLimitShopConfig callback", zap.Uint32("conn_id", ctx.ConnID), zap.Error(err))
	}

	return nil, nil
}

func (h *Handler) GetRemainShopConfig(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {

	result := []interface{}{0, "", []interface{}{}}
	if err := ctx.Connection.SendCallback("onGetRemainShopConfig", result); err != nil {
		h.logger.Error("Failed to send onGetRemainShopConfig callback", zap.Uint32("conn_id", ctx.ConnID), zap.Error(err))
	}

	return nil, nil
}
