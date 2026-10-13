import { questListQuerySchema } from "@/lib/quests/validation"
import { questRoute, questRpc } from "@/lib/server/quest-admin"
import type { QuestListResponse } from "@/types/quests"
export const dynamic = "force-dynamic"
export async function GET(request: Request) {
    return questRoute(request,"read",async auth => {
        const query = questListQuerySchema.parse(Object.fromEntries(new URL(request.url).searchParams))
        const data = await questRpc<QuestListResponse>("quest_admin_list",{p_search:query.search,p_id:query.id ?? null,p_type:query.type ?? null,p_min_level:query.minLevel ?? null,p_max_level:query.maxLevel ?? null,p_page:query.page,p_page_size:query.pageSize})
        return {...data,canWrite:auth.canWrite}
    })
}
