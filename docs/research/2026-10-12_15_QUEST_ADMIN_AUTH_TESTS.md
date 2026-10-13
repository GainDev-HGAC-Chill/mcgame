# Quest admin authorization tests — owner runbook

## Status and scope

The opt-in script is `dashboard/tests/quest-auth.integration.mjs`. It has **not been run against live Auth or the quest API**. Quest route source was not yet present when this script was written; the owner/parent must review the final deployed API contract before running. No production-deploy or gameplay verification is claimed here.

The pure tests remain independent:

```powershell
node --test dashboard/tests/quest-permissions.test.mjs
```

The integration file deliberately does not match `*.test.mjs`; always invoke pure suites with explicit filenames or `*.test.mjs`, not every `.mjs` file.

## Mandatory safety review before running

1. Deploy the production dashboard with the quest routes. Use a loopback origin (`localhost` or `127.0.0.1`), with the proxy preserving that origin for write checks.
2. Through the sanctioned local compose database inspection path, inspect live triggers on `auth.users`, their called functions, Auth hooks, and any insert/delete cascades. Verify that creating/deleting these Auth-only users cannot create player characters, modify protected accounts/progress, grant unexpected roles, or populate gameplay data. **Do not acknowledge the trigger flag if this cannot be proven.** The script does not execute SQL or automatically inspect/disable triggers; review is the parent's responsibility. Stop rather than modifying hooks or gameplay tables to make the test run.
3. Review the final API: `/api/quests` GET returns 401 anonymously and 200 to viewers/editors; the access endpoint exposes a boolean `canWrite` at the root or under `data`; the reviewed lookup path returns JSON/200; `/api/quests/{id}` PATCH checks authorization and origin before parsing JSON. Supply an existing quest ID, not a player ID.
4. Explicitly authorize the three short-lived local Auth-only users and inspect this script before acknowledging its flags. No existing user's role changes are performed.

## Run after approval and deploy

From repository root, PowerShell (replace the example URL, lookup path/query and quest ID with reviewed values):

```powershell
$env:QUEST_ADMIN_INTEGRATION = "1"
$env:QUEST_ADMIN_AUTH_TRIGGERS_REVIEWED = "1"
$env:QUEST_ADMIN_API_CONTRACT_REVIEWED = "1"
$env:QUEST_ADMIN_DASHBOARD_URL = "http://127.0.0.1:3000"
$env:QUEST_ADMIN_ACCESS_PATH = "/api/quests/access"
$env:QUEST_ADMIN_LOOKUP_PATH = "/api/quests/lookups?kind=item&search="
$env:QUEST_ADMIN_QUEST_ID = "1"
$env:QUEST_ADMIN_TEST_ORIGIN = "1" # optional missing-Origin rejection probe
node dashboard/tests/quest-auth.integration.mjs
```

`QUEST_ADMIN_LOOKUP_PATH` and `QUEST_ADMIN_QUEST_ID` are required: their example values are **not a confirmed API contract**. The access path defaults to `/api/quests/access`. The dashboard URL must be an HTTP(S) loopback origin without credentials, query, fragment or path. Supabase always uses `http://127.0.0.1:<gateway port>`, with the port derived from `docker/.env` (`API_GW_HTTP_PORT`, then `KONG_HTTP_PORT`, then compose's 8000 default). Keys are read using Node `fs` from that same file, never printed. No arbitrary Supabase URL override is accepted. Requests have timeouts and do not follow redirects with credentials.

Clear the acknowledgement flags after the run so they cannot authorize a later unattended run:

```powershell
Remove-Item Env:QUEST_ADMIN_INTEGRATION, Env:QUEST_ADMIN_AUTH_TRIGGERS_REVIEWED, Env:QUEST_ADMIN_API_CONTRACT_REVIEWED -ErrorAction SilentlyContinue
```

## Checks and cleanup

- Before user creation, anonymous list/access requests must return 401. Missing/unavailable routes stop the run without creating users.
- Exactly three random-email/random-password users are created via the local Supabase Admin SDK: no trusted role plus forged `user_metadata`, trusted `quest_viewer`, trusted `quest_editor`. Email confirmation is administrative; no email delivery or game login/character creation is requested.
- Each signs in with `signInWithPassword`; `getUser(access_token)` verifies the server-side user ID and expected trusted role before use.
- Forged user metadata must receive 403. Viewer GET is allowed, `canWrite=false`, PATCH denied403. Editor GET is allowed, `canWrite=true`, catalog lookup JSON allowed. Optional editor PATCH without Origin must return403.
- **Every PATCH body is deliberately malformed JSON (`{`). No valid save payload is sent.** Parent added an editor same-origin malformed-JSON probe expecting422 and canonical detail/revision comparison before/after; the deployed contract must be reviewed before execution. These checks do not prove successful transaction delivery, database rollback or SQL RLS.
- `finally` attempts global session revocation and client sign-out, then deletes only captured IDs returned by this run's `createUser` calls. There is no user listing, prefix deletion or broad SQL cleanup. Ordinary SIGINT/SIGTERM requests deferred cleanup; forced process termination cannot guarantee cleanup.
- A per-run recovery journal stores only captured IDs/deletion markers under a unique private temp directory (Windows: `%LOCALAPPDATA%\Temp\opencode\quest-auth-*`). No email/password/token/key is persisted or logged. Node file modes restrict access on POSIX; on Windows access inherits the owner's temp-directory ACL. Successful cleanup removes the journal file. Cleanup failures retain the journal and report its path, not its IDs; review it privately and revoke/delete **only** those captured users through the local Admin API. Do not paste credentials or journals into public reports.
- If Auth commits a create but the network response is lost before an ID is received, the script cannot safely identify/delete that user. Inspect local Auth logs privately; never compensate with a broad delete. A cleanup failure or test failure exits nonzero. Revocation does not instantly invalidate already issued JWTs; these tokens are never persisted, and the API authenticates with `getUser`.

## README: how the owner obtains quest access

Existing dashboard open-session/admin-secret access does not grant quest access. An explicitly selected owner's **Supabase Auth** identity needs trusted `app_metadata.quest_role` (`quest_viewer` for read-only, `quest_editor` for editing; `admin` is also recognized but is broader and unnecessary here). Trusted `app_metadata.admin_role` or exact `permissions` arrays (`quests:read`, `quests:write`) are alternate gates. Never put authorization in user-editable `user_metadata`.

After separate owner authorization, use the local Supabase Admin Auth API from a private server-side console, with the service-role key from compose. Fetch the exact selected user ID first, preserve existing application metadata, and update **only that identity**:

```js
// Illustrative owner-admin procedure, NOT executed by the integration script.
const { data, error } = await admin.auth.admin.getUserById(explicitlyApprovedOwnerId)
if (error || !data.user) throw new Error("Owner lookup failed")
const result = await admin.auth.admin.updateUserById(data.user.id, {
    app_metadata: { ...data.user.app_metadata, quest_role: "quest_editor" },
})
if (result.error) throw new Error("Owner role update failed")
```

Never expose the service-role key in browser code, grant all existing users roles, change player/game accounts, or automate a universal admin grant. Re-login/refresh the selected owner's session after role provisioning. Quest API authorization uses fresh server-verified `getUser()` records, not `getSession()` or client-decoded claims.

## Documentation checked

- [Supabase `getUser`](https://supabase.com/docs/reference/javascript/auth-getuser): performs an Auth-server request; returned user is authentic for authorization.
- [SSR clients](https://supabase.com/docs/guides/auth/server-side/creating-a-client): do not trust cookie-loaded `getSession()` user objects.
- [Admin `createUser`](https://supabase.com/docs/reference/javascript/auth-admin-createuser): server-only administrative creation and email confirmation.
- [Admin `signOut`](https://supabase.com/docs/reference/javascript/auth-admin-signout): revoke a logged-in JWT's session using the explicit global scope.

Local verification performed: `node --check dashboard/tests/quest-auth.integration.mjs` passed; `node --test dashboard/tests/quest-permissions.test.mjs` passed all eight tests. The integration script itself was not executed and no Auth users were created. Keep verification limited to these offline checks until the owner approves the live integration run.
