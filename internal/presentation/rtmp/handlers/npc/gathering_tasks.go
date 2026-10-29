// Gathering tasks own cancellable fishing and herb completion lifecycles.
package npc

import (
	"context"
	"errors"
	"time"

	appgathering "mcgame-server/internal/application/gathering"
	domaingathering "mcgame-server/internal/domain/gathering"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
	pkgerrors "mcgame-server/pkg/errors"
)

type fishingTask struct {
	characterID int64
	npcID       int
	command     domaingathering.CompletionCommand
	cancel      chan struct{}
}

type herbTask struct {
	characterID int64
	npcID       int
	command     domaingathering.CompletionCommand
	cancel      chan struct{}
}

func (h *Handler) startHerbGathering(ctx *rtmp.RPCContext, characterID int64, npc *models.NpcTemplate) error {
	if h.gatheringService == nil {
		return pkgerrors.ErrSystemError
	}
	placementAuthorized := h.herbPlacementAuthorized(ctx, characterID, npc.GetID())
	attempt, err := h.gatheringService.PrepareHerb(ctx.Context, characterID, npc.GetID(), placementAuthorized)
	if err != nil {
		h.sendGatheringStartError(ctx, err)
		return nil
	}
	if err := ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 28}); err != nil {
		return err
	}
	task := &herbTask{characterID: characterID, npcID: npc.GetID(), command: attempt.Completion, cancel: make(chan struct{})}
	if prev := h.replaceHerbTask(ctx.ConnID, task); prev != nil {
		close(prev.cancel)
	}
	if err := ctx.Connection.SendCallback("onGathering", map[string]interface{}{"hint": " Đang hái dược", "delay": attempt.Start.Delay.Milliseconds()}); err != nil {
		h.cancelHerbTask(ctx.ConnID)
		_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
		return err
	}
	go func() {
		timer := time.NewTimer(attempt.Start.Delay)
		defer timer.Stop()
		select {
		case <-task.cancel:
			return
		case <-timer.C:
		}
		if !h.completeHerbTask(ctx.ConnID, task) {
			return
		}
		commandCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		result, commandErr := h.gatheringService.Complete(commandCtx, task.command)
		cancel()
		h.finishGathering(ctx, result, commandErr)
	}()
	return nil
}

func (h *Handler) startFishing(ctx *rtmp.RPCContext, characterID int64, npc *models.NpcTemplate) error {
	if h.gatheringService == nil {
		return pkgerrors.ErrSystemError
	}
	attempt, err := h.gatheringService.PrepareFishing(ctx.Context, characterID, npc.GetID())
	if err != nil {
		h.sendGatheringStartError(ctx, err)
		return nil
	}
	if err := ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 26}); err != nil {
		return err
	}
	task := &fishingTask{characterID: characterID, npcID: npc.GetID(), command: attempt.Completion, cancel: make(chan struct{})}
	if prev := h.replaceFishingTask(ctx.ConnID, task); prev != nil {
		close(prev.cancel)
	}
	if err := ctx.Connection.SendCallback("onGathering", map[string]interface{}{"hint": " Đang câu cá", "delay": attempt.Start.Delay.Milliseconds()}); err != nil {
		h.cancelFishingTask(ctx.ConnID)
		_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
		return err
	}
	go func() {
		timer := time.NewTimer(attempt.Start.Delay)
		defer timer.Stop()
		select {
		case <-task.cancel:
			return
		case <-timer.C:
		}
		if !h.completeFishingTask(ctx.ConnID, task) {
			return
		}
		commandCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		result, commandErr := h.gatheringService.Complete(commandCtx, task.command)
		cancel()
		h.finishGathering(ctx, result, commandErr)
	}()
	return nil
}

func (h *Handler) herbPlacementAuthorized(ctx *rtmp.RPCContext, characterID int64, npcID int) bool {
	if h.sceneService == nil || h.charService == nil {
		return false
	}
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil || char == nil || char.MapID == 200 {
		return false
	}
	npcs, err := h.sceneService.GetMapNPCsForClient(ctx.Context, char.MapID)
	if err != nil {
		return false
	}
	for _, mapNPC := range npcs {
		if mapNPC.NID == npcID && mapNPC.Type == 23 {
			return true
		}
	}
	return false
}

func (h *Handler) sendGatheringStartError(ctx *rtmp.RPCContext, err error) {
	switch {
	case errors.Is(err, appgathering.ErrSkillLevelNotEnough):
		_ = ctx.Connection.SendCallback("onSystemSay", "Cấp độ kỹ năng không đủ")
	case errors.Is(err, appgathering.ErrToolNotEquipped), errors.Is(err, appgathering.ErrToolInvalid):
		_ = ctx.Connection.SendCallback("onSystemSay", "Bạn chưa trang bị cần câu phù hợp")
	case errors.Is(err, appgathering.ErrToolBroken):
		_ = ctx.Connection.SendCallback("onSystemSay", "Cần câu đã hỏng")
	case errors.Is(err, appgathering.ErrVigorNotEnough):
		_ = ctx.Connection.SendCallback("onSystemMsgCode", 20)
	default:
		_ = ctx.Connection.SendCallback("onSystemSay", "Tài nguyên này hiện chưa thể thu thập")
	}
}

func (h *Handler) finishGathering(ctx *rtmp.RPCContext, result *domaingathering.CompletionResult, commandErr error) {
	if commandErr == nil && result != nil {
		if result.Succeeded && result.Reward != nil {
			h.sendCommittedGatheringReward(ctx, result.Reward)
		}
		if result.Character != nil && h.skillService != nil {
			rtmputils.SendLifeSkillStateUpdate(ctx, h.logger, h.skillService, result.Character)
		}
	}
	_ = ctx.Connection.SendCallback("quitGathering", nil)
	_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
}

func (h *Handler) sendCommittedGatheringReward(ctx *rtmp.RPCContext, reward *domainitem.Item) {
	name := h.getItemTemplateName(reward.TemplateID)
	effectData := map[string]interface{}{"s": 1, "tt": "6", "i": reward.TemplateID, "c": "0", "t": 29, "n": name, "q": 0}
	slotData := map[string]interface{}{"c": "0", "b": 0, "t": reward.TemplateID, "n": reward.StackCount}
	_ = ctx.Connection.SendCallback("onAddTempDirect", effectData, reward.SlotIndex+1, slotData, h.countTempBagItems(ctx, reward.CharacterID))
}

func (h *Handler) replaceFishingTask(connID uint32, task *fishingTask) *fishingTask {
	h.fishingTaskMu.Lock()
	defer h.fishingTaskMu.Unlock()
	prev := h.fishingTasks[connID]
	h.fishingTasks[connID] = task
	return prev
}

func (h *Handler) completeFishingTask(connID uint32, task *fishingTask) bool {
	h.fishingTaskMu.Lock()
	defer h.fishingTaskMu.Unlock()
	current, ok := h.fishingTasks[connID]
	if !ok || current != task {
		return false
	}
	delete(h.fishingTasks, connID)
	return true
}

func (h *Handler) cancelFishingTask(connID uint32) *fishingTask {
	h.fishingTaskMu.Lock()
	defer h.fishingTaskMu.Unlock()
	task, ok := h.fishingTasks[connID]
	if !ok {
		return nil
	}
	delete(h.fishingTasks, connID)
	close(task.cancel)
	return task
}

func (h *Handler) replaceHerbTask(connID uint32, task *herbTask) *herbTask {
	h.herbTaskMu.Lock()
	defer h.herbTaskMu.Unlock()
	prev := h.herbTasks[connID]
	h.herbTasks[connID] = task
	return prev
}

func (h *Handler) completeHerbTask(connID uint32, task *herbTask) bool {
	h.herbTaskMu.Lock()
	defer h.herbTaskMu.Unlock()
	current, ok := h.herbTasks[connID]
	if !ok || current != task {
		return false
	}
	delete(h.herbTasks, connID)
	return true
}

func (h *Handler) cancelHerbTask(connID uint32) *herbTask {
	h.herbTaskMu.Lock()
	defer h.herbTaskMu.Unlock()
	task, ok := h.herbTasks[connID]
	if !ok {
		return nil
	}
	delete(h.herbTasks, connID)
	close(task.cancel)
	return task
}
