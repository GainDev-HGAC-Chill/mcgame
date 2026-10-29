// Farm harvest tasks own cancellation, completion, and committed callbacks.
package npc

import (
	"errors"
	"time"

	appfarm "mcgame-server/internal/application/farm"
	"mcgame-server/internal/infrastructure/rtmp"
)

type farmHarvestTask struct {
	characterID int64
	plotNPCID   int
	cancel      chan struct{}
}

func (h *Handler) replaceFarmHarvestTask(connID uint32, task *farmHarvestTask) *farmHarvestTask {
	h.farmHarvestMu.Lock()
	defer h.farmHarvestMu.Unlock()
	prev := h.farmHarvests[connID]
	h.farmHarvests[connID] = task
	return prev
}

func (h *Handler) completeFarmHarvestTask(connID uint32, task *farmHarvestTask) bool {
	h.farmHarvestMu.Lock()
	defer h.farmHarvestMu.Unlock()
	current, ok := h.farmHarvests[connID]
	if !ok || current != task {
		return false
	}
	delete(h.farmHarvests, connID)
	return true
}

func (h *Handler) cancelFarmHarvestTask(connID uint32) *farmHarvestTask {
	h.farmHarvestMu.Lock()
	defer h.farmHarvestMu.Unlock()
	task, ok := h.farmHarvests[connID]
	if !ok {
		return nil
	}
	delete(h.farmHarvests, connID)
	close(task.cancel)
	return task
}

func (h *Handler) startFarmHarvest(ctx *rtmp.RPCContext, characterID int64, ownerCharacterID int64, plotNPCID int) error {
	task := &farmHarvestTask{characterID: characterID, plotNPCID: plotNPCID, cancel: make(chan struct{})}
	if prev := h.replaceFarmHarvestTask(ctx.ConnID, task); prev != nil {
		close(prev.cancel)
	}
	if err := ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 27}); err != nil {
		h.cancelFarmHarvestTask(ctx.ConnID)
		return err
	}
	if err := ctx.Connection.SendCallback("onGathering", map[string]interface{}{"hint": " Đang thu hoạch", "delay": 5000}); err != nil {
		h.cancelFarmHarvestTask(ctx.ConnID)
		_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
		return err
	}
	go func() {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-task.cancel:
			return
		case <-timer.C:
		}
		if !h.completeFarmHarvestTask(ctx.ConnID, task) {
			return
		}
		result, err := h.farmService.HarvestPlot(ctx.Context, ownerCharacterID, characterID, plotNPCID)
		h.finishFarmHarvest(ctx, characterID, plotNPCID, result, err)
		_ = ctx.Connection.SendCallback("quitGathering", nil)
		_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
	}()
	return nil
}

func (h *Handler) QuitGatheringClinet(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if task := h.cancelFarmHarvestTask(ctx.ConnID); task != nil {
		_ = ctx.Connection.SendCallback("quitGathering", nil)
		_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
		return map[string]interface{}{"success": true, "plotId": task.plotNPCID}, nil
	}
	if task := h.cancelFishingTask(ctx.ConnID); task != nil {
		_ = ctx.Connection.SendCallback("quitGathering", nil)
		_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
		return map[string]interface{}{"success": true, "npcId": task.npcID}, nil
	}
	if task := h.cancelHerbTask(ctx.ConnID); task != nil {
		_ = ctx.Connection.SendCallback("quitGathering", nil)
		_ = ctx.Connection.SendCallback("onSetCharState", map[string]interface{}{"cid": ctx.CharacterID, "state": 1})
		return map[string]interface{}{"success": true, "npcId": task.npcID}, nil
	}
	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) finishFarmHarvest(ctx *rtmp.RPCContext, characterID int64, plotNPCID int, result *appfarm.HarvestResult, err error) {
	if err != nil {
		msg := "Không thể thu hoạch nông sản."
		systemMsg := ""
		switch {
		case errors.Is(err, appfarm.ErrPlotNotFound):
			msg = "Bạn chưa trồng."
		case errors.Is(err, appfarm.ErrCropNotReady):
			msg = "Nông sản chưa chín."
		case errors.Is(err, appfarm.ErrPlotAlreadyShared):
			systemMsg = "Bạn đã thu hoạch mảnh vườn này rồi, phải chừa lại cho chủ vườn 1 ít chứ"
		case errors.Is(err, appfarm.ErrPlotSharedOut):
			systemMsg = "Mảnh vườn này đã được người khác thu hoạch phụ rồi."
		}
		if systemMsg != "" {
			_ = ctx.Connection.SendCallback("onSystemSay", systemMsg)
			return
		}
		_ = ctx.Connection.SendCallback("onNpcMsg", plotNPCID, msg)
		return
	}
	if result != nil {
		_ = ctx.Connection.SendCallback("onAddItem", h.buildFarmRewardEffect(result))
		if result.AddedInventoryItem != nil {
			_ = ctx.Connection.SendCallback("onAddCharactorSlot", result.AddedInventoryItem.ToDTO())
		}
	}
	if result == nil || !result.IsSharedHarvest {
		h.sendFarmPlotReset(ctx, plotNPCID)
	}
	if result != nil {
		h.sendFarmDataUpdate(ctx, result.OwnerCharacterID)
		return
	}
	h.sendFarmDataUpdate(ctx, characterID)
}
