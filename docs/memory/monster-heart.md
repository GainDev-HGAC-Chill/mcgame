# Monster Heart (Ma Vật Tâm)

Baseline-level DTO repair deployed17:34:09 (2026-10-12); tests/vet/build PASS, game running/restart_count0. No DB repair or synthesized heart rewards. Flash retest NOT TESTED. [Batch results](../research/2026-10-12_09_PANEL_PROTOCOL_ECONOMY_TRIAGE_RESEARCH.md).

## Panel-open contract correction (2026-10-12)

- `MonsterHeartPanel.as:4825–4846` indexes `TBL_CREATUREH_CONTAIN` using `box*10 + heartBoxLev[box]` and reads `.exp`/`.num` without a null guard. The old baseline level `0` selected nonexistent IDs `10/20/30/40/50`, causing `freshenNeedExp` to dereference null.
- Baseline box levels are now **1**, matching the actual 30 exported templates (`11–16`, `21–26`, ..., `51–56`). Decode preserves valid persisted levels `1–6`; missing, zero, negative or out-of-range levels fall back to baseline in the loaded representation only. No repair is written to the DB by panel initialization.
- Tests verify baseline IDs against `docs/database/game_data/TBL_CREATUREH_CONTAIN.json`, persisted-level normalization, and the init responder's five baseline levels. Focused application/handler/activity suites pass; real Flash validation is not performed.
- Runtime client dependencies remain `TBL_CREATUREH_CONTAIN` (133), `TBL_CREATUREH_POINT` (134), `TBL_CREATUREH_HEART` (131), `TBL_CREATUREH_COMBINE` and item template `5884`. The contain export exists; if the running client lacks its selected template, a server state fallback cannot restore that asset/catalog. Do not fabricate contain costs, capacities, rewards, or higher levels to hide missing client data.

## Feature Overview

Panel system where the player drags monster-heart items from a bag into a 5-box × 7-hole grid. The panel state (`{bag, data}`) is persisted in the generic feature-state store under `FeatureMonsterHeart = "monster_heart"`.

## Protocol

- `initMonsterHeartData` → `_result {bag, data}` (NOT a push; uses Responder in AS3).
- `monsterHeartSet [bagItemID, holePos, box, type]` → no Responder; server pushes `onInitViewProp` then `updateMonsterHeartPanel [{bag, data}]`.
- `monsterHeartReMove [holePos, box]` → no Responder; server pushes `onInitViewProp` then `updateMonsterHeartPanel [{bag, data}]`.

`type` (4th arg of `monsterHeartSet`) is the item's `monsterHeartType` category — advisory, not validated server-side (bag ownership check covers security).

## Wire Shape

```
{
  bag: {
    ji: {<itemId>: {itemId, n}, ...},
    lon: {...},
    mo: {...},
    ren: {...},
    shou: {...},
    te: {...},
    zhi: {...}
  },
  data: {
    hadActCombine: {1: {<itemId>: {color, talent}}, ..., 5: {...}},
    hadActHole:    {1: {1: itemId|-1, ..., 7: itemId|-1}, ..., 5: {...}},
    heartBoxLev:   {1: 1, ..., 5: 1}
  }
}
```

Empty hole value is `-1`.

## Open Questions

**OQ10 resolution**: `initMonsterHeartData` returns real state (populated `{bag, data}`) via `_result`. The two connect-time `_error` replies in the live log were transient server flakes — the panel-open call succeeds. Mirroring `_error` would break the panel. Decision: implement real init.

## Code Layout

- Application: `internal/application/monsterheart/`
  - `state.go` — `State`, `NewState()`, `DecodeState()`, `Encode()`, bag/hole helpers.
  - `helpers.go` — `intKey`, `parseIntKey`, `anyInt`, `IsValidBox`, `IsValidHole`.
  - `service.go` — `Service.Load`, `SetSlot`, `RemoveSlot` using `FeatureMonsterHeart` feature-state key.
- Handler: `internal/presentation/rtmp/handlers/monsterheart/`
  - `handler.go` — `Handler`, `RegisterHandlers`, `charIDFromCtx`, `pushViewProps`.
  - `init.go` — `InitMonsterHeartData`.
  - `actions.go` — `MonsterHeartSet`, `MonsterHeartReMove`, `toInt` helper.
  - `wire.go` — `stateToWire`, `emptyPanelWire`.
  - `init_test.go` — basic empty-state assertion.

## Notes

- `initMonsterHeartData` was previously a stub in `internal/presentation/rtmp/handlers/activity/icon_panels.go`. It was moved to the new monsterheart handler package (M8 Batch 2 refactor). The activity handler's `RegisterHandlers` no longer registers it.
- No currency is spent by set/remove. No `onAddMoney`/`onMinusMoney` callbacks needed.
- `pushViewProps` emits `onInitViewProp` after set/remove (matching live sequence). `onUPP` is NOT sent (the live log shows `onInitViewProp` only for these two RPCs).
- Security: `bagItemID` is validated against player-owned bag (`BagContains`); `box` and `holePos` are validated against server-side allow-lists before any state mutation.
- `type` arg (monsterHeartSet arg[3]) is accepted but not validated server-side per research note OQ5.
- `FOR UPDATE` is not used since the feature-state store uses `UpsertCharacterFeatureState` (single-row upsert on PK); concurrent double-send risk is low for UI drag operations.
