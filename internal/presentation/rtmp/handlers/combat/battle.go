// Open-sourced by BaoLT

// Combat battle flow handlers.
package combat

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	appcombat "mcgame-server/internal/application/combat"
	domainchar "mcgame-server/internal/domain/character"
	combatdomain "mcgame-server/internal/domain/combat"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/infrastructure/rtmp"
	questhandler "mcgame-server/internal/presentation/rtmp/handlers/quest"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

const randomEncounterCooldown = 4 * time.Second

func (h *Handler) HitNpc(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	var npcID int
	switch v := args[0].(type) {
	case float64:
		npcID = int(v)
	case int:
		npcID = v
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	playerInFront := ctx.Connection == nil || ctx.Connection.IsPlayerInFront()
	currentMapID, _ := ctx.Connection.GetSceneInfo()
	channelID := 0
	if ctx.Connection != nil {
		channelID = ctx.Connection.GetChannelID()
	}
	roster := h.eligibleBattleRoster(ctx.Context, characterID, currentMapID)

	var battle *combatdomain.Battle
	if h.combatService.IsScheduleBoss(npcID) {
		battle, err = h.combatService.StartScheduleBossBattleWithRoster(ctx.Context, roster, npcID, currentMapID, channelID, playerInFront)
	} else if len(roster) > 1 {
		npcTemplate, templateErr := h.combatService.BuildNPCTemplateForCharacter(ctx.Context, characterID, npcID)
		if templateErr != nil {
			return rtmp.ErrorToResponse(templateErr), nil
		}
		battle, err = h.combatService.StartPVEBattleWithRoster(ctx.Context, roster, currentMapID, channelID, []appcombat.NPCTemplateData{npcTemplate}, playerInFront)
	} else {
		battle, err = h.combatService.StartPVEBattle(ctx.Context, characterID, npcID, channelID, playerInFront)
	}
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("PVE battle initiated",
		zap.Int64("character_id", characterID),
		zap.Int("npc_id", npcID),
		zap.Int64s("roster", roster),
		zap.Bool("player_in_front", playerInFront),
		zap.String("battle_id", battle.ID))

	battleDTO := battle.ToDTO()
	onBattleStartData := h.notifyBattleStarted(battle)

	h.logger.Info("Sending onBattleStart",
		zap.String("battle_field_id", battle.BattleFieldID),
		zap.Bool("player_in_front", playerInFront),
		zap.Any("onBattleStartData", onBattleStartData))

	return map[string]interface{}{
		"success":    true,
		"battleId":   battle.ID,
		"battleData": battleDTO,
	}, nil
}

func (h *Handler) CombatAction(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 4 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	var battleID string
	var actionType, skillID int
	var targetID string

	switch v := args[0].(type) {
	case string:
		battleID = v
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}

	switch v := args[1].(type) {
	case float64:
		actionType = int(v)
	case int:
		actionType = v
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}

	switch v := args[2].(type) {
	case float64:
		skillID = int(v)
	case int:
		skillID = v
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}

	switch v := args[3].(type) {
	case string:
		targetID = v
	case float64:
		targetID = string(rune(int(v)))
	case int:
		targetID = string(rune(v))
	default:
		targetID = ""
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	playerAction, err := h.combatService.ExecuteAction(
		ctx.Context,
		battleID,
		fmt.Sprintf("%d", characterID),
		combatdomain.ActionType(actionType),
		skillID,
		targetID,
	)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	battle, err := h.combatService.GetBattleForCleanup(battleID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	var npcActions []*combatdomain.BattleAction
	if battle.IsActive() {
		npcActions = h.combatService.ProcessNPCTurn(ctx.Context, battle)
	}

	actionDTOs := make([]map[string]interface{}, 0)
	actionDTOs = append(actionDTOs, playerAction.ToDTO())
	for _, action := range npcActions {
		actionDTOs = append(actionDTOs, action.ToDTO())
	}

	response := map[string]interface{}{
		"success":      true,
		"battleId":     battleID,
		"round":        battle.CurrentRound,
		"actions":      actionDTOs,
		"participants": nil,
	}

	participants := make([]map[string]interface{}, len(battle.Participants))
	for index, participant := range battle.Participants {
		participants[index] = participant.ToDTO()
	}
	response["participants"] = participants

	if battle.IsEnded() && battle.Result != nil {
		response["battleEnded"] = true
		response["result"] = battle.Result.ToDTO()
	}

	h.logger.Debug("Combat action processed",
		zap.String("battle_id", battleID),
		zap.Int64("character_id", characterID),
		zap.Int("skill_id", skillID))

	return response, nil
}

func (h *Handler) GetBattleInfo(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	var battleID string
	characterID := ""
	if ctx != nil {
		characterID = ctx.CharacterID
	}

	if len(args) >= 1 && args[0] != nil {
		switch v := args[0].(type) {
		case string:
			battleID = v
		}
	}

	if battleID == "" && ctx != nil && ctx.Connection != nil {
		battleID = ctx.Connection.GetBattleID()
	}

	if battleID == "" {
		return emptyBattleInfoResponse(), nil
	}

	battle, err := h.lookupBattleForInfo(battleID)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			h.logger.Debug("Battle info requested after battle cleanup",
				zap.String("battle_id", battleID),
				zap.String("character_id", characterID))
			return emptyBattleInfoResponse(), nil
		}

		h.logger.Warn("Failed to load battle info",
			zap.String("battle_id", battleID),
			zap.String("character_id", characterID),
			zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}

	return buildBattleInfoResponse(battle), nil
}

func (h *Handler) lookupBattleForInfo(battleID string) (*combatdomain.Battle, error) {
	if h.battleInfoLookup != nil {
		return h.battleInfoLookup(battleID)
	}
	if h.combatService == nil {
		return nil, pkgerrors.ErrNotFound
	}

	return h.combatService.GetBattleForCleanup(battleID)
}

func buildBattleInfoResponse(battle *combatdomain.Battle) map[string]interface{} {
	if battle == nil {
		return emptyBattleInfoResponse()
	}

	response := make(map[string]interface{}, len(battle.Participants))
	for _, participant := range battle.Participants {
		if participant == nil {
			continue
		}

		response[strconv.Itoa(participant.Position)] = map[string]interface{}{
			"keepRound": map[string]interface{}{},
		}
	}

	if len(response) == 0 {
		return emptyBattleInfoResponse()
	}

	return response
}

func emptyBattleInfoResponse() map[string]interface{} {
	return map[string]interface{}{}
}

func (h *Handler) EndBattle(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, parseErr := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if parseErr != nil || characterID <= 0 {
		return nil, pkgerrors.ErrUnauthorized
	}
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
	if battleID == "" {
		return map[string]interface{}{
			"success": true,
			"rewards": map[string]interface{}{
				"exp":   0,
				"money": 0,
				"items": []interface{}{},
			},
		}, nil
	}

	battle, err := h.combatService.GetCharacterBattleForCleanup(battleID, characterID)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrUnauthorized) {
			return rtmp.ErrorToResponse(err), nil
		}
		if h.battleService != nil {
			h.battleService.ClearBattleDeadlines(battleID)
		}
		ctx.Connection.ClearBattle()
		ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{
			"cid":   ctx.CharacterID,
			"state": clientCharStateNormal,
		})
		h.broadcastSceneCharState(ctx.Connection, ctx.CharacterID, clientCharStateNormal)
		h.logger.Warn("Battle not found on battlePlayEnd, cleared connection state",
			zap.String("battle_id", battleID),
			zap.String("character_id", ctx.CharacterID),
			zap.Error(err))
		return map[string]interface{}{
			"success": true,
			"rewards": map[string]interface{}{
				"exp":   0,
				"money": 0,
				"items": []interface{}{},
			},
		}, nil
	}

	if battle.IsActive() {
		ended, winner := battle.CheckVictory()
		if !ended {
			winner = combatdomain.SideEnemy
		}
		h.combatService.EndBattle(ctx.Context, battle, winner)
	}

	if parseErr == nil {
		updatedQuests, questKill, creditErr := h.collectQuestKillUpdates(ctx.Context, characterID, battle)
		if creditErr != nil {
			return nil, creditErr
		}
		activeCount := 0
		if len(updatedQuests) > 0 {
			activeQuests, activeErr := h.questService.GetActiveQuests(ctx.Context, characterID)
			if activeErr != nil {
				h.logger.Warn("Failed to load active quest count after battle", zap.Error(activeErr))
			} else {
				activeCount = len(activeQuests)
			}
		}
		for _, questProgress := range updatedQuests {
			dto := h.questService.EnrichQuestDTO(ctx.Context, questProgress)
			payload := questhandler.BuildAddQuestPayload(questProgress, dto, activeCount)
			if err := ctx.Connection.SendCallback("onAddChaQuest", payload); err != nil {
				h.logger.Warn("Failed to send onAddChaQuest callback",
					zap.Int64("char_id", characterID),
					zap.Int("quest_id", questProgress.QuestID),
					zap.Error(err))
			}

			if questProgress.IsComplete() && h.questNotify != nil && h.gameData != nil {
				questTemplate := h.gameData.GetQuest(questProgress.QuestID)
				if questTemplate != nil {
					finishNPC := int(questTemplate.FinishNPC)
					if finishNPC > 0 {
						h.questNotify.PushNpcQuestStates(ctx, characterID, []int{finishNPC})
					}
				}
			}
		}
		if len(questKill) > 0 {
			if err := ctx.Connection.SendCallback("onUpdateQuestKill", questKill); err != nil {
				h.logger.Warn("Failed to send onUpdateQuestKill callback",
					zap.Int64("char_id", characterID),
					zap.Error(err))
			}
		}
	}

	var starInstanceResponse interface{}
	if battle.BattleType == combatdomain.BattleTypeStarInstance && characterID != 0 && h.starInstanceService != nil {
		deaths := 0
		for _, p := range battle.Participants {
			if p.Side == combatdomain.SidePlayer && p.CurrentHP <= 0 {
				deaths++
			}
		}
		var elapsedSec int64
		if battle.StartedAt != nil && !battle.StartedAt.IsZero() {
			elapsedSec = int64(time.Since(*battle.StartedAt).Seconds())
		}
		win := battle.Result != nil && battle.Result.WinnerSide == combatdomain.SidePlayer
		siResult, siErr := h.starInstanceService.OnBattleEnd(ctx.Context, characterID, battle.WarMapSid, battle.WarMapLevel, deaths, elapsedSec, win)
		if siErr != nil {
			h.logger.Error("StarInstance OnBattleEnd failed",
				zap.Int64("char_id", characterID),
				zap.Error(siErr))
		} else {
			_ = ctx.Connection.SendCallback("setStarPoint", siResult.NewStarPnt)
			rVal := 0
			if win {
				rVal = 1
			}
			starInstanceResponse = map[string]interface{}{
				"r": rVal,
				"s": map[string]interface{}{
					"id":    battle.WarMapSid,
					"level": battle.WarMapLevel,
					"score": siResult.Score,
					"time":  int(elapsedSec),
					"die":   deaths,
				},
			}
		}
	}

	result := battle.Result
	if result == nil {
		result = &combatdomain.BattleResult{
			WinnerSide:       combatdomain.SideEnemy,
			ItemRewards:      []combatdomain.ItemDrop{},
			PetRewards:       []combatdomain.PetReward{},
			CharacterRewards: map[int64]*combatdomain.CharacterReward{},
		}
	}
	if h.achievementService != nil {
		won := result.WinnerSide == combatdomain.SidePlayer
		update, progressErr := h.achievementService.RecordBattleEnded(ctx.Context, characterID, battleID, battle.BattleType, won, time.Now().UTC())
		if progressErr != nil {
			h.logger.Warn("Failed to record battle achievement progress",
				zap.Int64("character_id", characterID),
				zap.String("battle_id", battleID),
				zap.Error(progressErr))
		} else {
			rtmputils.SendAchievementProgressCallbacks(ctx.Connection, update)
		}
	}
	characterID, _ = strconv.ParseInt(ctx.CharacterID, 10, 64)
	rewardDTO := result.ToDTOForCharacter(characterID)
	playerParticipant := battle.GetParticipant(fmt.Sprintf("%d", characterID))
	if playerParticipant != nil {
		char, charErr := h.charService.GetByID(ctx.Context, characterID)
		if charErr == nil {
			h.logger.Info("BATTLE_HP_DEBUG",
				zap.Int64("character_id", characterID),
				zap.Int("db_loaded_hp", char.CurrentHP),
				zap.Int("db_loaded_max_hp", char.MaxHP),
				zap.Int("battle_hp", playerParticipant.CurrentHP),
				zap.Int("battle_mp", playerParticipant.CurrentMP),
			)
			char.CurrentHP = playerParticipant.CurrentHP
			char.CurrentMP = playerParticipant.CurrentMP
			char.CurrentSP = playerParticipant.CurrentSP
			if err := h.charService.Save(ctx.Context, char); err != nil {
				h.logger.Warn("Failed to save character HP after battle", zap.Error(err))
			} else {
				rewardSnapshot := result.CharacterReward(characterID)
				expReward := int64(0)
				leveledUp := false
				skills := []map[string]interface{}{}
				if rewardSnapshot != nil {
					expReward = rewardSnapshot.ExpReward
					leveledUp = rewardSnapshot.LeveledUp
				}
				if leveledUp && h.achievementService != nil {
					update, progressErr := h.achievementService.RecordLevel(ctx.Context, characterID, char.Level, time.Now().UTC())
					if progressErr != nil {
						h.logger.Warn("Failed to record level achievement progress", zap.Error(progressErr))
					} else {
						rtmputils.SendAchievementProgressCallbacks(ctx.Connection, update)
					}
				}
				if leveledUp && h.skillService != nil {
					loadedSkills, skillErr := h.skillService.GetSkillsForCallback(ctx.Context, characterID)
					if skillErr != nil {
						h.logger.Warn("Failed to build skill callback payload after battle level up",
							zap.Int64("character_id", characterID),
							zap.Error(skillErr))
					} else {
						skills = loadedSkills
					}
				}
				var bonusesPtr *domainchar.EquipmentStatBonuses
				if h.itemService != nil {
					bonuses := h.itemService.AggregateEquipmentStats(ctx.Context, characterID)
					bonusesPtr = &bonuses
				}
				sendBattleCharacterProgressCallbacks(ctx.Connection, char, expReward, leveledUp, bonusesPtr, skills)
				battlePet, hasBattlePet := playerBattlePetParticipantByOwner(battle.Participants, ctx.CharacterID)
				if hasBattlePet && h.petService != nil {
					refreshedPet, petErr := h.petService.GetPetDetail(ctx.Context, characterID, battlePet.EntityID)
					if petErr != nil {
						h.logger.Warn("Failed to load battle pet for progress refresh",
							zap.Int64("character_id", characterID),
							zap.Error(petErr))
					} else if refreshedPet != nil {
						sendBattlePetProgressCallbacks(ctx.Connection, refreshedPet, refreshedPet.Level > battlePet.Level)
					}
				}
				sendBattleRewardItemCallbacks(ctx.Connection, rewardDTO, h.gameData)
				if pets, ok := rewardDTO["pets"].([]map[string]interface{}); ok {
					for _, petReward := range pets {
						if petData, ok := petReward["pet"].(map[string]interface{}); ok {
							_ = ctx.Connection.SendCallback("onAddPet", petData)
						}
					}
				}
				if h.itemService != nil {
					rtmputils.CheckBagCapacityWarning(ctx.Context, ctx.Connection, h.itemService, characterID)
				}
			}
		}
	}

	if h.battleService != nil {
		h.battleService.ClearBattleDeadlines(battleID)
	}

	ctx.Connection.ClearBattle()
	ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{
		"cid":   ctx.CharacterID,
		"state": clientCharStateNormal,
	})
	h.broadcastSceneCharState(ctx.Connection, ctx.CharacterID, clientCharStateNormal)
	ctx.Connection.SetRandomEncounterCooldown(ctx.Timestamp.Add(randomEncounterCooldown))
	h.combatService.AckBattleCleanup(battleID, characterID)
	if h.sthBattleResultHandler != nil {
		won := result.WinnerSide == combatdomain.SidePlayer
		if sthErr := h.sthBattleResultHandler(ctx, characterID, battleID, won); sthErr != nil {
			h.logger.Error("STH battle completion failed", zap.String("battle_id", battleID), zap.Error(sthErr))
		}
	}

	h.logger.Info("Battle play ended",
		zap.String("battle_id", battleID),
		zap.Int64("exp_reward", result.ExpReward),
		zap.Int64("money_reward", result.MoneyReward))

	if starInstanceResponse != nil {
		return starInstanceResponse, nil
	}

	return map[string]interface{}{
		"success": true,
		"rewards": rewardDTO,
	}, nil
}

func incrementQuestKillCount(current interface{}) int {
	switch typed := current.(type) {
	case int:
		return typed + 1
	case int32:
		return int(typed) + 1
	case int64:
		return int(typed) + 1
	case float64:
		return int(typed) + 1
	default:
		return 1
	}
}

func (h *Handler) collectQuestKillUpdates(ctx context.Context, characterID int64, battle *combatdomain.Battle) ([]*domainquest.QuestProgress, map[string]interface{}, error) {
	if h.questService == nil || h.combatService == nil || battle == nil {
		return nil, nil, nil
	}
	claimedBattle, claimed, err := h.combatService.ClaimQuestKillCredit(battle.ID, characterID)
	if err != nil {
		return nil, nil, err
	}
	if !claimed {
		return nil, nil, nil
	}
	defer h.combatService.ReleaseQuestKillCredit(battle.ID, characterID)
	updated, kills, err := h.applyQuestKillProgress(ctx, characterID, claimedBattle)
	if err != nil {
		return nil, nil, err
	}
	h.combatService.CompleteQuestKillCredit(battle.ID, characterID)
	return updated, kills, nil
}

func (h *Handler) applyQuestKillProgress(ctx context.Context, characterID int64, battle *combatdomain.Battle) ([]*domainquest.QuestProgress, map[string]interface{}, error) {
	enemies := battle.GetParticipantsBySide(combatdomain.SideEnemy)
	questKill := map[string]interface{}{}
	updatedQuests := make([]*domainquest.QuestProgress, 0)
	for _, enemy := range enemies {
		if enemy == nil || enemy.EntityType != combatdomain.ParticipantTypeCreature || enemy.IsAlive || enemy.CurrentHP > 0 {
			continue
		}

		entityID := strconv.FormatInt(enemy.EntityID, 10)
		questKill[entityID] = incrementQuestKillCount(questKill[entityID])

		eventID := battle.ID + ":" + enemy.ID
		quests, err := h.questService.OnMonsterKilledWithCredit(ctx, characterID, int(enemy.EntityID), eventID)
		if err != nil {
			h.logger.Warn("Failed to update quest progress", zap.Error(err))
			return nil, nil, fmt.Errorf("battle %s quest kill %s: %w", battle.ID, enemy.ID, err)
		}
		updatedQuests = append(updatedQuests, quests...)
	}

	return updatedQuests, questKill, nil
}

func (h *Handler) LearnSkill(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	skillID, ok := parseSkillLearnID(args[0])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}

	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}

	if response, handled, handlerErr := h.tryUpgradeLifeSkill(ctx, characterID, skillID); handled || handlerErr != nil {
		return response, handlerErr
	}

	if h.skillService == nil {
		return false, nil
	}

	position, positionProvided, err := parseSkillLearnPosition(args)
	if err != nil {
		return nil, err
	}

	upgrade, err := h.skillService.UpgradeSkillWithResult(ctx.Context, characterID, skillID)
	if err != nil {
		h.logger.Warn("Failed to learn or upgrade skill",
			zap.Int64("character_id", characterID),
			zap.Int("skill_id", skillID),
			zap.Error(err))
		return false, nil
	}
	charSkill := upgrade.Skill

	if positionProvided {
		if err := h.skillService.ApplySkillPosition(ctx.Context, characterID, charSkill.SkillID, position); err != nil {
			h.logger.Warn("Failed to apply skill position",
				zap.Int64("character_id", characterID),
				zap.Int("skill_id", charSkill.SkillID),
				zap.Error(err))
			return false, nil
		}
	}

	h.sendSkillStateUpdate(ctx, characterID)
	if upgrade.Changed && upgrade.Character != nil {
		payload := map[string]interface{}{
			"expSkill": upgrade.Character.Experience,
		}
		if upgrade.MoneyKey == "money" {
			payload["money"] = upgrade.Character.Money
		} else {
			payload["moneyBind"] = upgrade.Character.MoneyBind
		}
		ctx.Connection.SetExpSkill(upgrade.Character.Experience)
		_ = ctx.Connection.SendCallback("onUPP", payload)
	}

	h.logger.Info("Skill learned",
		zap.Int64("character_id", characterID),
		zap.Int("skill_id", skillID))

	return true, nil
}

func parseSkillLearnID(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case string:
		parsed, err := strconv.Atoi(typed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func parseSkillLearnPosition(args []interface{}) (*int, bool, error) {
	if len(args) < 2 {
		return nil, false, nil
	}

	switch v := args[1].(type) {
	case float64:
		position := int(v)
		if position <= 0 {
			return nil, true, nil
		}
		return &position, true, nil
	case int:
		position := v
		if position <= 0 {
			return nil, true, nil
		}
		return &position, true, nil
	case nil:
		return nil, true, nil
	default:
		return nil, false, pkgerrors.ErrInvalidArgs
	}
}

func shouldResolveDefeatedPlayerBattle(battle *combatdomain.Battle, characterID int64) bool {
	if battle == nil || !battle.IsActive() {
		return false
	}

	player := battle.GetParticipant(fmt.Sprintf("%d", characterID))
	return player != nil && !player.IsAlive
}

func (h *Handler) BattleReplayEnd(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	h.logger.Info("Battle replay ended",
		zap.String("character_id", ctx.CharacterID))
	return map[string]interface{}{
		"success": true,
	}, nil
}

func (h *Handler) WatchBattle(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}

	var battleID string
	switch v := args[0].(type) {
	case string:
		battleID = v
	default:
		return nil, pkgerrors.ErrInvalidArgs
	}

	if battleID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "missing_battle_id",
		}, nil
	}

	battle, err := h.combatService.GetBattle(battleID)
	if err != nil {
		h.logger.Warn("WatchBattle: battle not found",
			zap.String("battle_id", battleID),
			zap.String("character_id", ctx.CharacterID),
			zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}

	battle.AddWatcher(ctx.Connection)

	h.logger.Info("WatchBattle: watcher added",
		zap.String("battle_id", battleID),
		zap.String("character_id", ctx.CharacterID))

	return map[string]interface{}{
		"success":    true,
		"battleId":   battleID,
		"battleData": battle.ToDTO(),
	}, nil
}
