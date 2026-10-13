import "server-only"

import { createServerClient, parseCookieHeader, serializeCookieHeader } from "@supabase/ssr"
import { createClient } from "@supabase/supabase-js"

import {
    authorizeQuestUser,
    QuestAuthError,
    readQuestBearer,
    requireQuestSameOrigin,
    type QuestIdentity,
    type QuestPermission,
} from "@/lib/quests/permissions"
import { getSupabasePublishableKey, getSupabasePublicUrl, getSupabaseServerUrl } from "@/lib/supabase/env"

export { QuestAuthError } from "@/lib/quests/permissions"

export interface QuestAdminIdentity extends QuestIdentity {
    responseHeaders: Headers
}

// Route handlers must attach responseHeaders to every response, including errors.
export async function requireQuestPermission(
    request: Request,
    permission: QuestPermission
): Promise<QuestAdminIdentity> {
    const responseHeaders = new Headers({ "Cache-Control": "private, no-store" })
    try {
        const bearer = readQuestBearer(request.headers)
        const url = getSupabaseServerUrl()
        const key = getSupabasePublishableKey()
        if (!url || !key) {
            throw new QuestAuthError(503, "quest_auth_unavailable", "Dịch vụ xác thực chưa sẵn sàng.")
        }
        const supabase = bearer
            ? createClient(url, key, {
                auth: { persistSession: false, autoRefreshToken: false, detectSessionInUrl: false },
            })
            : createServerClient(url, key, {
                cookieOptions: {
                    name: `sb-${new URL(getSupabasePublicUrl() || url).hostname.split(".")[0]}-auth-token`,
                },
                cookies: {
                    getAll() {
                        return parseCookieHeader(request.headers.get("Cookie") ?? "")
                            .map(({ name, value }) => ({ name, value: value ?? "" }))
                    },
                    setAll(cookiesToSet, cacheHeaders) {
                        cookiesToSet.forEach(({ name, value, options }) => {
                            responseHeaders.append("Set-Cookie", serializeCookieHeader(name, value, options))
                        })
                        Object.entries(cacheHeaders ?? {}).forEach(([name, value]) => {
                            responseHeaders.set(name, value)
                        })
                        responseHeaders.set("Cache-Control", "private, no-store")
                    },
                },
            })
        const { data, error } = await supabase.auth.getUser(bearer)
        if (error) {
            throw new QuestAuthError(401, "quest_unauthenticated", "Phiên đăng nhập không hợp lệ.")
        }
        const identity = authorizeQuestUser(data.user, permission)
        if (permission === "write" || !["GET", "HEAD", "OPTIONS"].includes(request.method)) {
            requireQuestSameOrigin(request)
        }
        return { ...identity, responseHeaders }
    } catch (error) {
        const authError = error instanceof QuestAuthError
            ? error
            : new QuestAuthError(503, "quest_auth_unavailable", "Dịch vụ xác thực chưa sẵn sàng.")
        responseHeaders.forEach((value, name) => authError.responseHeaders.set(name, value))
        const cookies = responseHeaders.getSetCookie()
        if (cookies.length) {
            authError.responseHeaders.delete("Set-Cookie")
            cookies.forEach(value => authError.responseHeaders.append("Set-Cookie", value))
        }
        throw authError
    }
}

export function applyQuestAuthHeaders(
    response: Response,
    auth: QuestAdminIdentity | QuestAuthError
): Response {
    auth.responseHeaders.forEach((value, name) => {
        if (name !== "set-cookie") response.headers.set(name, value)
    })
    auth.responseHeaders.getSetCookie().forEach(value => response.headers.append("Set-Cookie", value))
    response.headers.set("Cache-Control", "private, no-store")
    return response
}

export function questAuthErrorResponse(error: QuestAuthError): Response {
    return applyQuestAuthHeaders(Response.json(
        { error: { code: error.code, message: error.message } },
        { status: error.status }
    ), error)
}
