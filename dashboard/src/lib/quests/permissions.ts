export type QuestPermission = "read" | "write"

export interface VerifiedQuestUser {
    id: string
    email?: string
    app_metadata?: unknown
    user_metadata?: unknown
}

export interface QuestIdentity {
    userId: string
    email: string | null
    canWrite: boolean
}

export class QuestAuthError extends Error {
    readonly status: 401 | 403 | 503
    readonly code: string
    readonly responseHeaders = new Headers({ "Cache-Control": "private, no-store" })

    constructor(status: 401 | 403 | 503, code: string, message: string) {
        super(message)
        this.name = "QuestAuthError"
        this.status = status
        this.code = code
    }
}

function metadataRecord(value: unknown): Record<string, unknown> {
    return value !== null && typeof value === "object" && !Array.isArray(value)
        ? value as Record<string, unknown>
        : {}
}

// Only call with the user returned by Supabase Auth getUser(), never cookie/JWT decoding.
export function authorizeQuestUser(
    user: VerifiedQuestUser | null | undefined,
    permission: QuestPermission
): QuestIdentity {
    if (!user?.id || typeof user.id !== "string") {
        throw new QuestAuthError(401, "quest_unauthenticated", "Đăng nhập để quản lý nhiệm vụ.")
    }
    const metadata = metadataRecord(user.app_metadata)
    const roles = [metadata.admin_role, metadata.quest_role]
    const permissions = Array.isArray(metadata.permissions) ? metadata.permissions : []
    const canWrite = roles.some(role => role === "admin" || role === "quest_editor") ||
        permissions.includes("quests:write")
    const canRead = canWrite || roles.includes("quest_viewer") || permissions.includes("quests:read")
    if (!canRead || (permission === "write" && !canWrite)) {
        throw new QuestAuthError(403, "quest_forbidden", "Không có quyền truy cập nhiệm vụ.")
    }
    return { userId: user.id, email: user.email ?? null, canWrite }
}

export function requireQuestSameOrigin(request: Pick<Request, "url" | "headers">): void {
    const origin = request.headers.get("Origin")
    const fetchSite = request.headers.get("Sec-Fetch-Site")
    let valid = false
    try {
        const parsed = new URL(origin ?? "")
        // Exact origins only. Do not blindly trust client-controlled X-Forwarded-Host.
        const allowed = (process.env.DASHBOARD_ALLOWED_ORIGINS ?? "")
            .split(",")
            .map(value => value.trim())
            .filter(Boolean)
        valid = parsed.origin === origin &&
            (parsed.protocol === "http:" || parsed.protocol === "https:") &&
            (origin === new URL(request.url).origin || allowed.includes(origin!))
    } catch {
        valid = false
    }
    // A browser calling the same-origin /api/... endpoint should send same-origin.
    if (!valid || (fetchSite !== null && fetchSite !== "same-origin")) {
        throw new QuestAuthError(403, "quest_origin_forbidden", "Yêu cầu phải đến từ cùng nguồn.")
    }
}

export function readQuestBearer(headers: Headers): string | undefined {
    const authorization = headers.get("Authorization")
    if (authorization === null) return undefined
    if (authorization.length > 8192 || !/^Bearer [A-Za-z0-9._~+\/-]+=*$/i.test(authorization)) {
        throw new QuestAuthError(401, "quest_invalid_bearer", "Thông tin đăng nhập không hợp lệ.")
    }
    return authorization.slice(7)
}
