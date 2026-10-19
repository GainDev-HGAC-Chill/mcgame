import { npcLookupQuerySchema } from "@/lib/npcs/validation"
import { npcLookup, npcRoute } from "@/lib/server/npc-admin"

export const dynamic = "force-dynamic"

export async function GET(request: Request) {
    return npcRoute(request, "read", async () => {
        const query = npcLookupQuerySchema.parse(
            Object.fromEntries(new URL(request.url).searchParams)
        )
        return npcLookup(query)
    })
}
