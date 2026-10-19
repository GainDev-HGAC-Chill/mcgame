import { npcSaveSchema } from "@/lib/npcs/validation"
import {
    decorateNpc,
    npcRoute,
    npcRpc,
    parseNpcPathId,
    readNpcBody,
} from "@/lib/server/npc-admin"
import type { NpcDetail } from "@/types/npcs"

export const dynamic = "force-dynamic"
type Context = { params: Promise<{ id: string }> }

export async function GET(request: Request, context: Context) {
    return npcRoute(
        request,
        "read",
        async auth =>
            await decorateNpc(
                await npcRpc<NpcDetail>("npc_admin_detail", {
                    p_id: parseNpcPathId((await context.params).id),
                }),
                auth.canWrite
            )
    )
}

export async function PATCH(request: Request, context: Context) {
    return npcRoute(request, "write", async auth => {
        const body = npcSaveSchema.parse(await readNpcBody(request))
        const data = await npcRpc<NpcDetail>("npc_admin_save", {
            p_id: parseNpcPathId((await context.params).id),
            p_expected_revision: body.expectedRevision,
            p_expected_shared_count: body.expectedSharedCount,
            p_confirm_shared: body.confirmShared,
            p_patch: body.patch,
            p_reason: body.reason,
            p_request_id: body.requestId,
            p_actor_id: auth.userId,
        })
        return await decorateNpc(data, auth.canWrite, true)
    })
}
