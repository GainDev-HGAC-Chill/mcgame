import { questLoopSaveSchema } from "@/lib/quests/validation"
import { decorateLoop, parseQuestPathId, questRoute, questRpc, readQuestBody } from "@/lib/server/quest-admin"
import type { QuestLoopDetail } from "@/types/quests"
export const dynamic = "force-dynamic"
type Context = { params: Promise<{id:string}> }
export async function GET(request: Request, context: Context) {
    return questRoute(request,"read",async auth => decorateLoop(await questRpc<QuestLoopDetail>("quest_admin_loop_detail",{p_id:parseQuestPathId((await context.params).id)}),auth.canWrite))
}
export async function PATCH(request: Request, context: Context) {
    return questRoute(request,"write",async auth => {
        const body = questLoopSaveSchema.parse(await readQuestBody(request))
        const data = await questRpc<QuestLoopDetail>("quest_admin_save_loop",{p_id:parseQuestPathId((await context.params).id),p_expected_revision:body.expectedRevision,p_expected_affected_count:body.expectedAffectedCount,p_confirm_shared:body.confirmShared,p_patch:body.patch,p_reason:body.reason,p_request_id:body.requestId,p_actor_id:auth.userId})
        return decorateLoop(data,auth.canWrite,true)
    })
}
