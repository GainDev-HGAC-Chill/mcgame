import assert from "node:assert/strict"
import { test } from "node:test"
import {
    authorizeQuestUser,
    QuestAuthError,
    readQuestBearer,
    requireQuestSameOrigin,
} from "../src/lib/quests/permissions.ts"

const user = (app_metadata = {}, user_metadata = {}) => ({
    id: "verified-user",
    app_metadata,
    user_metadata,
})
const denied = (operation, status) =>
    assert.throws(
        operation,
        error => error instanceof QuestAuthError && error.status === status
    )

test("unauthenticated identities fail closed", () => {
    for (const identity of [null, undefined, {}, { id: "" }]) {
        denied(() => authorizeQuestUser(identity, "read"), 401)
        denied(() => authorizeQuestUser(identity, "write"), 401)
    }
})

test("wrong roles and malformed permission metadata fail closed", () => {
    for (const metadata of [
        null,
        [],
        { role: "admin" },
        { admin_role: "owner" },
        { quest_role: "ADMIN" },
        { permissions: "quests:write" },
        { permissions: ["quests:*", {}] },
    ]) {
        denied(() => authorizeQuestUser(user(metadata), "read"), 403)
        denied(() => authorizeQuestUser(user(metadata), "write"), 403)
    }
})

test("trusted admin/editor roles grant read and write through either role key", () => {
    for (const key of ["admin_role", "quest_role"]) {
        for (const role of ["admin", "quest_editor"]) {
            for (const permission of ["read", "write"]) {
                assert.deepEqual(
                    authorizeQuestUser(user({ [key]: role }), permission),
                    { userId: "verified-user", email: null, canWrite: true }
                )
            }
        }
    }
})

test("viewer roles grant read only", () => {
    for (const key of ["admin_role", "quest_role"]) {
        const identity = user({ [key]: "quest_viewer" })
        assert.equal(authorizeQuestUser(identity, "read").canWrite, false)
        denied(() => authorizeQuestUser(identity, "write"), 403)
    }
})

test("trusted permission arrays grant exact scopes; write includes read", () => {
    const reader = user({ permissions: ["quests:read"] })
    assert.equal(authorizeQuestUser(reader, "read").canWrite, false)
    denied(() => authorizeQuestUser(reader, "write"), 403)
    for (const permission of ["read", "write"]) {
        assert.equal(
            authorizeQuestUser(
                user({ permissions: ["quests:write"] }),
                permission
            ).canWrite,
            true
        )
    }
})

test("user-editable metadata cannot escalate any role or scope", () => {
    const escalation = {
        admin_role: "admin",
        quest_role: "quest_editor",
        permissions: ["quests:read", "quests:write"],
    }
    denied(() => authorizeQuestUser(user({}, escalation), "read"), 403)
    denied(
        () =>
            authorizeQuestUser(
                user({ quest_role: "quest_viewer" }, escalation),
                "write"
            ),
        403
    )
})

test("write origin requires exact origin and rejects cross-site fetch hints", () => {
    const request = (origin, site) =>
        new Request("https://dashboard.example/api/quests", {
            headers: {
                ...(origin === undefined ? {} : { Origin: origin }),
                ...(site ? { "Sec-Fetch-Site": site } : {}),
            },
        })
    requireQuestSameOrigin(request("https://dashboard.example", "same-origin"))
    requireQuestSameOrigin(request("https://dashboard.example"))
    for (const origin of [
        undefined,
        "null",
        "https://evil.example",
        "http://dashboard.example",
        "https://dashboard.example:444",
        "https://dashboard.example/path",
    ]) {
        denied(() => requireQuestSameOrigin(request(origin)), 403)
    }
    for (const site of ["cross-site", "same-site", "none"]) {
        denied(
            () =>
                requireQuestSameOrigin(
                    request("https://dashboard.example", site)
                ),
            403
        )
    }
})

test("quest writes grant the same origin contract to local and VPN dashboard hosts", () => {
    for (const origin of [
        "http://localhost:3000",
        "http://127.0.0.1:3000",
        "http://26.31.111.123:3000",
    ]) {
        assert.doesNotThrow(() =>
            requireQuestSameOrigin(
                new Request(`${origin}/api/quests`, {
                    method: "PATCH",
                    headers: {
                        Origin: origin,
                        "Sec-Fetch-Site": "same-origin",
                    },
                })
            )
        )
    }
})

test("bearer tokens are optional, bounded, and malformed headers never fall back", () => {
    assert.equal(readQuestBearer(new Headers()), undefined)
    assert.equal(
        readQuestBearer(
            new Headers({ Authorization: "Bearer eyJ.test.signature" })
        ),
        "eyJ.test.signature"
    )
    for (const value of [
        "Basic secret",
        "Bearer",
        "Bearer a b",
        "Bearer a,b",
        "Bearer " + "a".repeat(8192),
    ]) {
        denied(
            () => readQuestBearer(new Headers({ Authorization: value })),
            401
        )
    }
})
