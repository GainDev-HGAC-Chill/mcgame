# Fishing, Herb Gathering, and Farm Atomic Safety — Implementation Plan

**Date:** 2026-10-29 · **Status:** Approved  
**Goal:** Make ordinary fishing, herb gathering, and existing farm mutations cancellation-safe, catalog-gated, and transactionally atomic while preserving the Flash RPC contract. Remove reward-before-timer and partial-commit paths, and fail closed for special pools, unplaced herb nodes, unverified fishing costs/durability, and free farm boosting.

Research base (verified 2026-10-29):

- `docs/research/2026-10-29_03_FISHING_HERB_FARM_RESEARCH.md` — wire contract, ID registries, gameplay evidence, defects, and authority gaps.
- `docs/memory/farm-life-skill.md` — current farm domain and life-skill integration.
- `docs/client/view/view/compMain/GatherProgressCanvas.as` — timer/cancel/close behavior and the `quitGatheringClinet` typo.
- `docs/client/view/view/compDragable/ProductPanel.as` — separate generic-product contract.

## 1. Confirmed mechanics

### 1.1 Wire contract

- `queryFishPool(npcID)`, `queryHerb(npcID)`, and `queryGather(npcID)` return `{name, level, remain}`.
- Start uses `onGathering({hint, delay, prop?})` with a millisecond delay.
- Cancel is inbound `quitGatheringClinet()`; server close is outbound `quitGathering`.
- Character states are fishing `26`, farm harvest `27`, herb `28`, and idle `1`.
- Skill types/properties are Fishing `14`/`fishDex`, Plant `15`/`plantDex`, and Herb `16`/`herbDex`.
- Generic type-24 product gathering remains on `ProductPanel` and is not converted into the fishing/herb path.

### 1.2 Ordinary material registry

The implementation uses an explicit registry, never a scan by item type.

| Level | Fish IDs | Herb IDs |
|---:|---|---|
| 1 | `2361`, `2362` | `2491`, `2492` |
| 2 | `2363`, `2364` | `2493`, `2494` |
| 3 | `2365`, `2366` | `2495`, `2496` |
| 4 | `2367`, `2368` | `2497`, `2498` |
| 5 | `2369`, `2370` | `2499`, `2500` |
| 6 | `2589`, `2588` | `2604`, `2603` |
| 7 | `2591`, `2592` | `2606`, `2605` |
| 8 | `2594`, `2595` | `2607`, `2608` |
| 9 | `2597`, `2599` | `2609`, `2610` |
| 10 | `2600`, `2602` | `2611`, `2612` |

Special fish `2590`, `2596`, `2598`, and `2601` are explicitly excluded. Contest NPCs `1573..1577` and golden NPC `1635` are not ordinary nodes. Actual ordinary fishing NPCs only authorize levels 1–7. Herb templates `1241..1250` are placeholder-map records and do not authorize invented world placement.

### 1.3 Gates and progression

- Fishing starts only when skill level is at least node level and an equipped, non-broken type-1201 rod exists in slot 13.
- Fishing delay is the equipped rod `pro_time` converted from seconds to milliseconds. No bait reduction is applied in this plan.
- Fishing success uses the rod `succ_rate` percentage. Fishing vigor is not charged.
- Herb starts only when skill level is at least node level and completion can pay two vigor.
- On successful fishing/herb completion, mastery gain is `+2` for same level, `+1` for one level below the skill, otherwise `+0`.
- Failed fishing produces no reward/mastery. Herb has no evidenced random failure rate.
- Cancel produces no durable mutation.
- Broken rods are rejected. Rod durability is not decremented because timing/cost authority is absent.
- Planting retains the existing 25-vigor gate, seed/crop/plot rules, and configured timing/yield behavior, but all durable effects become one command.
- The current harvest-time `+20 plantDex` is removed. No planting mastery is added until its exact amount is authoritative.
- `DoDoubleHarvest` fails closed without changing the plot until booster identity/cost is authoritative.

## 2. Design

### 2.1 Single source of truth

Add a focused application registry (for example `internal/application/gathering/registry.go`) containing immutable ordinary pairs and explicit node-family allow-lists:

```go
type ResourceKind uint8

type LevelRewards struct {
	Level       int
	TemplateIDs [2]int
}

var ordinaryFishRewards = [...]LevelRewards{
	{Level: 1, TemplateIDs: [2]int{2361, 2362}},
	{Level: 2, TemplateIDs: [2]int{2363, 2364}},
	{Level: 3, TemplateIDs: [2]int{2365, 2366}},
	{Level: 4, TemplateIDs: [2]int{2367, 2368}},
	{Level: 5, TemplateIDs: [2]int{2369, 2370}},
	{Level: 6, TemplateIDs: [2]int{2589, 2588}},
	{Level: 7, TemplateIDs: [2]int{2591, 2592}},
	{Level: 8, TemplateIDs: [2]int{2594, 2595}},
	{Level: 9, TemplateIDs: [2]int{2597, 2599}},
	{Level: 10, TemplateIDs: [2]int{2600, 2602}},
}
```

Herbs use the matching table from section 1.2. Selection is server-side uniform selection between the two ordinary entries. Registry validation must assert every configured item exists and has the expected catalog level/type, and that excluded special IDs never appear.

### 2.2 Gathering lifecycle

Create `internal/application/gathering/` with a narrow service and command repository.

Start result:

```go
type StartResult struct {
	Kind         ResourceKind
	NPCID        int
	ResourceLevel int
	ToolItemID   int64
	Delay        time.Duration
}
```

Start performs read-only validation and returns timer metadata. It does not charge vigor, add mastery, grant items, or change durability. A pending task captures character ID, NPC ID, resource level, selected reward ID, tool item ID where applicable, and a server-generated success roll. Replacing/cancelling a task only closes its channel.

Completion command/result:

```go
type CompletionCommand struct {
	CharacterID     int64
	Kind            ResourceKind
	NPCID           int
	ResourceLevel   int
	RewardTemplateID int
	ToolItemID      int64
	Succeeded       bool
	VigorCost       int
	MasteryGain     int
}

type CompletionResult struct {
	Character *character.Character
	Skill     *skill.CharacterSkill
	Reward    *item.Item
	Tool      *item.Item
	Succeeded bool
}
```

`player.complete_life_gathering_atomic` must lock and revalidate the character, matching learned life skill, supplied equipped rod for fishing, available herb vigor, and reward ID allowed by the command's kind/level. It deducts herb vigor and, on success, inserts/stacks exactly one reward and synchronizes dex plus `character_skills.exp`. Failure returns committed character state without reward/mastery. Any validation error rolls back everything.

The PostgreSQL adapter maps stable SQL errors. `CachedGatheringRepository` wraps the command in `runPlayerCommand`, flushes write-behind state, and reconciles committed character/skill/item/tool snapshots without dirty writes.

Completion callback sequence, all after commit:

1. Reward and inventory callbacks when successful.
2. Vigor/mastery and life-skill callbacks from the committed snapshot.
3. `quitGathering`.
4. `onSetCharState(... state: 1)`.

On command failure, omit reward/property callbacks but always attempt steps 3–4. This order is the safest UI-preserving default and remains a required Flash smoke because original-server ordering is unavailable.

### 2.3 Farm atomic commands

Extend `internal/domain/farm/repository.go` with a separate command capability so read-only repository users remain narrow:

- `PlantAtomic`: lock character, inventory seed when required, and conflicting plot rows; validate vigor/seed/plot uniqueness; deduct 25 vigor, consume seed, and insert plot atomically.
- `HarvestAtomic`: lock plot and harvester inventory; revalidate readiness/shared-harvest eligibility; add/stack reward and update or delete the plot atomically. Do not mutate plant mastery.
- `RipenAndHarvestAtomic`: lock character/plot/inventory; validate ownership and Gold; charge the preserved nine-Gold compatibility cost, ripen, reward, and delete in one transaction.
- `UprootAtomic`: lock character/plot; validate ownership and Gold; charge the preserved one-Gold compatibility cost and delete in one transaction.
- `DoubleHarvestAtomic` is not exposed in this wave; the handler returns a clear unavailable message and leaves state unchanged.

Each result returns the committed character, affected inventory item/seed, and plot state needed by callbacks. The cached farm command adapter uses `runPlayerCommand` for the acting character and reconciles all committed player-owned rows. Farm plot rows are database-owned and are returned directly; they are not added to `PlayerData`.

Withered-plot cleanup remains a database-owned delete, but mutation entry points must not perform cleanup as an untracked pre-command side effect. Atomic functions handle a withered row under lock as absent/available according to the existing farm rule.

### 2.4 SQL and migration workflow

Draft canonical functions in `supabase/data/farm_life_skill_commands.sql` using `security invoker`, `set search_path to ''`, and fully schema-qualified relations. Apply only after a pre-apply checkpoint and rollback smoke. Capture the resulting schema with:

```text
bin/supabase.exe db diff --db-url "$DB_URL" -f farm_life_skill_atomic_commands
```

Normalize only unrelated generated drift, verify shadow replay, and restrict function execution to `service_role`. No hand-written migration is allowed.

### 2.5 Touch points

| Touch point | Change |
|---|---|
| `internal/application/gathering/registry.go` | Add explicit ordinary fish/herb registries, special exclusions, node gates, and validation. |
| `internal/application/gathering/service.go` | Add read-only start validation, delay/success/mastery calculation, and completion command orchestration. |
| `internal/application/gathering/*_test.go` | Test registry, levels, rods, rates, delays, vigor, mastery, cancellation inputs, and exclusions. |
| `internal/domain/gathering/repository.go` | Define completion command/result repository capability. |
| `internal/domain/farm/repository.go` | Add atomic farm command/result capability without widening basic reads. |
| `internal/application/farm/service.go` | Route plant/harvest/ripen/uproot through command methods; remove split item/character writes and harvest `+20` mastery. |
| `internal/infrastructure/persistence/postgres/gathering_repository.go` | Call and scan the gathering function; map stable errors. |
| `internal/infrastructure/persistence/postgres/farm_repository.go` | Implement atomic farm commands and committed result scans. |
| `internal/infrastructure/cache/cached_gathering_repo.go` | Use `runPlayerCommand` and reconcile committed gathering snapshots. |
| `internal/infrastructure/cache/cached_farm_repo.go` | Use `runPlayerCommand` and reconcile committed farm snapshots. |
| `internal/infrastructure/cache/player_data.go` | Add lock-internal committed reconciliation helpers for character, skill, and item snapshots. |
| `internal/presentation/rtmp/handlers/npc/gathering.go` | Keep query/product helpers; remove pre-timer mutation and delegate fishing/herb lifecycle. |
| `internal/presentation/rtmp/handlers/npc/gathering_tasks.go` | Own cancellable task lifecycle and post-commit callbacks in a focused file. |
| `internal/presentation/rtmp/handlers/npc/farm.go` | Delegate atomic farm operations and fail closed for free double harvest. |
| `internal/presentation/rtmp/handlers/npc/farm_harvest.go` | Move timer/completion callback logic out of the oversized farm handler. |
| `cmd/gameserver/main.go` | Wire PostgreSQL repositories through cached command adapters into services/handlers. |
| `supabase/data/farm_life_skill_commands.sql` | Canonical atomic function definitions and ACL. |
| CLI-generated migration | Capture only approved function/schema changes. |

## 3. Non-goals

- Special fish, contest `1573..1577`, and golden node `1635`: rates/rewards are not authoritative, so ordinary fishing remains safe without them.
- Level 8–10 fishing spawns and all herb world placement: catalog rows do not prove map placement; no NPC is invented.
- Fishing vigor cost: absent authority means no charge is safer than preserving the current invented `1`.
- Rod durability decrement: broken rods are rejected, but durability is unchanged until decrement timing is captured.
- Bait/support effects (`2391..2399`, `2982..2987`), nets, osprey, spear, and sesame oil: require an authoritative active-effect lifecycle.
- Farm planting mastery amount: gameplay proves timing but not amount; remove incorrect harvest mastery and defer the replacement value.
- Booster-backed double harvest: free doubling is disabled until item identity, consumption order, and eligibility are proven.
- Rebalancing crop durations/yields or one-active-plot policy: preserve existing behavior unless separate evidence proves it wrong.
- Generic type-24 product gathering: separate `ProductPanel` contract and separate atomic-hardening scope.
- Exact original-server callback order: use the safe post-commit order and keep methods logged until Flash smoke confirms it.

## 4. Verification

1. Registry tests:
   - every ordinary pair exists with correct level/type;
   - excluded special IDs never appear;
   - ordinary fish nodes activate only catalog NPC levels 1–7;
   - placeholder herb rows do not create world spawns.
2. Gathering service/handler tests:
   - missing/low skill, missing/wrong-slot/broken rod, and unsupported NPC reject without mutation;
   - rod seconds become milliseconds and success rate boundaries are deterministic under an injected RNG;
   - cancel before expiry performs zero repository commands;
   - herb insufficient vigor rejects atomically;
   - success/failure mastery and reward branches match the confirmed rules;
   - command errors still close progress/reset state and never send reward.
3. Farm tests:
   - seed/vigor/plot failure leaves every row unchanged;
   - harvest inventory failure retains the plot;
   - simultaneous owner/shared harvest has one valid serialization result;
   - ripen/uproot failure never charges Gold;
   - harvest no longer adds `+20 plantDex`;
   - double harvest returns unavailable with no plot mutation.
4. Cache tests: dirty state flushes before each command; committed character/skill/item snapshots are clean after reconcile; failed commands do not reconcile.
5. PostgreSQL rollback tests: success and every rejection path for gathering, plant, harvest, shared harvest, ripen, and uproot. Verify exact retained player/plot/item sequence baselines afterward.
6. Concurrency tests against disposable/synthetic rows, with a checkpoint and guarded cleanup, for duplicate plant, double completion, shared harvest, ripen-vs-harvest, and uproot-vs-harvest.
7. Build and static verification:

```text
go test ./internal/application/gathering ./internal/application/farm ./internal/infrastructure/cache ./internal/infrastructure/persistence/postgres ./internal/presentation/rtmp/handlers/npc -count=1
go test ./... -count=1
go vet ./...
go build -o bin/gameserver.exe ./cmd/gameserver
git diff --check
```

8. Flash smoke, with new/changed RPC methods left logged:
   1. Click a fish node without a rod, with a broken rod, with low skill, and with a valid rod.
   2. Start and cancel before expiry; verify no fish/mastery/cost change.
   3. Complete multiple valid attempts; verify 20-second rod timing, both success/failure behavior, ordinary rewards only, and UI closure.
   4. If an authoritative herb node is available, repeat low-skill/low-vigor/cancel/success checks and verify exactly two vigor on completion.
   5. Plant with insufficient vigor/seed and with valid inputs; verify plot, vigor, and seed commit together.
   6. Harvest owner/shared plots and exercise immediate ripen/uproot; verify no partial charge/reward and correct plot visuals.
   7. Invoke double harvest and verify the unavailable message with unchanged yield.
9. After implementation, update `docs/memory/farm-life-skill.md`, `docs/ai-memory/WORK_STATE.md`, and `docs/ai-memory/FIX_HISTORY.md`. Do not add methods to `quietMethods` until the operator confirms the wire smoke.
