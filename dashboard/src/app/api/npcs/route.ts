import { npcListQuerySchema } from "@/lib/npcs/validation"
import { npcRoute, npcRpc } from "@/lib/server/npc-admin"
import type { NpcListResponse } from "@/types/npcs"

export const dynamic = "force-dynamic"

export async function GET(request: Request) {
    return npcRoute(request, "read", async auth => {
        const query = npcListQuerySchema.parse(
            Object.fromEntries(new URL(request.url).searchParams)
        )
        const data = await npcRpc<NpcListResponse>("npc_admin_list", {
            p_search: query.search,
            p_map_id: query.mapId ?? null,
            p_type: query.type ?? null,
            p_has_dialogue: query.hasDialogue ?? null,
            p_has_shop: query.hasShop ?? null,
            p_page: query.page,
            p_page_size: query.pageSize,
        })
        return { ...data, canWrite: auth.canWrite }
    })
}
