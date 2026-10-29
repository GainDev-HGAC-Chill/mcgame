# Fishing, Herb Gathering, and Farm Safety Research

**Date:** 2026-10-29  
**Scope:** Flash wire contract, shipped game data, current Go/PostgreSQL behavior, and original/Vietnamese gameplay references.  
**Implementation state:** Research only. No code, schema, database data, runtime, or deployment change was made for this scope.

## 1. Sources and authority

Primary contract and data sources:

- `docs/client/view/view/compMain/GatherProgressCanvas.as`
- `docs/client/view/view/compDragable/ProductPanel.as`
- `docs/client/view/view/compGameStage/NPCView.as`
- `docs/client/predef/GamePredef.as`
- `supabase/seeds/data_tbl_npc.sql`
- `supabase/seeds/data_tbl_item_template.sql`
- `supabase/seeds/data_tbl_equipt_template.sql`
- `supabase/seeds/data_tbl_skill.sql`
- Current Go farm/gathering services and handlers listed in section 7

Gameplay references:

- http://web.4399.com/mlxt/zx/31353.html
- https://vuaphapthuat.go.vn/bai-viet/cau-ca/4363.html
- http://vuaphapthuat.go.vn/bai-viet/hai-thuoc/4361.html
- http://vuaphapthuat.go.vn/bai-viet/trong-trot/4362.html

The Flash client is authoritative for RPC names and payloads. Seeds are authoritative for shipped IDs and catalog fields. Websites corroborate gameplay only; they do not define RPC or database contracts.

## 2. Flash wire contract

### 2.1 Resource query and gathering progress

- `queryFishPool(npcID)`, `queryHerb(npcID)`, and `queryGather(npcID)` return `{name, level, remain}`.
- Gathering starts through `onGathering({hint, delay, prop?})`; `delay` is milliseconds.
- `GatherProgressCanvas` only completes its local progress bar when the delay expires. The server closes it through `quitGathering`.
- The cancel button calls the misspelled inbound RPC `quitGatheringClinet()` and immediately hides the panel locally.
- Fishing state is `26`; farm harvest is `27`; herb gathering is `28`; idle reset is `1`.
- Life-skill properties are `fishDex`, `plantDex`, and `herbDex` for skill types `14`, `15`, and `16`.

The client does not prove the exact original-server order between the completion reward callback, `quitGathering`, state reset, and life-skill updates. A post-commit ordering must therefore be smoke-tested.

### 2.2 Generic product panel is a separate system

`ProductPanel` calls `onProductReady({npcId, gloveId})`, expects compact status values such as `e`, `ed`, `eg`, `em`, `eb`, `ea`, `ee`, and `el`, and repeats production automatically. It uses equipped slot `13`, glove production time, MP, and durability. This contract must not be copied onto herb gathering: gameplay independently specifies herb skill gating and two vigor per collection.

## 3. Shipped catalog

### 3.1 NPC families

| Family | NPC type | Shipped IDs | Notes |
|---|---:|---|---|
| Ordinary fishing | 21 | `1189..1218`, `1596..1598` | Actual nodes cover levels 1–7 only. |
| Fishing contest | 21 | `1573..1577` | Separate activity authority; not ordinary pools. |
| Golden fishing | 21 | `1635` | Separate special node. |
| Herb | 23 | `1241..1250` | Levels 1–10 exist only at placeholder map `200`; dynamic world-spawn authority is absent. |
| Generic product | 24 | catalog-driven | Uses `ProductPanel`, not gathering-progress semantics. |

### 3.2 Canonical ordinary fish materials

The level 1–7 pairs are corroborated by the Vietnamese gameplay table and seed level/type fields. For levels 8–10, the shipped contiguous expansion block contains three type-610 rows per level, but exactly one has the special-value marker `199`; the two ordinary rows retain the normal level-scaled value. Level 6 demonstrates the same structure: VPT names only the two ordinary rows and omits `2590 Sứa Biển` (`199`). This yields the explicit registry below without scanning all type-610 rows.

| Level | Ordinary material IDs and names | Explicitly excluded special row |
|---:|---|---|
| 1 | `2361` Cá Bống; `2362` Cá Chạch | — |
| 2 | `2363` Cá Trê; `2364` Lươn | — |
| 3 | `2365` Cá Đổng Quéo; `2366` Cá Bơn | — |
| 4 | `2367` Cá Trích; `2368` Cá Hồi | — |
| 5 | `2369` Cá Lóc; `2370` Cá Vược | — |
| 6 | `2589` Cá Tuyết; `2588` Cá Hồi Xám | `2590` Sứa Biển |
| 7 | `2591` Cá Gỗ; `2592` Cá Dao | — |
| 8 | `2594` Cá Dương Tử; `2595` Cá Đoạt Bảo | `2596` Cá Mập Medusa |
| 9 | `2597` Rùa Cạn; `2599` Tôm Sú | `2598` Cá Mập Trắng |
| 10 | `2600` Cá Voi Trắng; `2602` Hàu Bá Vương | `2601` Cá Kappa |

Only levels represented by an authoritative ordinary world node may be activated. The registry does not authorize inventing level 8–10 fishing spawns.

### 3.3 Canonical ordinary herb materials

| Level | Material IDs and names |
|---:|---|
| 1 | `2491` Ninh Thần Thảo; `2492` Mạn Thủy Đằng |
| 2 | `2493` Mê Tâm Thảo; `2494` Sa Cúc Căn |
| 3 | `2495` Anh Hồn Thảo; `2496` Lăng Tinh Hoa |
| 4 | `2497` Toái Diệp Thảo; `2498` Băng Phong Hoa |
| 5 | `2499` Thần Quang Thảo; `2500` Cang Cúc Diệp |
| 6 | `2604` Khiên Bán Đằng; `2603` La Cân Thảo |
| 7 | `2606` Thạch Trúc Căn; `2605` Thái Diệp Thảo |
| 8 | `2607` Dung Băng Hoa; `2608` Phục Ma Thảo |
| 9 | `2609` Thần Hộ Đằng; `2610` Mê Quang Diệp |
| 10 | `2611` Băng Liên Hoa; `2612` Thánh Quang Thảo |

VPT directly corroborates levels 1–7. Levels 8–10 are catalog-complete pairs, but cannot become reachable until herb spawn placement is authoritative.

### 3.4 Fishing equipment and bait

- Rod equipment has `type=1201`, equipped slot `13`, `pro_time` in seconds, and `succ_rate` as a percentage.
- Rods `1437` and `1438` use 20 seconds and 70/75 percent success; max durability is 1000.
- Bait/support items are type `515`, including `2391..2399` and `2982..2987`.
- Public gameplay and seed descriptions corroborate the main time reductions of 2/4/6/8 seconds and selected 5-percent success effects, with a minimum five-second duration.
- The server has no authoritative persisted active-effect model tying those items to a gathering attempt. Bait, net, osprey, spear, sesame oil, and other auxiliary effects therefore remain deferred.

## 4. Confirmed gameplay rules

- Fishing requires the Fishing skill at least equal to the node level and an equipped, non-broken fishing rod.
- Herb gathering requires the Herb skill at least equal to the node level and costs two vigor per collection.
- Successful same-level fishing/herb collection grants `+2` mastery; collecting one level lower grants `+1`; lower than that grants `+0`.
- Fishing success is governed by the rod success rate. The available sources do not establish a fishing vigor cost.
- The sources do not establish when rod durability is consumed; the safe implementation may reject a broken rod but must not invent a decrement rule.
- Planting requires Plant skill at least equal to crop level. Public gameplay says vigor is consumed and mastery is gained when planting, not when harvesting, but does not give an authoritative mastery amount in the available text.

## 5. Current defects

### 5.1 Fishing and herb

`internal/presentation/rtmp/handlers/npc/gathering.go` mutates the player before the timer:

- Herb increments mastery and grants hard-coded item `2492` before `onGathering`.
- Fishing subtracts an unverified one vigor, increments mastery, and grants a reward before `onGathering`.
- Cancel therefore keeps reward/mastery/cost mutations.
- Fishing reward selection is `2360 + level`, which is not the two-item ordinary registry and is invalid beyond the low levels.
- There is no life-skill-level gate, rod gate, broken-tool gate, or success-rate roll.
- Fishing uses hard-coded `19500ms`; herb uses hard-coded `5000ms`.
- Mutations use independent write-behind service calls rather than a serialized atomic completion command.

### 5.2 Farm

- Plant persists the plot (and may consume the money-tree seed) before separately subtracting 25 vigor.
- Harvest credits inventory before deleting/updating the plot.
- Harvest then adds a hard-coded `+20 plantDex` in a separate service call, although gameplay places mastery at planting and the exact amount is unverified.
- Immediate ripening subtracts nine Gold before changing/harvesting the plot.
- Uproot subtracts one Gold before deleting the plot.
- `doubleHarvest` currently doubles output for free; the booster identity/cost is not authoritative.
- Every sequence can partially commit and can race with write-behind state.

## 6. Atomicity and cache boundary

`PlayerCache.runPlayerCommand` is the established repository-adapter boundary. It serializes per-character commands, flushes dirty write-behind state before the database command, and reconciles committed character/item/skill snapshots into the hot cache without marking them dirty again.

The safe ownership model is:

1. Start handlers validate immutable catalog facts and create an in-memory cancellable timer only. No durable player mutation occurs at start.
2. Timer completion invokes one PostgreSQL command through a cached command repository.
3. PostgreSQL locks and revalidates current character, skill/tool/item/plot state and performs all effects in one transaction.
4. The result returns committed character, skill, item, equipment, and/or plot snapshots.
5. The cache adapter reconciles those snapshots.
6. Only then may the handler send reward, property, inventory, plot, `quitGathering`, and idle-state callbacks.

Cancellation deletes the pending timer and performs no command. Any command failure grants no reward and still closes/reset the gathering UI so the client cannot remain stuck.

## 7. Implementation touch points discovered

- `internal/presentation/rtmp/handlers/npc/gathering.go`
- `internal/presentation/rtmp/handlers/npc/farm.go`
- `internal/application/farm/service.go`
- `internal/domain/farm/repository.go`
- `internal/infrastructure/persistence/postgres/farm_repository.go`
- `internal/infrastructure/cache/mutation_barrier.go`
- `internal/infrastructure/cache/player_data.go`
- `internal/application/skill/life_skill.go`
- `internal/application/item/service.go`
- `cmd/gameserver/main.go`
- Existing farm migrations/functions under `supabase/migrations/`

The two NPC handler files are already too large. New registry, command, and completion logic must be split into focused files/packages rather than extending the monoliths.

## 8. Authority gaps and safe deferrals

- Exact original-server completion callback order: requires Flash smoke/live capture.
- Fishing vigor: no mutation until confirmed.
- Rod durability decrement timing: reject broken rods, but do not decrement yet.
- Special/contest/golden fishing rewards and rates: separate registries and captures required.
- Level 8–10 fish world nodes and all herb dynamic spawn placement: no invented spawns.
- Bait and auxiliary fishing effects: no active-effect authority.
- Farm planting mastery amount: remove the incorrect harvest `+20`; do not invent a planting value.
- Farm booster identity/cost: `doubleHarvest` must fail closed rather than remain a free mutation.
- Existing short/long crop timings and yields are preserved unless separately proven wrong; website prose alone is insufficient for exact values.
