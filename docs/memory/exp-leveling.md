# EXP & Level-Up System

## Centralized EXP Callbacks

`SendExpAndLevelUpCallbacks(sender, char, expGained, leveledUp)` in `internal/presentation/rtmp/utils/exp_helper.go`.

Callback protocol (order matters):

1. `onUPP` -- sends full `char.ToDTO()` with `expSkill` override and nested `property` map with refreshed stats. Must be sent first.
2. `onPlayerLevelUp` (only if leveled up) -- character ID, cumulative `expSkill`, attr points, level.

`lvUp` is a client-to-server RPC, not a callback. Never send `SendCallback("lvUp", ...)`: the live Flash client has no `CallBack.lvUp`, causing #2095/#1069. Client calls `lvUp(currentLevel, passwordHashOrNull)` with a Boolean RPC responder. Keep the server RPC registration; use only existing `onUPP` / `onPlayerLevelUp` pushes for level updates. The decompiled `CallBack.as` is incomplete, so automatic visual effects from `onPlayerLevelUp` are not independently verified.

2026-10-11: removed invalid callback from both the shared EXP helper and offline EXP claim refresh. Regression tests assert callback names/order with and without equipment. Live level-up must be retested after deployment; no player progression or password changes are part of this fix.

Callers: Quest handler, Chat handler (admin `/stat exp`), Character handler (client `lvUp` RPC).

## Level-Up Stat Refresh Fix

Fixed stale stats in level-up `onUPP` by sending nested property-aware payload. `BuildUPPPayload` helper sends both top-level player fields and nested `property` data with refreshed `final*` fields.

## changeProperty onUPP Noise Fix

`changeProperty` must not send the full top-level `BuildUPPPayload` unchanged. Flash `CallBack.onUPP` treats top-level `bagSlotNum`, `bankSlotNum`, and `petMaxNum` as slot-expansion events and shows mid-screen notices whenever those keys are present, even if the RPC only changed stats. The fix is a stat-refresh-specific payload that keeps the nested `property` map and refreshed stat fields but strips unrelated notice-trigger keys before sending `onUPP`.

## bagSlotNum / bankSlotNum / petMaxNum Semantics

Flash client expects `bagSlotNum` as the number of bag tabs, `bankSlotNum` as the number of bank tabs, and `petMaxNum` as the direct pet-count cap. Server-side code should use `char.MaxBagSlots()` and `char.MaxBankSlots()` when actual slot capacity is needed.

Admin and runtime writes must treat `BagSlotNum` / `BankSlotNum` as the authoritative values. Raw slot totals are derived as `bagSlotNum * 30` and `bankSlotNum * 30`; legacy `bagSlots` / `bankSlots` inputs should only be accepted when they cleanly map back to that 30-slot page size.

The purchasable regular bag count is capped at 7 pages. Bag pages 8 and 9 are the fixed quest-item and pet-item pages and are not part of `bagSlotNum`. Bank count is capped at 5 pages.

## /stat Chat Command

Syntax: `/stat <field> <amount>` where field is `money`, `moneybind`, `gold`, `goldbind`, `exp`, or `level`. `/add` and `/stat` require `character.GMLevel >= 5`.

Callbacks for currency: `onAddMoney(characterID, fieldKey, amount, currentTotal)` and `onUPP({ fieldKey: currentTotal })`. For EXP: `onUPP({"expSkill": amount})`, plus level-up callbacks if applicable. `/stat level <target>` is upward-only, zeroes `expSkill`, and reuses the same `onUPP` and `onPlayerLevelUp` callback flow as a normal level-up refresh.

## Character GM Level

Flash client GM badge visibility is driven by `data.gmLevel > 0` in `CharactorInfoPanel`. Server-side `gmLevel` is stored on `player.characters.gm_level`, must be scanned through `player.list_character_details()`, and should be present in both `Character.ToDTO()` payloads and the login `cData` object.

## Gold/GoldBind int4 Overflow Fix

Database columns `gold` and `gold_bind` in `player.characters` were `integer` (int4, max 2,147,483,647). Large gold values from `/stat gold` commands caused persistent save failures blocking all character data for that player.

Fix: migrated both columns to `bigint` (migration `20260403034544_character_gold_bigint.sql`). Updated Go domain model `Gold`/`GoldBind` from `int` to `int64`. Removed unnecessary `int()` casts across shop, quest, trade, pet, and chat handlers.

## Administrative absolute target level (2026-10-12)

- `POST /api/admin/players/action`: `{"action":"set_target_level","targetId":<character ID>,"payload":{"targetLevel":<integer 1..MaxLevel>}}`. Invalid/fractional/null/missing/out-of-range targets fail; equal level preserves EXP and all state.
- `Character.SetTargetLevel` validates point accounting (`remaining + distributed == maximum`, maximum at least the old level's earned points) before mutation and delegates raising to `RaiseLevelTo`. Lowering removes one raw base point per lost level, resets EXP, refunds distributed allocations, rebuilds target-level remaining/max points and shared base/final stats, and caps current resources. Bonus points above `(oldLevel-1)*5` are retained even at target level 1. Assets, class aptitude, evolution, quests/history and owned skills are not modified.
- Actual `AllocateAttribute` stores allocations in `Apt*` (unlike older aptitude documentation). Refund is safe only when the recorded distributed total matches those fields; mixed/unidentifiable aptitude plus allocation is rejected without mutation. Unallocated aptitude bonuses are preserved. A per-stat allocation ledger would be needed to support ambiguous legacy characters safely; no guessed destructive refund is performed.
- `PlayerCache.UpdateCachedCharacterConditional` retains the registry read lock throughout the player lock and commit, so eviction cannot remove membership before a successful transition is committed. Current character and inventory are detached under the same player lock; equipment template preflight and character mutation run within that lock without repository calls. Rejection/no-op preserves both dirty flags and revision. Same-level decisions use this authoritative state, never the prefetched HTTP character snapshot. Invalid or unknown equipped items block downgrade. Item/pet repositories are never mutated.
- Mutation runs against the current registered cache state even without a connected client. **BLOCKED: uncached/offline target changes** — existing repositories offer no atomic lifecycle/revision guard against simultaneous login/eviction and a stale full-row write. The target action deliberately returns HTTP 409 without any database write (including same-level requests) instead of advertising unsafe offline success. A missing conditional updater returns 503; existing `PlayerCache` wiring supplies it automatically through interface detection. Online `onUPP` uses stat-refresh-only payloads with current equipment/feature bonuses; callback failure is reported as `live_session_callback_failed`. Cached success does not certify write-behind durability.
- Wiring: `adminServer.SetProgressionStatProvider(itemService)` must be configured in bootstrap. Legacy `set_progression.level` and `set_attributes.level` are rejected with instructions to use the target action; no unsafe direct lowering remains in those actions.
- Re-review equip race closure: cached `item.Service.EquipItem` detects `CachedItemRepository.UpdateEquippedSlotsConditional`. Eligibility is rechecked against the **current** character, incoming item and inventory under the same registry/player locks as downgrade; both incoming and displaced item moves are committed in one revision. A preliminary equip read cannot authorize a later stale commit. Generic cached moves/updates cannot introduce equipment outside this path; generic equipped-slot drag moves reject before any partial swap, leaving explicit equip/unequip controls as the supported path. Unregistered cache equipment operations fail closed without delegate persistence.
- `syncHPMPAfterEquipChange` now invokes `SyncEquipmentResources`, which computes bonuses outside cache locks and patches only current HP/MP inside a conditional current-character mutation. It never calls `charService.Update` with the handler's stale whole-character snapshot; downstream handler refresh uses the returned current character. Tests pause equip after preliminary read, commit downgrade, then resume and assert level rejection with both items unmoved; the opposite ordering also blocks downgrade. Handler regression confirms resource sync cannot restore stale level/wealth.
- Focused domain/admin HTTP/cache tests PASS locally, including bonus points, stale same-level prefetch, no-op/error dirty tracking, equipment locking and coordinated eviction. Vet PASS. Race testing BLOCKED by disabled CGO in the local environment. No deployment, live player mutation, grant, or Flash test was performed.
