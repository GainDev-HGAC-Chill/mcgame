// Opt-in deployed-API checks; intentionally not named *.test.mjs.
import { randomBytes, randomUUID } from "node:crypto"
import { mkdir, mkdtemp, readFile, unlink, writeFile } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { createClient } from "@supabase/supabase-js"

class SafeFailure extends Error {}
const requireCheck = (condition, message) => {
    if (!condition) throw new SafeFailure(message)
}

function parseEnv(text) {
    const values = Object.create(null)
    for (const line of text.split(/\r?\n/)) {
        const match = line.match(/^\s*(?:export\s+)?([A-Z][A-Z0-9_]*)\s*=\s*(.*?)\s*$/)
        if (!match) continue
        let value = match[2]
        if ((value.startsWith('"') && value.endsWith('"')) ||
            (value.startsWith("'") && value.endsWith("'"))) {
            value = value.slice(1, -1)
        } else {
            value = value.replace(/\s+#.*$/, "").trim()
        }
        values[match[1]] = value
    }
    return values
}

function dashboardOrigin() {
    let url
    try { url = new URL(process.env.QUEST_ADMIN_DASHBOARD_URL ?? "") } catch {
        throw new SafeFailure("Set QUEST_ADMIN_DASHBOARD_URL to the deployed loopback dashboard origin.")
    }
    requireCheck(["localhost", "127.0.0.1"].includes(url.hostname) &&
        ["http:", "https:"].includes(url.protocol) && !url.username && !url.password &&
        url.pathname === "/" && !url.search && !url.hash,
    "Dashboard URL must be an http(s) localhost/127.0.0.1 origin without credentials or paths.")
    return url.origin
}

function apiPath(value, name) {
    requireCheck(typeof value === "string" && /^\/api\/quests(?:\/|\?|$)/.test(value) &&
        !value.includes("\\") && !value.includes("#"), `Set ${name} to a reviewed /api/quests path.`)
    const parsed = new URL(value, "http://127.0.0.1")
    requireCheck(parsed.pathname === "/api/quests" || parsed.pathname.startsWith("/api/quests/"),
        `${name} must stay inside /api/quests.`)
    return parsed.pathname + parsed.search
}

async function main() {
    requireCheck(process.env.QUEST_ADMIN_INTEGRATION === "1",
        "Not run: set QUEST_ADMIN_INTEGRATION=1 only after owner approval and production deploy.")
    requireCheck(process.env.QUEST_ADMIN_AUTH_TRIGGERS_REVIEWED === "1",
        "Not run: review live auth insert/delete triggers and hooks, then set QUEST_ADMIN_AUTH_TRIGGERS_REVIEWED=1.")
    requireCheck(process.env.QUEST_ADMIN_API_CONTRACT_REVIEWED === "1",
        "Not run: confirm the deployed API paths/response contract, then set QUEST_ADMIN_API_CONTRACT_REVIEWED=1.")
    const origin = dashboardOrigin()
    const lookupPath = apiPath(process.env.QUEST_ADMIN_LOOKUP_PATH, "QUEST_ADMIN_LOOKUP_PATH")
    const id = process.env.QUEST_ADMIN_QUEST_ID
    requireCheck(/^[1-9][0-9]*$/.test(id ?? "") && Number.isSafeInteger(Number(id)),
        "Set QUEST_ADMIN_QUEST_ID to a reviewed existing positive quest ID.")
    const detailPath = `/api/quests/${id}`
    const accessPath = apiPath(process.env.QUEST_ADMIN_ACCESS_PATH ?? "/api/quests/access", "QUEST_ADMIN_ACCESS_PATH")
    const listPath = "/api/quests?page=1&pageSize=1"
    const env = parseEnv(await readFile(new URL("../../docker/.env", import.meta.url), "utf8"))
    const port = env.API_GW_HTTP_PORT || env.KONG_HTTP_PORT || "8000"
    requireCheck(/^[0-9]+$/.test(port) && Number(port) >= 1 && Number(port) <= 65535,
        "Compose Supabase gateway port is invalid.")
    const supabaseUrl = `http://127.0.0.1:${Number(port)}`
    const publicKey = env.SUPABASE_PUBLISHABLE_KEY || env.ANON_KEY
    requireCheck(Boolean(publicKey && env.SERVICE_ROLE_KEY), "Compose Supabase auth keys are missing.")

    // Never follow redirects with bearer/service credentials, including SDK requests.
    const localFetch = async (input, init = {}) => {
        const url = new URL(typeof input === "string" || input instanceof URL ? input : input.url)
        requireCheck(url.origin === origin || url.origin === supabaseUrl,
            "Refused a request outside the approved loopback origins.")
        return fetch(input, { ...init, redirect: "error", signal: AbortSignal.timeout(15000) })
    }
    const newClient = key => createClient(supabaseUrl, key, {
        auth: { persistSession: false, autoRefreshToken: false, detectSessionInUrl: false },
        global: { fetch: localFetch },
    })
    const admin = newClient(env.SERVICE_ROLE_KEY)
    let stage = "anonymous API preflight"
    let interrupted = false
    const onSignal = () => { interrupted = true }
    process.on("SIGINT", onSignal)
    process.on("SIGTERM", onSignal)
    const checkInterrupted = () => requireCheck(!interrupted, "Interrupted; own-user cleanup attempted.")
    const request = async (path, token, method = "GET", withOrigin = true) => {
        checkInterrupted()
        const headers = { Accept: "application/json" }
        if (token) headers.Authorization = `Bearer ${token}`
        if (method === "PATCH") {
            headers["Content-Type"] = "application/json"
            if (withOrigin) headers.Origin = origin
        }
        const response = await localFetch(new URL(path, origin), {
            method, headers, cache: "no-store",
            // Invalid JSON by construction: never send a valid aggregate or save payload.
            ...(method === "PATCH" ? { body: "{" } : {}),
        })
        return response
    }
    const expectStatus = async (path, token, status, method = "GET", withOrigin = true) => {
        const response = await request(path, token, method, withOrigin)
        requireCheck(response.status === status, `${stage}: expected HTTP ${status}, received ${response.status}.`)
        return response
    }
    const json = async response => {
        try { return await response.json() } catch { throw new SafeFailure(`${stage}: expected JSON response.`) }
    }
    const own = []
    let journalPath
    let failure
    const cleanupFailures = []
    const journal = async () => writeFile(journalPath, JSON.stringify({
        purpose: "quest-admin-auth-integration", createdAt: startedAt,
        users: own.map(({ id, deleted }) => ({ id, deleted })),
    }, null, 2), { mode: 0o600 })
    const startedAt = new Date().toISOString()
    try {
        await expectStatus(listPath, undefined, 401)
        await expectStatus(accessPath, undefined, 401)
        // Create the private recovery journal before creating any user. No credentials are written.
        const base = process.platform === "win32" && process.env.LOCALAPPDATA
            ? join(process.env.LOCALAPPDATA, "Temp", "opencode") : tmpdir()
        await mkdir(base, { recursive: true, mode: 0o700 })
        const directory = await mkdtemp(join(base, "quest-auth-"))
        journalPath = join(directory, "created-user-ids.json")
        await journal()
        const roles = [
            { label: "forged", app: {}, user: { admin_role: "admin", quest_role: "quest_editor", permissions: ["quests:write"] } },
            { label: "viewer", app: { quest_role: "quest_viewer" }, user: {} },
            { label: "editor", app: { quest_role: "quest_editor" }, user: {} },
        ]
        for (const role of roles) {
            checkInterrupted()
            stage = `create isolated ${role.label} auth user`
            const email = `quest-auth-${role.label}-${randomUUID()}@example.com`
            const password = `Aa1!${randomBytes(32).toString("base64url")}`
            const { data, error } = await admin.auth.admin.createUser({
                email, password, email_confirm: true,
                app_metadata: role.app, user_metadata: role.user,
            })
            // Capture an ID even when the SDK also returns an error; never list/delete by prefix.
            if (data?.user?.id) {
                own.push({ id: data.user.id, label: role.label, deleted: false })
                await journal()
            }
            requireCheck(!error && data?.user?.id, `${stage} failed; details suppressed to protect credentials.`)
            const entry = own[own.length - 1]
            checkInterrupted()
            stage = `authenticate isolated ${role.label} user`
            entry.client = newClient(publicKey)
            const login = await entry.client.auth.signInWithPassword({ email, password })
            entry.token = login.data?.session?.access_token
            requireCheck(!login.error && entry.token, `${stage} failed; details suppressed.`)
            const verified = await entry.client.auth.getUser(entry.token)
            requireCheck(!verified.error && verified.data.user?.id === entry.id,
                `${stage}: server getUser verification failed.`)
            requireCheck(role.label === "forged"
                ? !verified.data.user.app_metadata?.quest_role && !verified.data.user.app_metadata?.admin_role &&
                    !verified.data.user.app_metadata?.permissions
                : verified.data.user.app_metadata?.quest_role === role.app.quest_role,
            `${stage}: unexpected trusted metadata; stop and review auth hooks.`)
        }
        const token = label => own.find(entry => entry.label === label).token
        stage = "forged user_metadata denial"
        await expectStatus(listPath, token("forged"), 403)
        stage = "viewer list read"
        await json(await expectStatus(listPath, token("viewer"), 200))
        stage = "viewer access read-only"
        const viewerAccess = await json(await expectStatus(accessPath, token("viewer"), 200))
        requireCheck((viewerAccess?.canWrite ?? viewerAccess?.data?.canWrite) === false,
            "Viewer access endpoint must expose canWrite=false.")
        stage = "viewer PATCH denial"
        await expectStatus(detailPath, token("viewer"), 403, "PATCH")
        stage = "editor list read"
        await json(await expectStatus(listPath, token("editor"), 200))
        stage = "editor access write capability"
        const editorAccess = await json(await expectStatus(accessPath, token("editor"), 200))
        requireCheck((editorAccess?.canWrite ?? editorAccess?.data?.canWrite) === true,
            "Editor access endpoint must expose canWrite=true.")
        stage = "editor catalog lookup"
        await json(await expectStatus(lookupPath, token("editor"), 200))
        stage = "editor detail read"
        const before = await json(await expectStatus(detailPath, token("editor"), 200))
        requireCheck(typeof before?.revision === "string" && before.canWrite === true,
            "Editor detail must include the canonical revision and canWrite=true.")
        stage = "editor malformed JSON rejection"
        await expectStatus(detailPath, token("editor"), 422, "PATCH")
        stage = "rejected write leaves catalog unchanged"
        const after = await json(await expectStatus(detailPath, token("editor"), 200))
        requireCheck(before.revision === after.revision &&
            JSON.stringify([before.quest, before.requirements, before.awards, before.prerequisites]) ===
            JSON.stringify([after.quest, after.requirements, after.awards, after.prerequisites]),
        "Malformed PATCH changed the quest catalog.")
        if (process.env.QUEST_ADMIN_TEST_ORIGIN === "1") {
            stage = "editor missing Origin denial"
            await expectStatus(detailPath, token("editor"), 403, "PATCH", false)
        }
        console.log("PASS: anonymous/forged/viewer/editor quest API checks (no successful catalog writes).")
    } catch (error) {
        failure = error instanceof SafeFailure ? error.message : `${stage} failed; raw error suppressed.`
    } finally {
        // Independent best-effort cleanup of ONLY IDs captured from this run's createUser responses.
        for (const entry of [...own].reverse()) {
            try {
                if (entry.token) {
                    const result = await admin.auth.admin.signOut(entry.token, "global")
                    if (result.error) cleanupFailures.push("session revocation")
                }
            } catch { cleanupFailures.push("session revocation") }
            try {
                if (entry.client) {
                    const result = await entry.client.auth.signOut({ scope: "global" })
                    if (result.error) cleanupFailures.push("client sign-out")
                }
            } catch { cleanupFailures.push("client sign-out") }
            try {
                const result = await admin.auth.admin.deleteUser(entry.id)
                if (result.error) cleanupFailures.push("own-user deletion")
                else entry.deleted = true
            } catch { cleanupFailures.push("own-user deletion") }
            try { await journal() } catch { cleanupFailures.push("recovery journal update") }
        }
        process.off("SIGINT", onSignal)
        process.off("SIGTERM", onSignal)
        if (journalPath && !cleanupFailures.length && own.every(entry => entry.deleted)) {
            try { await unlink(journalPath) } catch { cleanupFailures.push("recovery journal removal") }
        }
    }
    if (failure) console.error(`FAIL: ${failure}`)
    if (cleanupFailures.length) {
        console.error("FAIL: best-effort cleanup incomplete; owner review required (IDs are not printed).")
        if (journalPath) console.error(`Private own-user ID journal: ${journalPath}`)
    } else if (own.length) {
        console.log("CLEANUP: captured own auth users deleted; session revocation/sign-out attempted.")
    }
    if (failure || cleanupFailures.length) process.exitCode = 1
}

main().catch(error => {
    console.error(error instanceof SafeFailure ? error.message : "Integration setup failed; raw error suppressed.")
    process.exitCode = 1
})
