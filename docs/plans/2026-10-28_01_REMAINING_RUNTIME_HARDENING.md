# Remaining Runtime Hardening — Implementation Plan

**Date**: 2026-10-28 · **Status**: Completed 2026-10-29
**Approval**: The operator explicitly requested continuing all remaining work without step-by-step prompting.
**Goal**: Remove every remaining evidence-safe runtime defect after the guild-loop E2E pass: repair Achievement RID 84 authority, eliminate stale full-suite expectations, make SendCombine purchase durable and atomic, implement authoritative guild disband, apply Hall level on building finish, and automate feasible race/concurrency checks. Features lacking protocol, identity, amount, or aggregation authority remain fail-closed rather than guessed.

Research base verified 2026-10-28:
- `docs/memory/achievement-system.md`: durable acquisition facts and event deduplication.
- `docs/memory/guild-system.md`: lifecycle commands, cache barriers, building catalog and callback contracts.
- `docs/memory/quest-loop-system.md`: completed atomic guild-loop authority and explicit reward exclusions.
- `docs/research/2026-10-27_02_GUILD_SYSTEM_AUDIT_RESEARCH.md`: guild client/catalog/server survey.
- Current catalog, compose DB, Go source, served decompiled Flash client, live logs, and three independent scoped audits in this session.

## 1. Confirmed mechanics

### 1.1 Achievement RID 84

- Catalog and live DB define RID 84 / AID 2017 as `Bắt pet thành công`, target 5000.
- Title 256 `Đại Sư Bắt Pet` is AID 2017's reward, not completion authority.
- Every validated `pet-captured:v1:<petID>` durable fact increments both RID 83/AID 2016 target 1 and RID 84/AID 2017 target 5000.
- Current title-state ownership of RID 84 emits target 1 and aborts the full login state reconciliation pass.
- No historical count may be inferred from current Pet roster or title ownership.

### 1.2 Full-suite stale expectations

- Quest requirement-policy census is 242: type1=75, type2=67, type4=94, type10=6; overlap and fingerprint diagnostics are empty.
- Chat stat callbacks use the current unified `BuildUPPPayload`; tests must compare callback values to the current character maxima, not obsolete formula literals.
- SendCombine test fixtures must load reward templates through `gamedata.Manager`; production purchase behavior still requires transactional hardening and cannot be fixed only by changing the fixture.

### 1.3 SendCombine purchase

- Purchase currency debit, all reward item grants, event-window validation and quota increment form one business transaction.
- Quota must survive restart and concurrent final-slot purchases.
- Cache/callback reconciliation occurs only after DB commit; failure leaves wallet, inventory and quota unchanged.
- Existing client RPC and callback names/payloads remain unchanged.

### 1.4 Guild disband

- Only the current leader may disband.
- Every member's currency 20/21 is zeroed and floor(90%) is stored in the existing restore fields, matching leave/kick semantics.
- Every member's `characters.guild_id` is cleared; guild-owned child rows disappear with the guild transaction.
- Exact global callback is `onDelGuild(gid)` after commit. Guild quests are abandoned idempotently for every former member.
- Multi-member locks use stable character-ID order and the shared cache mutation barrier.

### 1.5 Hall level

- Hall templates carry level property values 1–6, and shipped guild help independently states Hall level is guild level.
- Successful Hall finish atomically sets `guilds.level` to the completed Hall level. A finish may not skip, lower, or update the level on rollback.
- The committed guild refresh callback follows the existing global-safe contract.

### 1.6 Concurrency

- Linux Docker `go test -race` is the supported local race detector because the Windows host has CGO disabled and no C compiler.
- Database concurrency tests must prove one winner or safe serialization for duplicate guild-loop completion, lifecycle conflicts, warehouse conflicts, and building finish/reset.

## 2. Design

### 2.1 Authority registry repair

- Remove RID 84 from exact title state requirements.
- Classify RID 84 as event authority; census becomes 475 state / 81 event / 141 unsupported.
- Expand the Pet-captured fact update set to RID 83 and RID 84 under the same fact event key.

### 2.2 Durable SendCombine command

- Add a server-only runtime purchase ledger keyed by character, activity key and event version.
- Add one schema-qualified PostgreSQL command that locks wallet, ledger and inventory, validates the active window/quota/templates/capacity, grants all rewards, debits currency, increments quota and returns committed item/wallet state.
- Draft canonical SQL under `supabase/data/`; apply to compose only after a pre-apply checkpoint, then capture with `supabase db diff -f send_combine_atomic_purchase` and normalize unrelated drift.
- Route the handler through a narrow application/repository command and shared player mutation barrier.

### 2.3 Guild disband command

- Add `player.disband_guild_atomic` to canonical guild lifecycle SQL.
- Return a deterministic array of committed per-character states for cache reconciliation.
- Extend domain/application/PostgreSQL/cache command interfaces; flush and reconcile all affected online/offline players in stable order.
- Broadcast `onDelGuild(gid)` only after commit and clean guild quests for all former members.
- Capture schema/function change through `supabase db diff -f guild_disband_and_hall_level` after rollback/concurrency smoke.

### 2.4 Hall finish effect

- Resolve Hall target level from the validated building catalog in application code and pass it to the finish command.
- The PostgreSQL building finish command applies level only for the exact Hall family and returns the committed guild snapshot needed by callbacks/cache.

### 2.5 Automated verification

- Add deterministic unit and command tests first.
- Add PostgreSQL integration/concurrency tests using synthetic fixtures inside transactions or a disposable database; never mutate retained player data without checkpoint and cleanup.
- Run Docker Linux race tests for cache/guild/quest/PostgreSQL/handlers.

### 2.6 Touch points

| Touch point | Change |
|---|---|
| `internal/application/achievement/title_state.go` | Remove false RID 84 title ownership. |
| `internal/application/achievement/registry.go` | Reclassify RID 84 and update census. |
| `internal/application/achievement/facts.go` | Increment RID 83 and 84 from one capture fact. |
| Achievement tests/research/memory | Lock authority, targets, census and no login warning. |
| `cmd/questaudit/main_test.go` | Update audited census. |
| Chat handler tests | Assert current unified payload invariants. |
| `internal/presentation/rtmp/handlers/activity/send_combine_buy.go` | Replace multi-step mutation with atomic command. |
| `supabase/data/` SendCombine SQL | Durable ledger and atomic purchase function. |
| Item/cache/PostgreSQL adapters | Committed wallet/item reconciliation. |
| `supabase/data/guild_lifecycle_commands.sql` | Atomic multi-member disband. |
| Guild domain/application/PostgreSQL/cache/handler tests | Multi-member result, barrier, cleanup and callback. |
| `supabase/data/guild_building_commands.sql` | Atomic Hall-level update and committed guild return. |
| Building resolver/application/handler tests | Exact Hall family/level and refresh semantics. |
| `supabase/migrations/` | CLI-generated normalized migrations only. |
| `docs/ai-memory/*`, matching `docs/memory/*` | Checkpoints, fix history and final invariants. |

## 3. Non-goals and evidence blockers

- Guild invitation remains fail-closed: invitee callback, accept/refuse RPC, expiry and offline lifecycle are unknown.
- House capacity remains unchanged: two-House aggregation and interaction with default capacity 50 are unknown.
- Warehouse page count remains unchanged: catalog property does not prove set/add/cap semantics.
- Construction tasks remain unavailable: no building-to-quest selector, limits, progress model or reward command is authoritative.
- `Kết Tinh...`, character EXP, guild funds and rounds 5/10/15 extras remain excluded from guild-loop rewards until exact identity and amounts are captured.
- No historical RID 84 backfill from mutable roster/title state.
- No ActionScript or game-data JSON edits. No commit/push.

## 4. Verification

1. Focused unit tests for Achievement, questaudit, chat, activity, guild, cache, PostgreSQL and handlers.
2. Rollback/idempotency/ACL/shadow replay for every database function and generated migration.
3. `go test ./... -count=1`; the target is zero failures after stale expectations and SendCombine are corrected.
4. `go vet ./...` and `go build -o bin/gameserver.exe ./cmd/gameserver`.
5. Docker Linux `-race` across touched cache/application/PostgreSQL/handler packages.
6. Manual Flash smokes only after healthy deploy and exact pre-mutation snapshots: Achievement login warning absent, SendCombine success/failure rollback, two-client disband callback, Hall finish/relogin.
7. Exact cleanup of every synthetic/live delta and final runtime baseline comparison.

## 5. Completion record (2026-10-29)

- RID84 authority, stale full-suite assertions, durable atomic SendCombine, atomic multi-member guild disband and Hall-derived guild level are implemented and deployed.
- Applied migrations: `20261028174942_send_combine_atomic_purchase.sql` and `20261028175744_guild_disband_and_hall_level.sql`; history, shadow replay/post-diff and service-role-only ACL checks pass.
- Rollback/rejection and committed concurrency checks pass: SendCombine final quota has one winner; duplicate disband has one winner; Hall finish/reset serializes safely. Every synthetic/player delta was removed with guarded cleanup.
- Full `go test ./... -count=1`, `go vet ./...`, Windows gameserver build and scoped Docker Linux race suites pass. Deployed gameserver container `6b592e88356f...` has restart count 0, loads 118,540 game-data records, and returns HTTP 200 for health and metrics with no startup error.
- Final retained baseline matches the pre-test snapshot, including character3 wallet/progression/location/HP/MP, guild1 level/population/ledgers/currencies/buildings, empty SendCombine ledger and item sequence/max 2585.
- The login-only Flash smoke for observing the RID84 warning was not run because no interactive Flash operator was available. Unit, registry and full-suite coverage pass; no login warning was observed in available post-deploy logs, but the login path was not exercised in that observation window.
- Section 3 evidence blockers remain fail-closed. The configured SendCombine event ended before 2026-10-29, so a live success purchase was intentionally not attempted; expired rejection and atomic success were verified at the database command boundary.
