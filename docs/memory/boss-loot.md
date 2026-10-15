# Boss Loot System

Unified loot table for ground/flying schedule bosses AND daily handbook bosses (the two systems target the same logical bosses with different ID spaces). Distinct from `creature_loot` which still serves trash/regular mob drops.

## Schema (canonical registry approach)

- `data.data_tbl_schedule_boss` is the single canonical boss registry.
  - `nid` is the canonical boss key. Use the world NPC ID (706–1556) for paired bosses; use the daily ID for daily-only bosses with `kind='daily_only'`.
  - New nullable column `daily_boss_id` — unique partial index `where daily_boss_id is not null`. Set on the paired schedule_boss rows so the daily handler can resolve `daily_id → nid`.
  - `kind` check expanded to `('ground','flying','daily_only')`. Domain eligibility explicitly accepts only active normal `ground|flying`; `daily_only` rows never enter map bootstrap, respawn, scene, or engagement paths.

- `data.data_tbl_boss_loot` (new) — one row per drop entry, keyed by `boss_nid` FK to schedule_boss.
  - `rate integer` on the **0–50000 scale, same as `creature_loot.rate`** (NOT 0..1). Drop fires when `rand.Intn(50000)+1 <= rate`.
  - `qty_min` / `qty_max` integer range, uniform roll between them; defaults are both 1.
  - `quality int default 0` (0 = use item template default), `bound bool`, `type int`.
  - **`type` follows the box-items award scheme**: `<30` → existing item path (e.g. 19 = equipment, 29 = generic item, 508/509 = quest items, 550 = pet-item-bag). `30` → basic currency (`item_id` ∈ `{0,1,2}` for money/gold/honor; `bound=true` routes to moneyBind/goldBind). `31` → experience (`item_id` ignored, qty range = EXP amount). `35` → game currency (`item_id` = currency type id; covers all 60+ wallet keys: `wisdonCrystal`=213, `decoSilver`=216, `warSprite`=229, `heiyaoshiPoint`=233, etc).
  - `tier_filter text` — null = drops for all tiers; otherwise must equal the boss's tier from schedule_boss (`normal|mythic|special`).
  - `source_filter text` — null = drops from both kill sources; otherwise must equal `schedule` or `daily`.
  - `qid int` — quest-only drop, same semantics as `creature_loot.qid`.
  - `notes`, `is_active`, `created_at` for admin authoring.

## DB functions (`supabase/migrations/20260515063522_boss_loot_functions.sql`)

| Function | Purpose | Caller |
| --- | --- | --- |
| `data.list_active_boss_loot()` | bulk load all active rows at boot | `BossLootConfigRepository.List` |
| `data.lookup_schedule_boss_by_daily_id(p_daily_boss_id)` | resolve daily ID to canonical nid | lazy-cached in `bossloot.Service.ResolveBossNIDByDailyID` |

Both are `language sql`, `security invoker`, `set search_path to ''`, schema-qualified per project convention.

## Go package layout

Mirrors the `scheduleboss` precedent — boss_loot does **not** live in `gamedata.Manager`.

- `internal/domain/bossloot/` — `types.go` (`LootEntry`, `Source` enum), `repository.go` (`ConfigRepo`, `MappingRepo`).
- `internal/infrastructure/persistence/postgres/boss_loot_repository.go` — implements both interfaces.
- `internal/application/bossloot/service.go` — caches all rows at startup keyed by `boss_nid`; exposes `EntriesForBoss(nid, source, tier) []LootEntry` (Go-side filtering) and `ResolveBossNIDByDailyID(ctx, dailyID) (int, bool)` (lazy DB call + `sync.Map` cache).

## Kill-site wiring

`internal/application/combat/boss_loot.go` runs at battle end alongside the existing creature_loot path.

Two detection mechanisms feed the same calculator:

| Trigger | Detection | BossNID source | Tier source |
| --- | --- | --- | --- |
| schedule kill | `battle.BossCtx` populated by the roster-aware map-boss path; legacy enemy-ID lookup remains a fallback | explicit schedule NID | `cfg.Tier` |
| daily kill | `battle.BossCtx` populated by handler | resolved from `daily_id` via `bossloot.Service.ResolveBossNIDByDailyID` | inherited from the linked schedule_boss row in the same resolver call |

The daily handler `BossDailyBattle` (`internal/presentation/rtmp/handlers/activity/boss_daily.go`) sets `battle.BossCtx = &combatdomain.BossContext{BossNID: scheduleNID, Source: "daily", Tier: tier}` immediately after `StartPVEBattleWithMultipleEnemies`. The Postgres function `data.lookup_schedule_boss_by_daily_id` returns both `nid` and `tier` so a daily-kill loot row with `tier_filter='mythic'` still fires when the linked schedule_boss is mythic.

`combat.Service.EndBattle` invokes both `ApplyItemDrops` (creature_loot) and `ApplyBossLootDrops`. Drops are appended to the same `result.ItemRewards` slice, so the existing battle-end client payload delivers boss drops without any RPC changes.

The roll uses the same quest-aware loop as creature_loot: `filterEligibleLootRecipients` for `qid`-gated drops, `anyRecipientNeedsItemForQuest` for the 5x rate boost on active collect-item quests (capped at 50000).

## Bootstrap wiring (`cmd/gameserver/main.go`)

After the schedule-boss block:

```go
bossLootRepo := postgres.NewBossLootConfigRepository(gameDb)
bossLootService := appbossloot.NewService(bossLootRepo, bossLootRepo, log)
if err := bossLootService.Load(ctx); err != nil {
    log.Warn("Failed to load boss loot config", zap.Error(err))
}
combatService.SetBossLootService(bossLootService)
bossDailyService.SetBossLootResolver(bossLootService)
```

`Load` is non-fatal — empty boss_loot table results in `ApplyBossLootDrops` short-circuiting at the `len(kills) == 0` check.

## Generated default rewards

`cmd/seed-boss-loot` validates reward IDs against the local item/equipment catalogs and emits `supabase/seeds/data_tbl_schedule_boss_loot.sql`. The filename sorts after `data_tbl_schedule_boss.sql`, whose truncate cascades into loot.

The checked seed contains 862 active rows and covers all 150 registry entries (71 map + 79 handbook), with 3–7 entries per boss. Profiles follow the handbook descriptions:

- ground: level-scaled materials/gems/rare progression, sub magic weapons from level 30, both dusts from level 55;
- flying: materials/feathers/rare progression/sub magic weapons/both dusts;
- mythic: stronger materials/gems/rare progression/sub magic weapons/both dusts and Wings of Light.

Level bands are 1–29, 30–54, 55–89, 90–119, 120–149, and 150+. All generated rewards are bound. The application service caches rows at boot, so a gameserver restart is required after DB authoring.

## Edge cases

- **Same boss killed by both signals**: if `battle.BossCtx.BossNID` matches an enemy in `scheduleboss.ConfigCache`, the kill is recorded only once (`seen` set in `detectBossKills`). Daily takes precedence (added first).
- **Daily kill with no daily_boss_id mapping**: `ResolveBossNIDByDailyID` returns `(0, "", false)`, `BossCtx` is left nil, no boss drops. Falls back to creature_loot only.
- **boss_loot row pointing to non-eligible item**: `resolveRewardItemType` returns 0 if the template doesn't exist or isn't recognized — the entry is silently skipped.
- **first_kill_only**: explicitly **not implemented** in v1. Schema has no such column. Add via migration + a new `player.boss_first_kill_log` state table if needed later.

## Admin authoring UI

`dashboard/src/app/dashboard/boss-loot/` mirrors the creature-loot editor — list page filters by kind (ground/flying/daily_only) + tier; the per-boss editor at `/dashboard/boss-loot/[nid]` supports three reward families via the same workspace:

- **Vật phẩm thường / Trang bị / Quest** — item picker queries `vw_item_source` (icon-resolved).
- **Tiền tệ cơ bản (type 30)** — three rows for money/gold/honor; the `Khóa` toggle on each drop routes to moneyBind/goldBind.
- **Kinh nghiệm (type 31)** — single row, qty range becomes the EXP amount.
- **Tiền tệ wallet (type 35)** — full list of 60+ game currencies (cbM=3, wisdonCrystal=213, decoSilver=216, warSprite=229, heiyaoshiPoint=233, …) sourced from `BOX_ITEM_GAME_CURRENCY_OPTIONS` in `dashboard/src/lib/box-item-awards.ts`.

Per-row controls: rate (1–50000), qtyMin/qtyMax range, quality (item only), `tier_filter` and `source_filter` dropdowns, `qid` quest gate, `bound` switch, optional admin `notes`. Save replaces the entire row set under `boss_nid` atomically (DELETE + bulk INSERT).

API: `GET/PUT /api/boss-loot/[nid]`, `GET /api/boss-loot` (listing), `GET /api/boss-loot/item-options` (icon-resolved item search). All routes accept the same fields the server expects in `data_tbl_boss_loot`.

## Battle-end DTO surface

- `BattleResult.CurrencyRewards []CurrencyDrop` + `CharacterReward.CurrencyRewards` carry the rolled currency/exp grants.
- `BattleResult.ToDTO()` / `ToDTOForCharacter` emit a `currencies` array alongside `items`. Each entry has `clientKey`, `delta`, `total`, `isExp`, `leveledUp`.
- `internal/presentation/rtmp/handlers/combat/reward_callbacks.go::sendBattleRewardCurrencyCallback` sends one `onAddMoney(cid, clientKey, delta, total)` per currency drop and `onAddExp(cid, delta, total)` for experience.

## Schedule attribution invariant

Map encounters intentionally keep creature CIDs as enemy `EntityID` so combat/catalog lookups remain correct. Canonical schedule attribution must therefore come from `battle.BossCtx`; both the kill hook and boss-loot detector prefer it and deduplicate any legacy fallback match.
