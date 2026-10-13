# Admin Quest Manager

**Status:** Authorized by the user's explicit end-to-end implementation request; implementation follows confirmed contracts below. No additional gameplay semantics or destructive operations are authorized.

## Confirmed mechanics

- Quest catalogs reside in `data.data_tbl_quest`, `_quest_require`, `_quest_award`, `_quest_pre`, `_quest_loop`; player state/history are separate and must not be rewritten.
- Flash requirement kinds differ from domain objective IDs: item=1/table19 or29, kill=2/table12, submit pet=3/table12. Collect quality is exact catalog quality mapping; pet requirement quality is growth threshold scaled by10.
- Prerequisite kind1 is carried item; kind2 is completed quest. Quest `pre_quest_type`1=AND and2=OR; other values ignore quest prereqs in the Flash contract.
- Existing Box Item Award provides searchable catalog controls and Vietnamese multi-row editor patterns. Do not equate its reward enum/payload with quest awards without verifying mappings.
- Existing dashboard session helper is open, not authentication. Quest APIs must use verified Supabase users and trusted `app_metadata` permissions, never user-editable metadata or an open-session fallback.

## Design

1. `/dashboard/quests`: paginated searchable/filterable list and detail editor; Vietnamese text. Pick NPC/creature/item/pet/previous quest/loop using catalog search, not mandatory handwritten IDs.
2. New `/api/quests` routes protected by verified user permissions: admin or quest_editor can write; quest_viewer can read. Server-only service-role queries. Same-origin and strict JSON/body limits for writes. No anonymous CRUD.
3. Typed aggregate for one quest plus requirement/award/prerequisite rows and explicit loop edits. Unedited legacy/custom fields remain intact. Unsupported enum combinations are displayed as legacy and cannot silently be recast into valid-looking rows. SQL verifies catalog references/enum pairs and row ownership as well as API validation.
4. One SQL function atomically snapshots the before-image, checks optimistic revision/hash, validates references and writes only the selected aggregate. Existing child IDs preserved where possible; removals explicitly scoped to that quest. Shared loop changes require explicit selection and warning; no silent global loop mutation.
5. Initial local physical backup before schema/data writes. Per-save before-image/audit is a transaction prerequisite. Revoke direct quest-table writes by unprivileged roles to prevent bypassing new authenticated APIs. All schema changes captured with compose-targeted Supabase CLI diff, not handwritten migrations.
6. Verify backend manager and Flash data hydration. Backend restart alone activates only the backend cache. Research found no verified generic Flash catalog/index refresh callback: existing quest IDs bypass `gdc`, and replacing primary arrays does not rebuild qid/NPC/type indexes. Editor must report database-only save and blocked consistent activation, not promise restart/relog suffices. Trace any external loader data publication before declaring this impossible; don't invent callbacks or patch decompiled AS.

## Verified runtime limitations

- Newly introduced/changed reward rows must match implemented semantics: bound rewards only (`b=1`), one pet per pet award row (`num=1`), and automatic award mode (`at=1`). Existing legacy values are preserved unchanged and visibly warned about. No fabricated award selection or silently ignored reward quantities.
- Service lifecycle verification uses actual application services with isolated in-memory repositories; it is not a live Flash gameplay check. Flash end-to-end remains blocked until consistent client catalog activation is verified.
- `reconcileCatalogObjectives` matches objective quantity as well as type/target/quality. Changing those definitions can implicitly lose existing event progress after activation, even without a SQL progress update. Structural changes must reject persisted active-use conflicts; shared-loop changes must check active loop usage. Database checks do not by themselves certify unsaved live cache state. No automatic catalog activation/restart is permitted, and a future consistent activation procedure must drain/persist sessions and recheck active-use conflicts first.

## Touch points

| Area | Change |
|---|---|
| `dashboard/src/app/dashboard/quests/` | List, detail, guided objective/reward/gate/loop forms |
| `dashboard/src/app/api/quests/` | Authorized list/detail/lookups/save/access endpoints |
| `dashboard/src/lib/server/quest-admin-auth.ts` | Verified Supabase authentication/trusted role gate |
| `dashboard/src/lib/quests/`, `dashboard/src/types/quests.ts` | Exact catalogs, enums and validators |
| `supabase/data/` | Reviewed DB function drafts applied to local compose only |
| `supabase/migrations/` | CLI-generated schema diff |
| `internal/application/quest/` tests | Isolated real service receive/progress/complete/reward regression |
| `docs/research/`, `docs/memory/` | Schema/client/cache findings and validation evidence |

## Non-goals

- No player progress reset, history edits, compensation or replay of existing rewards.
- No ActionScript changes or invented evaluation for unsupported custom scripts/currency conditions.
- No global dashboard-auth replacement or public exposure disguised as this feature.
- No automatic restart/disconnect during each admin save. Restart requirements are explicit, owner-controlled.

## Verification

- Strict valid/invalid enum pairs, numeric bounds, referenced catalog existence, duplicate rows, row ownership, previous-quest cycles and optimistic conflicts; invalid save leaves all rows unchanged.
- Unauthorized401, authenticated wrong-role403, read-only role write403, editor write allowed; `user_metadata` escalation ignored; direct unprivileged SQL/API writes blocked.
- Local before-image backup present before writes; aggregate updates rollback on any failure; other quest catalogs and player-progress fingerprints unchanged.
- Frontend typecheck/production build and browser search/form validation. Focused backend Go tests/build/vet; report seven known baseline failures separately.
- Isolated sample quest with actual application service acceptance, monster/item progress, NPC proximity gate, completion and reward once-only checks. Local database execution is tested without touching protected characters; no claim of real Flash visual gameplay unless actually exercised.
