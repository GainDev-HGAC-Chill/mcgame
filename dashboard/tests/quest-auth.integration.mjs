// Opt-in deployed-API checks; intentionally not named *.test.mjs.

class SafeFailure extends Error {}

const requireCheck = (condition, message) => {
    if (!condition) throw new SafeFailure(message)
}

function dashboardOrigin() {
    let url
    try {
        url = new URL(process.env.QUEST_ADMIN_DASHBOARD_URL ?? "")
    } catch {
        throw new SafeFailure("Set QUEST_ADMIN_DASHBOARD_URL to the deployed loopback dashboard origin.")
    }
    requireCheck(
        ["localhost", "127.0.0.1"].includes(url.hostname) &&
            ["http:", "https:"].includes(url.protocol) &&
            !url.username &&
            !url.password &&
            url.pathname === "/" &&
            !url.search &&
            !url.hash,
        "Dashboard URL must be an http(s) localhost/127.0.0.1 origin without credentials or paths."
    )
    return url.origin
}

async function main() {
    requireCheck(
        process.env.QUEST_ADMIN_INTEGRATION === "1",
        "Not run: set QUEST_ADMIN_INTEGRATION=1 only after owner approval and dashboard deploy."
    )
    requireCheck(
        process.env.QUEST_ADMIN_API_CONTRACT_REVIEWED === "1",
        "Not run: confirm the deployed API paths/response contract, then set QUEST_ADMIN_API_CONTRACT_REVIEWED=1."
    )
    const origin = dashboardOrigin()
    const id = process.env.QUEST_ADMIN_QUEST_ID
    requireCheck(
        /^[1-9][0-9]*$/.test(id ?? "") && Number.isSafeInteger(Number(id)),
        "Set QUEST_ADMIN_QUEST_ID to a reviewed existing positive quest ID."
    )
    const detailPath = `/api/quests/${id}`
    const localFetch = async (path, init = {}) => {
        const url = new URL(path, origin)
        requireCheck(url.origin === origin, "Refused a request outside the approved dashboard origin.")
        return fetch(url, {
            ...init,
            cache: "no-store",
            redirect: "error",
            signal: AbortSignal.timeout(15000),
        })
    }
    const expectStatus = async (path, status, init) => {
        const response = await localFetch(path, init)
        requireCheck(response.status === status, `Expected HTTP ${status}, received ${response.status}.`)
        return response
    }
    const json = async response => {
        try {
            return await response.json()
        } catch {
            throw new SafeFailure("Expected a JSON response.")
        }
    }

    await json(await expectStatus("/api/quests?page=1&pageSize=1", 200))
    const access = await json(await expectStatus("/api/quests/access", 200))
    requireCheck(
        (access?.canWrite ?? access?.data?.canWrite) === true,
        "Anonymous dashboard access must expose canWrite=true."
    )
    await json(await expectStatus("/api/quests/lookups", 200))
    const before = await json(await expectStatus(detailPath, 200))
    requireCheck(
        typeof before?.revision === "string" && before.canWrite === true,
        "Anonymous quest detail must include revision and canWrite=true."
    )
    await expectStatus(detailPath, 422, {
        method: "PATCH",
        headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
            Origin: origin,
        },
        body: "{",
    })
    await expectStatus(detailPath, 403, {
        method: "PATCH",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: "{",
    })
    const after = await json(await expectStatus(detailPath, 200))
    requireCheck(
        before.revision === after.revision &&
            JSON.stringify([before.quest, before.requirements, before.awards, before.prerequisites]) ===
                JSON.stringify([after.quest, after.requirements, after.awards, after.prerequisites]),
        "Rejected PATCH requests changed the quest catalog."
    )
    console.log("PASS: anonymous Quest Admin API access and same-origin write guard.")
}

main().catch(error => {
    console.error(error instanceof SafeFailure ? error.message : "Integration check failed; raw error suppressed.")
    process.exitCode = 1
})
