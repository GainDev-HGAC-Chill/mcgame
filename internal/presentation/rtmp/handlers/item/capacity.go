package item

import "mcgame-server/internal/infrastructure/rtmp"

func sendCapacityUpdate(ctx *rtmp.RPCContext, result map[string]interface{}) {
	if updates, ok := result["capacity_updates"].(map[string]interface{}); ok && len(updates) > 0 {
		ctx.Connection.SendCallback("onUPP", updates)
	}
}
