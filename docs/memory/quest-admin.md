# Quest Admin Dashboard

## Authentication mode (2026-10-14)

The Quest Admin pages and `/api/quests/**` routes no longer require a Supabase Auth or admin session. `requireQuestPermission` supplies a fixed internal UUID and `canWrite=true` so list, detail, lookup and save flows work directly from the dashboard. Non-read requests still require an exact same-origin `Origin` header; request validation, revision conflict checks, active-use guards and audit rows remain unchanged. This intentionally makes Quest Admin unauthenticated and should only be exposed through the trusted dashboard network boundary.

Offline editor/validation/permission tests and dashboard typecheck PASS. The opt-in deployed API script now checks anonymous read/write capability, malformed-write rejection and the same-origin guard without creating Auth users. Production dashboard build compiles and typechecks but standalone trace copying is blocked by Windows symlink `EPERM` in the current environment.
