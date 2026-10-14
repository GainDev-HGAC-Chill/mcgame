import "server-only"

import {
    QuestAuthError,
    requireQuestSameOrigin,
    type QuestIdentity,
    type QuestPermission,
} from "@/lib/quests/permissions"

export { QuestAuthError } from "@/lib/quests/permissions"

export interface QuestAdminIdentity extends QuestIdentity {
    responseHeaders: Headers
}

const localQuestEditorID = "00000000-0000-4000-8000-000000000001"

// Route handlers must attach responseHeaders to every response, including errors.
export async function requireQuestPermission(
    request: Request,
    _permission: QuestPermission
): Promise<QuestAdminIdentity> {
    const responseHeaders = new Headers({ "Cache-Control": "private, no-store" })
    if (!["GET", "HEAD", "OPTIONS"].includes(request.method)) {
        requireQuestSameOrigin(request)
    }
    return {
        userId: localQuestEditorID,
        email: null,
        canWrite: true,
        responseHeaders,
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
