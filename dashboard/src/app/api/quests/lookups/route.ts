import { questListQuerySchema, questLookupKindSchema } from "@/lib/quests/validation"
import { questRoute, questRpc } from "@/lib/server/quest-admin"
import type { QuestLookupResponse } from "@/types/quests"
export const dynamic = "force-dynamic"
export async function GET(request: Request) {
    return questRoute(request,"read",async () => {
        const {kind,...raw} = Object.fromEntries(new URL(request.url).searchParams)
        const query = questListQuerySchema.pick({search:true,page:true,pageSize:true}).parse(raw)
        return questRpc<QuestLookupResponse>("quest_admin_lookups",{p_kind:questLookupKindSchema.parse(kind),p_search:query.search,p_page:query.page,p_page_size:query.pageSize})
    })
}
