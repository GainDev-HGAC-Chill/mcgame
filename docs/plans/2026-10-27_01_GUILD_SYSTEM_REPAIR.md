# Guild System Repair — Implementation Plan

**Date**: 2026-10-27 · **Status**: Approved; callback correction incorporated 2026-10-28  
**Goal**: Make the existing 32-RPC guild surface conform to the inspected Flash client while making membership, contribution, warehouse, skill, and building mutations durable, authorized, and atomic. Unknown original-server mechanics remain fail-closed rather than being reconstructed from guesses.

Research base (verified 2026-10-27):
- `docs/research/2026-10-27_02_GUILD_SYSTEM_AUDIT_RESEARCH.md` — full wire/catalog/schema/gap audit and source URLs.
- `docs/memory/guild-system.md` — repository map and consolidated guild invariants.
- `docs/client/view/view/compDragable/GuildPanel.as` — primary RPC and visible callback contract.
- `docs/database/game_data/TBL_BUILDING.json` and `TBL_SKILL.json` — authoritative costs, prerequisites, and chains.

## 1. Confirmed mechanics

- Preserve all 32 registered RPC names and existing client argument shapes.
- Guild creation requires level 50 and 1,000,000 silver in this client version.
- Complete shipped traits supersede the truncated-source assumption: global `onUpdateGuildRank` takes one `{gid,rankData}` wrapper.
- Global wrappers are `onGuildPropertyRefresh`, `onUpdateGuildNumProp` / `onUpdatePerNumProp`, and `updateGuildInfo`; similarly named GuildPanel methods are not NetConnection callbacks.
- `onUpdateNumProp.type` is 1 donation contribution, 2 normal contribution, 3 guild money, 4 guild experience.
- Rank settings contain display name plus `canAdd`, `canQuest`, `canSlot`, `canInfo`, `canDel`, and `canDuty` for ranks 1–6.
- Building placement is map 49 plus 11 catalog extension points; building templates/costs/prerequisites come only from `TBL_BUILDING`.
- Guild skills are `useEnv=4`; the client requests an exact skill row. Same-`codeName` progression is proven only through level 6.
- Guild quest loop `lid=2` already has a membership gate and departure cleanup.

## 2. Non-goals and fail-closed rules

- Do not edit ActionScript or game-data JSON.
- Do not implement guild war, daily maintenance/production, invented construction timers, or benefits not represented by current data.
- Do not guess skill transitions from level 6 to `*NO` rows 7–10.
- Do not invent costs for guild-level, bank-page, or member-capacity upgrades. Until an authoritative cost is found, keep these operations disabled with a truthful error rather than free progression.
- Do not change unresolved warehouse/building callback names or ordering without a runtime trace.
- Do not apply migrations, restart/deploy, backfill contribution, or mutate player/guild data without a separate checkpoint and approval.

## 3. Target transaction boundaries

Repository commands must own locks and all durable writes. Application services supply validated catalog facts but must not perform a durable mutation and then call another service hoping compensation succeeds.

| Command | Atomic writes and locks |
|---|---|
| Create guild | Lock character/wallet/membership; validate unique name, level and no membership; debit bound-or-unbound silver according to the current rule; insert guild and leader member; set character guild ID; initialize rank rows; commit once |
| Approve/leave/kick/disband | Lock guild, actor/target membership, application and character; validate permission/capacity; change member/application/population/character guild ID; transfer contribution to restore fields or restore canonical currencies as applicable; commit once |
| Contribute money | Lock character wallet/currency rows, guild and member; validate positive bounded amounts; debit money/gold; credit guild money and personal contribution ledgers/currencies; return committed totals |
| Contribute material | Lock owned item, guild/member and destination warehouse material slot; validate template/count; decrement/delete item; upsert warehouse resource; credit contribution; return committed snapshots |
| Move/drop warehouse | Lock guild/member and involved slots in stable ID order; authorize `canSlot`/leader rule; validate page/SID/stack; update/delete both slots in one transaction |
| Develop skill | Lock guild and family row; authorize; validate exact-next `useEnv=4` row and level <=6; debit catalog guild exp/money; upsert canonical family level; return guild/skill totals |
| Start/finish/reset building | Lock guild/building/warehouse resource rows; validate permission, extension, mode, predecessor/prerequisites/count limit; persist pending/final state and debit exact catalog costs in the selected lifecycle boundary; return committed building/resource snapshots |

## 4. Wave 0 — Characterization and contract fixtures

**Files**:
- Add focused tests under `internal/presentation/rtmp/handlers/guild/`.
- Add service/repository fixtures under `internal/application/guild/` and `internal/infrastructure/persistence/postgres/`.

**Work**:
1. Capture all 32 registrations and representative argument decoding.
2. Add AMF callback-spy tests for rank arity, property/numeric callback names, responder paths, and error paths.
3. Add catalog integrity tests: 18 building rows, 11 map-49 positions, 130 `useEnv=4` skills, 13 families × 10 levels, and the level-6/7 code-name boundary.
4. Add regression tests for current guild-loop membership gate and leave cancellation.

**Exit**: Tests reproduce every confirmed defect before production behavior changes. Unresolved callbacks are documented and kept logged.

## 5. Wave 1 — Wire contract and typed permissions

**Files**:
- `internal/presentation/rtmp/handlers/guild/*.go`
- `internal/application/guild/service_guild_management.go`
- `internal/domain/guild/*.go`
- `internal/infrastructure/persistence/postgres/guild_repository*.go`
- New migration in `supabase/migrations/` only after DB approval

**Work**:
1. Emit global `onUpdateGuildRank({gid,rankData})` as one wrapper object.
2. Use proven global wrappers `onGuildPropertyRefresh`, `onUpdateGuildNumProp` / `onUpdatePerNumProp`, and preserve exact payload keys.
3. Use proven global `updateGuildInfo`; never send panel-only `onUpdateGuildInfo`.
4. Add typed `player.guild_rank_settings` keyed by guild/rank. Validate rank range and six boolean flags; seed defaults on guild creation.
5. Centralize permission lookup and use it for application approval/refusal, invitation/member management, notice, rank/duty, warehouse, skill, and construction actions. Leader-only actions remain leader-only.
6. Remove runtime dependence on per-member permission booleans as an independent authority; retain compatibility reads only if needed during migration.

**Exit**: Callback tests pass; UI and server read one permission model; unauthorized direct RPC calls fail before mutation.

## 5.1 Follow-up addendum — remaining building and guild-quest scope

**Status**: Approved by the operator on 2026-10-28; implementation remains bounded by the missing-authority fail-closed rules below.

### Confirmed mechanics

1. The catalog has 16 playable templates across Hall 1–6, House 1–6, Warehouse 1–2, and Academy 1–2, plus empty/pending infrastructure rows.
2. Explicit finish is resource-gated, not time-gated: the shipped client checks guild money, guild EXP, and three material types, then calls `finishCreate` / `finishUpgrade`. No timer or construction percentage contract exists.
3. Hall 1 and Warehouse 1 cost zero; immediate finish for them is intentional. Every other playable row has nonzero costs.
4. Building property effects are currently missing server-side. Property labels establish member limit, building level, and guild warehouse, but House aggregation is not yet authoritative.
5. Guild loop ID 2 has 275 item/Pet child quests and no static award rows. Current completion consumes objectives and advances rounds but grants none of the documented dynamic personal/guild rewards.
6. Per-building construction tasks described by historical websites do not have a recoverable wire/data mapping in the current evidence set.

### Proposed waves

1. **Safety characterization**: add catalog topology/cost/property tests for every playable template, SQL insufficient-resource/rollback tests for every cost dimension, and a guild-loop regression proving the current zero-reward defect without mutating live data.
2. **Authority capture**: obtain original-server before/after traces for guild-loop rewards, milestone rounds, construction-task activation/completion, and House capacity aggregation. Record exact callbacks and amounts.
3. **Atomic building effects**: only after authority is complete, update building state and derived guild level/member cap/bank pages in the same finish transaction; gate Academy/Warehouse functions by completed persisted buildings.
4. **Atomic guild-loop rewards**: introduce one command that consumes the submitted item/Pet, credits personal contribution/currencies, guild funds/EXP/materials, grants character/milestone rewards, advances the loop, and returns one committed callback payload. No partial `applyRewards` chain.
5. **End-to-end matrix**: live-test each family and every level transition with exact resource checkpoints and rollback; separately test loop rounds 1, 5, 10, and 15. Restore baseline after each group.

### 2026-10-28 implementation checkpoint

- Wave 1 safety characterization is complete: all 16 playable building contracts are locked, completed-building function gates cover Warehouse/donation/Academy UI and direct RPC paths, and guild loop ID 2 fails before consumption or advancement while reward authority is absent.
- Wave 2 authority capture remains blocked on original-server evidence. Consequently, waves 3–5 cannot truthfully apply building properties, construction-task progress, or guild-loop rewards yet.
- Focused guild/quest/RTMP/cache/PostgreSQL/gameserver tests, `go vet ./...`, and the Windows gameserver build pass. Full tests retain only the four recorded out-of-scope baseline failures in questaudit, activity, and chat.

### 2026-10-28 NEO reward-direction addendum

The operator selected Vua Pháp Thuật NEO as the gameplay direction. This supersedes the earlier EXP-only interpretation but does not authorize invented formulas.

Confirmed from `https://www.youtube.com/watch?v=MT5gH-7QTkI` at character level 90:

- Two consecutive guild-quest completions show personal silver/bound-silver pairs `233,280` then `272,160`.
- The same completions show guild EXP `2,608` then `3,042`, personal contribution `1,304` then `1,521`, and `4,000` of a visible `Kết Tinh...` reward.
- Guild EXP is exactly twice personal contribution in both samples.
- The changing rewards fit the shipped/client `CLASS_QUEST_MONEY_EXP_NUM` first factors exactly: `233,280 / 0.6 = 272,160 / 0.7 = 388,800`; rounding `2,173 * 0.6/0.7` gives `1,304/1,521`, and rounding `4,346 * 0.6/0.7` gives `2,608/3,042`.

Still unconfirmed:

- How the level-dependent bases `388,800`, `2,173`, and `4,346` are derived outside level 90.
- Whether `Kết Tinh...` is an inventory item, a custom NEO currency absent from the shipped catalog, or another account field.
- Reward multipliers and milestones for rounds 11–15; the shipped multiplier table only has ten entries (`0.6` through `1.5`).
- Character EXP and guild-fund values for these exact NEO completions, plus callback ordering.

The operator approved calibrated approximation. The selected profile uses money base `48*level²`, contribution base `round(2173*level²/90²)`, guild EXP `2*contribution`, shipped factors `0.6..1.5`, and a `1.5` cap for rounds 11–15 over levels 30–160. `Kết Tinh...`, character EXP, and guild funds stay excluded until mapped. The Go command/cache path and rollback-verified SQL draft implement one atomic consume+reward+advance transaction; persistent migration capture/deploy remains a separately checkpointed step.

### Non-goals until evidence exists

- Do not invent build duration, progress percentage, workers, maintenance, or quest count gates.
- Do not infer guild-loop reward formulas/probabilities from website prose.
- Do not guess whether Hall and multiple Houses add, replace, or maximize member capacity.
- Do not expose construction-task activation without a proven RPC and durable progress model.

### Approval gate

The operator approved this addendum on 2026-10-28. This approval permits code and automated-test work only; migration apply, restart/deploy, original-server/live mutation, and invented mechanics remain prohibited by the existing safety gates.

## 6. Wave 2 — Atomic membership and contribution economy

**Files**:
- `internal/application/guild/service.go`, `service_extra.go`, `service_guild_helpers.go`, `service_guild_skills.go`
- PostgreSQL guild/character repositories and migration command functions
- Guild RTMP membership/contribution handlers

**Work**:
1. Replace multi-step creation with `CreateGuildCommand`; return committed guild, leader member, and wallet snapshot.
2. Replace lifecycle orchestration with atomic approve/leave/kick/disband commands. Correct affected-row and population checks.
3. Define character currencies 20/21 as canonical personal balances; update member ledger fields in the same transaction.
4. Add read-only drift diagnostics between character currencies and member contribution before deciding whether live reconciliation is safe. No automatic backfill in this wave.
5. Replace `ContribMoney` and `ContribMaterial` with transaction commands and emit callbacks only after commit.
6. Keep quest cancellation as post-commit idempotent cleanup; if it needs crash recovery, add a durable pending cleanup fact rather than extending a transaction across services.

**Exit**: Failure injection at every write point proves no lost wallet/item, free guild, duplicate membership, or half-applied restore. Returned and pushed totals come from committed rows.

## 7. Wave 3 — Atomic warehouse and guild skills

**Files**:
- `internal/application/guild/service_guild_skills.go`
- `internal/presentation/rtmp/handlers/guild/contrib_material.go`, `updates.go`
- PostgreSQL guild repository and migration functions

**Work**:
1. Implement move/merge/drop commands with stable row locking, page bounds, stack limits, ownership, and `canSlot` authorization.
2. Keep initial warehouse responder shape keyed by persistent slot ID and existing `sid/tid/stackNum` DTO keys.
3. Introduce canonical skill-family state or enforce a unique family mapping over existing `guild_skills`.
4. Build a catalog resolver that accepts only an exact-next same-`codeName` row with `useEnv=4` and level <=6.
5. Atomically debit `guildDevExp` and `guildDevMoney` and update the skill. Rows requiring unresolved `costGuildContrib` remain unavailable until payer semantics are proved.
6. Refresh skill state through the responder/read model; preserve `onGuildSkillDeveloped` only as an unresolved compatibility push until runtime evidence confirms it.

**Exit**: Concurrent move/merge/develop tests serialize correctly; arbitrary skill IDs and level skips fail; insufficient resources change nothing.

## 8. Wave 4 — Persisted buildings

**Files**:
- New domain/application building files under guild packages, split to remain under repository size limits
- PostgreSQL repository and migration for `player.guild_buildings`
- `internal/presentation/rtmp/handlers/guild/buildings.go`, `construction_rpc.go`, `build_rpc.go`

**Work**:
1. Add durable building state keyed by `(guild_id, extend_id)`, with unique active build ID and revision.
2. Replace `Handler.buildStates`; `createGuildBuildings` reads persisted rows and emits empty extension points only where no row exists.
3. Validate map 49, extension coordinates/type, target catalog row, create/upgrade mode, predecessor, prerequisites, family count limit, guild ownership, and permission.
4. Persist pending construction before any push. Finish/reset must verify actor guild and the exact pending row.
5. Debit catalog money/exp/material costs atomically at the approved lifecycle boundary. The recommended conservative boundary is finish: a pending row grants no building benefit, reset only removes pending state, and no unproven refund rule is needed. This specific economy choice still requires operator approval because client source does not establish original debit timing.
6. Do not couple building template level to `guilds.level`. Derive member cap/warehouse/academy availability from persisted functional buildings only where catalog `prop*` or two-source gameplay evidence defines the effect.

**Exit**: Restart reconstruction, cross-guild isolation, duplicate start, concurrent finish/reset, insufficient resources, prerequisite, and count-limit tests pass. No process-memory authority remains.

## 9. Wave 5 — Verification and documentation

1. Run focused unit/integration tests after each wave.
2. Run `go test ./internal/application/guild/... ./internal/infrastructure/persistence/postgres/... ./internal/presentation/rtmp/handlers/guild/...` and relevant quest/NPC tests.
3. Run `go vet ./...` and `go build -o bin/gameserver.exe ./cmd/gameserver`.
4. Run `git diff --check` and compare failures against recorded baseline; do not repair unrelated Achievement/Pet work.
5. For approved migrations: checkpoint backup first, apply through the compose Supabase path, verify privileges/RLS, rollback-smoke each command, shadow replay, and post-capture diff.
6. Only after separate restart approval, perform Flash smoke tests for panel load, create/apply/approve/leave, rank permissions, both contribution paths, warehouse move/merge, skill levels 1→2 and 6→blocked, construction restart recovery, and guild quest gate.
7. Update `docs/memory/guild-system.md`, `docs/ai-memory/WORK_STATE.md`, and `docs/ai-memory/FIX_HISTORY.md` after each completed implementation group.

## 10. Approval gates

Approval to implement this plan does **not** authorize migration apply, database mutation, restart, deployment, backfill, or use of live player resources.

Approved by the operator on 2026-10-27:

1. The six-wave code/schema scope.
2. Character currencies 20/21 as canonical personal contribution balances, with member columns retained as an atomic ledger.
3. Building costs debit atomically at finish; resetting pending construction is cost-neutral.
4. Skill development fails closed after level 6 and when `costGuildContrib` payer semantics are unresolved.

## 11. Completion criteria

- All 32 RPCs retain their names and tested argument contracts.
- Every confirmed client callback mismatch is corrected; unresolved pushes are neither silently removed nor claimed as verified.
- No guild economy mutation spans independent commits.
- Server-side authorization uses the same persisted rank settings displayed by the client.
- Contribution has one canonical balance plus an atomic membership ledger, with drift observable.
- Warehouse operations are transactional and concurrency-safe.
- Skill development enforces catalog, exact-next chain, permission, and resources.
- Buildings survive restart and are isolated by guild; no building is inferred from guild level.
- Existing guild quest behavior remains passing.
- No migration/runtime action occurs without its separate approval and checkpoint.
