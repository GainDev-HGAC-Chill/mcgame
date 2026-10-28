# Guild System

## 2026-10-27 full audit invariants

- Full evidence and gap matrix: `docs/research/2026-10-27_02_GUILD_SYSTEM_AUDIT_RESEARCH.md`; draft repair plan: `docs/plans/2026-10-27_01_GUILD_SYSTEM_REPAIR.md`.
- Flash wire authority exposes 32 guild RPCs. The initial audit used a truncated `CallBack.as`; complete shipped traits recovered on 2026-10-28 supersede its callback conclusions.
- Create contract in this client version is level 50 plus 1,000,000 silver. Historical website values are not wire/economy authority.
- Contribution, warehouse merge, guild creation/lifecycle restore, and other multi-step mutations are not end-to-end atomic. Character currencies 20/21 and `guild_members` contribution fields currently duplicate personal contribution authority.
- Rank settings and bank pages are JSON snapshots in `guild_logs` actions 9001/9002. Runtime member permission columns are not synchronized with the six client settings (`canAdd/canQuest/canSlot/canInfo/canDel/canDuty`).
- Construction authority is currently `Handler.buildStates`, process-local and keyed only by build ID. Persist one row per guild/extension; never infer buildings from guild level. `TBL_BUILDING` has 18 rows and map 49 has 11 extension positions.
- Guild skills are 130 `useEnv=4` rows: 13 families × 10 levels. Client exact-next lookup uses same `codeName`; every family changes to `*NO` at level 7, so level 7–10 must remain blocked without further authority.
- Guild quest loop `lid=2` / NPC 1069 already has membership gating and leave cleanup; preserve and regression-test it rather than rebuilding it.
- No guild code/schema/runtime change was made during this audit. Implementation, migration apply, restart/deploy, and player-data mutation remain approval-gated.

## 2026-10-27 Wave 0 wire characterization

- `internal/presentation/rtmp/handlers/guild/contract_test.go` locks all 32 registrations and the client-visible callback names/arity.
- Complete shipped `CallBack` requires `onUpdateGuildRank({gid,rankData})` as one wrapper object. Global wrappers are `onGuildPropertyRefresh(Object)`, `onUpdateGuildNumProp(Object)`, `onUpdatePerNumProp(Object)`, and `updateGuildInfo(Object)`; `onUpdateGuildProperty`, `onUpdateNumProp`, and `onUpdateGuildInfo` are panel methods and must never be sent as NetConnection callbacks.
- `catalog_contract_test.go` locks 18 buildings, 11 map-49 extension points, 130 `useEnv=4` skills, 13 ten-level families, and the `codeName` → `codeNameNO` level-6/7 boundary.
- Focused guild handler tests pass. Schema, runtime data, deployment, and unresolved warehouse/building callback ordering remain untouched.

## 2026-10-27 Wave 1 typed permission code

- `domain/guild.RankSetting` is now the typed authority for ranks 1–6 and all six flags. The Flash DTO remains integer `0/1`; malformed, duplicate, missing-rank, and mutable leader settings fail closed.
- Application approval/refusal, invitation, kick, notice, duty changes, and warehouse mutations authorize against the same rank settings shown in the panel. Legacy per-member permission booleans are no longer consulted for these actions.
- Skill development and construction remain leader-only until a narrower permission mapping is evidenced. Guild-level, bank-page, and member-capacity upgrades are disabled with `ErrUnsupportedAction` because no authoritative cost exists.
- Repository code now targets typed `list_guild_rank_settings` / `replace_guild_rank_settings`; the schema/function draft is in `supabase/data/guild_system.sql`. It remains DB-pending and must not be deployed before approved apply/capture.

## 2026-10-27 Wave 2 atomic create and money contribution

- `application/guild.CommandRepository` owns atomic creation and monetary contribution commands. RTMP handlers only emit callbacks after a committed result and no longer perform compensation-based multi-service writes.
- `cache.CachedGuildRepository` flushes write-behind player state before a PostgreSQL command and reconciles returned guild ID, wallets, contribution currencies, and restore balances without marking them dirty afterward. Existing concurrent dirty character state is preserved.
- The SQL draft intends to lock character/wallet/guild/member rows, enforce level 50 and 1,000,000 silver creation cost, seed typed ranks, update currencies 20/21 and member ledgers in one transaction, and compute donation conversion inside PostgreSQL. It is unverified and must not be applied without separate approval and review.
- Focused application guild, cache, PostgreSQL, guild handler, and gameserver tests pass. Compose DB apply, migration capture, rollback smoke, runtime restart, and player mutation have not occurred.
- Material contribution now uses the same command boundary: PostgreSQL locks the owned bag item, member, and guild; consumes the stack, upserts the warehouse material, and credits currency 21 plus member/guild ledgers atomically. The cache adapter reconciles both item count/deletion and character contribution state after commit.

### Safety correction

- The prior SQL draft incorrectly referenced wallet and level on `player.characters`; the current draft targets split `player.character_wallet` and `player.character_progression`, but has not been DB-verified.
- A shared per-player mutation barrier now serializes write-behind persistence with create/contribution commands. Character and item snapshots carry cache revisions, so a clone read before a guild command cannot overwrite the committed wallet/contribution/item state afterward; it is rejected as stale. Commands flush, commit, reconcile without dirtying state, then refresh Redis under the same barrier.
- The same barrier now covers character rename and DB-owned Pet fusion/illusion/skill-book commands; Pet snapshots also carry revisions. This closes cross-system stale writes that could otherwise race a guild economy command even when the guild repository itself was serialized.
- Focused concurrency regression passes, but Windows race instrumentation is unavailable without CGO. Direct cache mutation call sites, lifecycle integration, database failure/concurrency tests, and runtime behavior still require verification before deployment.

## 2026-10-27 skill catalog guard

- `ResolveGuildSkillDevelopment` checks exact-next same-code-name rows, `useEnv=4`, integer level 1–6 and nonnegative integral guild money/exp costs; nonzero unresolved `costGuildContrib`, malformed persisted rows, duplicate family state and skips fail closed.
- `DevelopSkill` uses the resolver and `develop_guild_skill_atomic` draft: leader and expected family state are rechecked under a guild lock; resource debit and replacement of the previous skill row share one transaction. Guild rename/notice now use targeted metadata commands instead of full-row writes and the old multi-commit `AddContribution` path is disabled. The SQL remains unverified and notice authorization still has a transaction race; do not deploy until DB concurrency and authorization are verified.
- Typed rank reads append and validate all scanned rows; if a guild has no typed rows, its latest legacy action-9001 snapshot is read and validated before considering defaults. This prevents silently broadening permissions when an existing guild has customized ranks.

## 2026-10-27 warehouse command draft

- Move, merge and drop now go through atomic repository commands rather than separate slot updates. Draft SQL locks the guild and affected slots, validates the persisted actor/rank, current page count, source identity and stack bounds, and returns committed counts for the existing callbacks.
- Typed rank updates acquire the guild lock. For legacy guilds without typed rank rows, warehouse SQL denies non-leaders rather than inferring permissions; a reviewed migration path is still needed.
- Focused unit tests pass. Draft authorization locks the guild, actor membership and typed rank row; legacy membership/rank paths and lock ordering still require PostgreSQL concurrency tests. The functions remain unverified against PostgreSQL; migration and runtime trace remain pending, so do not deploy.
- Material contribution now carries the catalog max stack and persisted bank page count into its transaction. It locks the guild before member/warehouse state, merges only into a stack that can hold the full committed amount, otherwise allocates the first empty slot below `pageCount*30`; overflow or a full warehouse rolls back the item consumption and contribution.

## 2026-10-27 persisted building draft

- Process-local `Handler.buildStates` and guild-level-derived fake layouts are removed. Building list/start/finish/reset now use a durable command boundary keyed by guild and extension; the draft table stores completed and pending template IDs plus mode/revision.
- Catalog validation enforces map 49 extension identity, base-create or exact-next same-family upgrade, prerequisites, family count limit, and integral nonnegative guild money/experience/general/rare/special material costs. Nonzero building gold cost is fail-closed because payer semantics are unresolved.
- Draft finish SQL serializes on guild/building/warehouse rows, debits costs only at finish, and reset only clears pending state without cost/refund. Focused tests pass, but SQL has not been applied or syntax/concurrency tested and callback ordering remains unverified; do not deploy.
- Static review fixed the PostgreSQL adapter's finish call from 11 placeholders to the function's 10 parameters. Start now rejects null or duplicate family-ID arrays so SQL null semantics cannot bypass the catalog-family guard.

## 2026-10-27 membership lifecycle draft

- Apply, approve, refuse, cancel, leave, kick and leadership transfer now use lifecycle command boundaries. Draft SQL locks durable authority, updates canonical contribution currencies 20/21 and member ledgers/restores atomically, and returns committed character/member/application snapshots for cache reconciliation.
- `demiseTo` is decoded as the client-sent membership row ID before resolving its character ID. Application cancellation is resolved before membership IDs to avoid sequence collisions. Duplicate approval delivery and the false contribution-reset message on refusal are removed.
- Departure cleanup deactivates guild loop 2 and then deletes active guild child quests post-commit. Invite and disband remain fail-closed: no client-proven invite acceptance exists, and disband still needs a safe multi-player cache barrier.
- Notice and rank-setting writes now recheck persisted permission/leadership under the guild lock. Focused tests pass, but lifecycle SQL remains unapplied and unverified on PostgreSQL; approval row-identity refresh and callback ordering still require runtime evidence.
- Refusal now takes the same per-character advisory lock as apply/approve/cancel, and approve/exit/transfer assert their expected affected-row counts. Economy commands lock guild before member rows to avoid the inverse member→guild order against leadership transfer.

## 2026-10-27 database verification

- The rank/economy/warehouse/building/lifecycle SQL is applied on the compose database and captured in `20261027104736_guild_system_atomic.sql`. Transaction rollback smoke covers create, both contribution classes, move/drop, building finish/reset, skill development, application approval, leadership transfer, and leave; the post-smoke census matches the original guild/member/item/character state.
- Two-session concurrency smoke confirms serialization for money contribution, approve versus refuse, warehouse move versus drop, and building finish versus reset. The blocked second session resumed after roughly 2.2 seconds without a deadlock or lock timeout.
- RLS, function execution grants, empty `search_path`, service-role access, migration shadow replay, and post-capture schema diff pass for guild objects. The remaining Player-schema lint errors are legacy character/farm functions outside guild scope.
- Legacy `upsert_guild_warehouse_material` had no caller after the atomic material command landed and lacked actor, page, max-stack, and item-debit authority. The dead Go interface/adapter was removed and migration `20261027105140_guild_legacy_warehouse_cleanup.sql` drops the unsafe function; subsequent guild lint and schema diff are clean.
- Approval callback identity is intentionally split: top-level `tableId` remains the deleted application row ID because Flash uses it to transition/remove the pending placeholder, while nested `gData.tableId` is the new durable membership row ID. The online applicant receives `onConfirmGuildApply` before the committed contribution `onUPP` and informational message; callback-spy regression locks this order.
- Flash source plus callback-spy verification establishes approval callback ordering. Invite and disband remain unsupported, guild-level/page/member-capacity upgrades remain blocked without cost authority, and distributed multi-server cache behavior remains unobserved.
- Live compose login for character 3 on map 49 reached `initViewGuildP`; the operator confirmed the `Gain` panel plus member/application, skill, and warehouse read views render normally. No guild mutation RPC was invoked and the post-session guild/application/warehouse/rank/building/skill census remained unchanged. Multi-client approval and distributed multi-server cache behavior remain unobserved.

## 2026-10-28 construction entry contract

- Complete traits recovered from the shipped main SWF prove that `onCreateGuildBuildings(Object)` hydrates map buildings through `sceneCreateGuildBuildings`, while `onShowNewBuildManager(int extendId)` resolves the empty extension through `view.getE(extendId)` and opens its construction manager.
- `NpcScriptPanel` invokes `npcScript("startConstruction")` without a responder. Returning extension ID 10 as the RPC result cannot open the panel; NPC 1098 must push `onShowNewBuildManager(10)`. A callback named `startConstruction` does not exist and was rejected by the live client.
- Extension 10 is the catalog empty point at `(245,407)`. The server emits the proven callback and has a regression for exact name, one-argument arity, value 10, and nil fire-and-forget result.
- Live start persisted pending state, survived gameserver restarts, reopened progress from the shipped pending path, and reset through exact `onCancelCreate({bid,ext})` back to empty extension. Live finish Đại sảnh cấp 1 committed template 2/revision 2 at cost 0; the test row was transactionally removed afterward and all guild/player resources returned to baseline.
- Complete shipped trait correction: `onUpdateGuildRank` receives one `{gid,rankData}` wrapper; global property/numeric/info wrappers are `onGuildPropertyRefresh`, `onUpdateGuildNumProp` / `onUpdatePerNumProp`, and `updateGuildInfo`. Their similarly named GuildPanel methods are not NetConnection callback names.

## 2026-10-28 remaining-building safety boundary

- The catalog contract now locks all 16 playable templates, including exact family/level topology, prerequisites, family limits, money/EXP/three-material costs, zero Gold cost, and property pairs. Hall 1 and Warehouse 1 are the only zero-resource playable finishes.
- Warehouse, donation, and Academy functions require a completed persisted template whose `funcScript` exposes the requested function. Pending rows do not qualify. This gate covers the fixed warehouse NPC, `execBuildFunc`, direct warehouse/skill reads, contribution, warehouse move/drop, and skill development RPC paths.
- Building property application remains deliberately deferred. Property labels identify member limit, building level, and guild warehouse, but current evidence does not establish House aggregation or whether Hall property 4 mutates `guilds.level`; do not derive those values by guess.
- Guild loop ID 2 is fail-closed until dynamic reward authority exists. New takes advertise unavailable, and retained active rounds are rejected before objective consumption, progress deletion, or loop advancement.

## Domain layout

- `internal/domain/guild/guild.go` — `Guild`, `GuildMember`, constants; `ToDTO()` currently emits 22 keys (the old 18-key note/comment is stale).
- `internal/domain/guild/extras.go` — `GuildApplication`, `GuildWarehouseSlot`.
- `internal/application/guild/service.go` — `Repository` interface + `Service` struct.
- `internal/application/guild/service_extra.go` — enrichment helpers, `GetGuildByMemberForClient`, `GetGuildSkillData`, `enrichGuild`, etc.
- `internal/infrastructure/persistence/postgres/guild_repository_extra.go` — repo impl; `GetGuildSkillLevels` does a direct query (no PG function needed — simple two-column SELECT).

## DB schema

- `player.guilds` — columns: `id, name, leader_id, level, experience, population, max_population, icon, funds, contribution_total, announcement, description, join_level_req, join_approval_required, activity_points, last_activity_reset, created_at, updated_at`. Note: no `icon_code`, `daily_produce`, `money_limit`, `total_exp` columns — these are stubs emitted as defaults in `ToDTO()`.
- `player.guild_skills` — `(id, guild_id, skill_id, level)`, UNIQUE on `(guild_id, skill_id)`. `level` is populated by `player.upsert_guild_skill` (inserts with level 1 on conflict does nothing — level upgrades not yet implemented).

## Wire shape (18-key guild object, M5 parity)

`ToDTO()` emits:
- `id`, `name`, `cid`, `ln` (leader id), `level`, `exp`, `money`, `guildInfo`, `bagSlotNum`, `memLimit`, `memberNumber`, `announcement`, `joinLevelReq`, `members` — from real DB data.
- `iconCode: null` — stub; DB has only `icon` column; live capture shows `null`.
- `dailyProduce: "0"`, `daliyCost: "0"` (sic — live protocol misspelling), `moneyLimit: "0"`, `timeStamp: null`, `totalExp: g.Experience` — stubs/defaults (no backing columns).
- `time` — formatted from `g.CreatedAt` as `"2006-01-02 15:04:05"` or `nil` if zero.
- `skillData` — JSON string `{skillId: level}` from `SkillLevels map[int]int`; `"{}"` when no skills.

## skillData flow

`enrichGuild` calls `GetGuildSkillLevels` (returns `map[int]int` of skill_id→level from `guild_skills`), stores in `Guild.SkillLevels`. `buildSkillDataJSON()` serializes to JSON string. The existing `GetGuildSkillData` and `SkillDevData` are retained for backward compat with non-login guild handlers.

## Open questions (M5)

- `iconCode` has no backing DB column. Live shows `null`. When guild emblem feature lands, add `icon_code` column via Supabase CLI + `db diff` and wire it.
- `daliyCost` / `dailyProduce` / `moneyLimit` / `timeStamp` / `totalExp` as separate columns — not yet in schema. Emit defaults for now.
- `upsert_guild_skill` always inserts level=1 on new; no level-upgrade path exists yet. When guild-skill leveling is implemented, add a `level_up_guild_skill` PG function.
### Disband and Hall level hardening (2026-10-29)

Guild disband is now one database-owned command. The service snapshots member IDs, the cache layer takes every player mutation barrier in stable character-ID order, and PostgreSQL rejects membership drift under the guild lock. A successful command applies the same floor-90% currency20/21 restore semantics as leave/kick to every member, clears character guild membership, deletes the guild and returns deterministic committed character states. Cache reconciliation and idempotent guild-quest cleanup occur before the global `onDelGuild(gid)` broadcast. Concurrent duplicate disband produced exactly one winner in compose verification.

The completed Hall family is exact catalog `BUILD100001`; its level rows independently encode prop1 type4/value equal to Hall level. Building finish now passes that derived level into PostgreSQL, which revalidates the catalog, prevents lower/skip transitions and updates `guilds.level` in the same transaction as resource consumption and building completion. Concurrent finish/reset serialized safely. Migration: `20261028175744_guild_disband_and_hall_level.sql`.
