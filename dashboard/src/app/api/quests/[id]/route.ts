import { questSaveSchema } from "@/lib/quests/validation"
import { decorateQuest, parseQuestPathId, questRoute, questRpc, readQuestBody } from "@/lib/server/quest-admin"
import type { QuestDetail } from "@/types/quests"
export const dynamic = "force-dynamic"
type Context = { params: Promise<{id:string}> }
export async function GET(request: Request, context: Context) {
    return questRoute(request,"read",async auth => decorateQuest(await questRpc<QuestDetail>("quest_admin_detail",{p_id:parseQuestPathId((await context.params).id)}),auth.canWrite))
}
export async function PATCH(request: Request, context: Context) {
    return questRoute(request,"write",async auth => {
        const body = questSaveSchema.parse(await readQuestBody(request))
        const data = await questRpc<QuestDetail>("quest_admin_save",{p_id:parseQuestPathId((await context.params).id),p_expected_revision:body.expectedRevision,p_patch:body.patch,p_reason:body.reason,p_request_id:body.requestId,p_actor_id:auth.userId})
        return decorateQuest(data,auth.canWrite,true)
    })
}
