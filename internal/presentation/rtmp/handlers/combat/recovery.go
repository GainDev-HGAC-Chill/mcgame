// Open-sourced by BaoLT

// Combat recovery and relive handlers.
package combat

import (
	"strconv"

	"go.uber.org/zap"
	appbloodbag "mcgame-server/internal/application/bloodbag"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

func (h *Handler) MazeRecover(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	var hp int
	switch v := args[0].(type) {
	case float64:
		hp = int(v)
	case int:
		hp = v
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	h.logger.Info("Maze recover",
		zap.Int64("character_id", characterID),
		zap.Int("hp", hp))

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	if hp > 0 {
		char.CurrentHP += hp
		if char.CurrentHP > char.MaxHP {
			char.CurrentHP = char.MaxHP
		}

		if err := h.charService.Save(ctx.Context, char); err != nil {
			h.logger.Warn("Failed to save character after maze recover", zap.Error(err))
		}

		ctx.Connection.SendCallback("onUPP", map[string]interface{}{
			"hp": char.CurrentHP,
		})
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

func (h *Handler) UseBloodBag(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}
	if h.bloodBag == nil {
		return nil, nil
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	var (
		charUsed       int64
		petHPUsed      int64
		petMPUsed      int64
		reserve        appbloodbag.State
		reserveChanged bool
		charHP         = char.CurrentHP
	)

	// 1. Hồi HP nhân vật.
	if char.CurrentHP > 0 {
		effective := *char
		effective.RecalculateStats()

		if h.itemService != nil {
			bonuses := h.itemService.AggregateEquipmentStats(
				ctx.Context, characterID,
			)
			effective.ApplyEquipmentBonuses(bonuses)
		}

		if effective.MaxHP > char.CurrentHP {
			newHP, used, state, recoveryErr :=
				h.bloodBag.RestoreCharacterHP(
					ctx.Context, characterID, effective.MaxHP,
				)

			if recoveryErr != nil {
				h.logger.Error("Blood bag character recovery failed",
					zap.Int64("character_id", characterID),
					zap.Error(recoveryErr))
			} else if used > 0 {
				charHP = newHP
				charUsed = used
				reserve = state
				reserveChanged = true

				// Đồng bộ cache để HP không bị ghi đè.
				char.CurrentHP = newHP
				if saveErr := h.charService.Save(
					ctx.Context, char,
				); saveErr != nil {
					h.logger.Error("Blood bag character cache sync failed",
						zap.Int64("character_id", characterID),
						zap.Error(saveErr))
					return nil, nil
				}
			}
		}
	}

	// 2. Hồi HP và MP pet, độc lập với HP nhân vật.
	if h.petService != nil {
		pet, petErr := h.petService.GetActivePetForRecovery(
			ctx.Context, characterID,
		)

		if petErr != nil {
			h.logger.Error("Blood bag active pet lookup failed",
				zap.Int64("character_id", characterID),
				zap.Error(petErr))
		} else if pet != nil && pet.Life > 0 {
			if (pet.CurrentHP > 0 && pet.CurrentHP < pet.MaxHP) ||
				(pet.CurrentMP < pet.MaxMP) {

				newPetHP, newPetMP, usedHP, usedMP, state, recoveryErr :=
					h.bloodBag.RestorePetVitals(
						ctx.Context,
						characterID,
						pet.ID,
						pet.MaxHP,
						pet.MaxMP,
					)

				if recoveryErr != nil {
					h.logger.Error("Blood bag pet recovery failed",
						zap.Int64("character_id", characterID),
						zap.Int64("pet_id", pet.ID),
						zap.Error(recoveryErr))
				} else if usedHP > 0 || usedMP > 0 {
					petHPUsed = usedHP
					petMPUsed = usedMP
					reserve = state
					reserveChanged = true

					h.logger.Info("Blood bag pet restored",
						zap.Int64("character_id", characterID),
						zap.Int64("pet_id", pet.ID),
						zap.Int("new_hp", newPetHP),
						zap.Int("new_mp", newPetMP),
						zap.Int64("hp_consumed", usedHP),
						zap.Int64("mp_consumed", usedMP))
				}
			}
		}
	}

	// 3. Gửi trạng thái dự trữ mới nhất về Flash.
	if reserveChanged && ctx.Connection != nil {
		_ = ctx.Connection.SendCallback(
			"onUPP",
			map[string]interface{}{
				"currentHp": charHP,
				"currentMp": char.CurrentMP,
				"ep":        reserve.String(),
			},
		)
	}

	if charUsed > 0 {
		h.logger.Info("Blood bag character HP restored",
			zap.Int64("character_id", characterID),
			zap.Int("new_hp", charHP),
			zap.Int64("consumed", charUsed))
	}

	_ = petHPUsed
	_ = petMPUsed

	return nil, nil
}

func (h *Handler) BBGoldRelive(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	h.logger.Info("BLOODBAG_HP_CHECK",
		zap.Int64("character_id", characterID),
		zap.Int("current_hp", char.CurrentHP),
		zap.Int("max_hp", char.MaxHP),
		zap.Int("current_mp", char.CurrentMP),
		zap.Int("max_mp", char.MaxMP),
		zap.Bool("service_ready", h.bloodBag != nil),
	)
	goldCost := int64(100)
	if char.MoneyBind < goldCost {
		h.logger.Warn("BBGoldRelive: insufficient gold",
			zap.Int64("character_id", characterID),
			zap.Int64("money_bind", char.MoneyBind),
			zap.Int64("cost", goldCost))
		return map[string]interface{}{
			"success": false,
			"error":   "insufficient_gold",
		}, nil
	}

	char.MoneyBind -= goldCost
	char.CurrentHP = char.MaxHP
	char.CurrentMP = char.MaxMP
	if err := h.charService.Save(ctx.Context, char); err != nil {
		h.logger.Warn("Failed to save character after bb gold relive", zap.Error(err))
	}

	ctx.Connection.SendCallback("onUPP", map[string]interface{}{
		"currentHp": char.CurrentHP,
		"currentMp": char.CurrentMP,
		"moneyBind": char.MoneyBind,
	})

	h.logger.Info("Battle boss gold relive",
		zap.Int64("character_id", characterID),
		zap.Int64("gold_spent", goldCost))

	return map[string]interface{}{
		"success": true,
	}, nil
}

func (h *Handler) BBNormalRelive(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	char.CurrentHP = char.MaxHP / 2
	char.CurrentMP = char.MaxMP / 2
	if char.CurrentHP < 1 {
		char.CurrentHP = 1
	}
	if err := h.charService.Save(ctx.Context, char); err != nil {
		h.logger.Warn("Failed to save character after bb normal relive", zap.Error(err))
	}

	ctx.Connection.SendCallback("onUPP", map[string]interface{}{
		"currentHp": char.CurrentHP,
		"currentMp": char.CurrentMP,
	})

	h.logger.Info("Battle boss normal relive",
		zap.Int64("character_id", characterID))

	return map[string]interface{}{
		"success": true,
	}, nil
}

func (h *Handler) FreeRelive(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	var reliveType int
	if len(args) >= 1 {
		switch v := args[0].(type) {
		case float64:
			reliveType = int(v)
		case int:
			reliveType = v
		}
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	char.CurrentHP = char.MaxHP
	char.CurrentMP = char.MaxMP
	if err := h.charService.Save(ctx.Context, char); err != nil {
		h.logger.Warn("Failed to save character after free relive", zap.Error(err))
	}

	ctx.Connection.SendCallback("onUPP", map[string]interface{}{
		"currentHp": char.CurrentHP,
		"currentMp": char.CurrentMP,
	})

	h.logger.Info("Free relive",
		zap.Int64("character_id", characterID),
		zap.Int("type", reliveType))

	return map[string]interface{}{
		"success": true,
	}, nil
}

func (h *Handler) ReliveUseItem(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	var itemID int64
	if len(args) >= 1 {
		switch v := args[0].(type) {
		case float64:
			itemID = int64(v)
		case int:
			itemID = int64(v)
		}
	}

	if itemID > 0 && h.itemService != nil {
		if err := h.itemService.ConsumeItemByID(ctx.Context, characterID, itemID); err != nil {
			h.logger.Warn("ReliveUseItem: failed to consume item",
				zap.Int64("character_id", characterID),
				zap.Int64("item_id", itemID),
				zap.Error(err))
			return map[string]interface{}{
				"success": false,
				"error":   "item_consume_failed",
			}, nil
		}
	}

	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	char.CurrentHP = char.MaxHP
	char.CurrentMP = char.MaxMP
	if err := h.charService.Save(ctx.Context, char); err != nil {
		h.logger.Warn("Failed to save character after relive", zap.Error(err))
	}

	ctx.Connection.SendCallback("onUPP", map[string]interface{}{
		"currentHp": char.CurrentHP,
		"currentMp": char.CurrentMP,
	})

	h.logger.Info("Relive use item",
		zap.Int64("character_id", characterID),
		zap.Int64("item_id", itemID),
		zap.Int("new_hp", char.CurrentHP))

	return map[string]interface{}{
		"success": true,
	}, nil
}

func (h *Handler) ReliveSwitch(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	h.logger.Debug("Relive switch requested",
		zap.String("character_id", ctx.CharacterID))

	return 1, nil
}

func (h *Handler) QuitWatch(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	var battleID string
	if len(args) >= 1 && args[0] != nil {
		switch v := args[0].(type) {
		case string:
			battleID = v
		}
	}

	if battleID == "" {
		battleID = ctx.Connection.GetBattleID()
	}

	if battleID != "" {
		battle, err := h.combatService.GetBattle(battleID)
		if err == nil {
			battle.RemoveWatcher(ctx.Connection)
			h.logger.Info("QuitWatch: watcher removed",
				zap.String("battle_id", battleID),
				zap.String("character_id", ctx.CharacterID))
		}
	}

	h.logger.Info("Quit watch battle",
		zap.String("character_id", ctx.CharacterID),
		zap.String("battle_id", battleID))

	return map[string]interface{}{
		"success": true,
	}, nil
}
