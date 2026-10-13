import { questRoute } from "@/lib/server/quest-admin"
export const dynamic = "force-dynamic"
export async function GET(request: Request) {
    return questRoute(request,"read",async auth => ({canRead:true,canWrite:auth.canWrite}))
}
