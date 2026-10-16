# Quest Admin Dashboard

## Authentication mode (2026-10-14)

The Quest Admin pages and `/api/quests/**` routes no longer require a Supabase Auth or admin session. `requireQuestPermission` supplies a fixed internal UUID and `canWrite=true` so list, detail, lookup and save flows work directly from the dashboard. Non-read requests still require an exact same-origin `Origin` header. This intentionally makes Quest Admin unauthenticated and should only be exposed through the trusted dashboard network boundary.

## Active quest editing (2026-10-16)

Quest detail editing remains enabled when `activeUse.activeQuests > 0`, including objectives, NPCs, rewards, requirements and prerequisites. `public.quest_admin_save` no longer raises `ACTIVE_QUEST_IN_USE`, but still enforces request validation, optimistic revision, same-transaction before-image/audit, ownership checks and prerequisite-cycle validation. Saving never resets, removes or compensates persisted player progress; runtime may compare existing progress against the new catalog definition. Shared loop editing keeps its separate active-use guard.

Offline editor/validation/permission tests and dashboard typecheck PASS. The opt-in deployed API script now checks anonymous read/write capability, malformed-write rejection and the same-origin guard without creating Auth users. Production dashboard build compiles and typechecks but standalone trace copying is blocked by Windows symlink `EPERM` in the current environment. Migration `20261016073610_allow_edit_active_quests.sql` is staged but was not applied locally because `supabase-db` was stopped.
