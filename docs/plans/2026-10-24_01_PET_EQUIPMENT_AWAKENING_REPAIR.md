# Pet Equipment and Awakening Repair — Implementation Plan

**Date**: 2026-10-24 · **Status**: Implemented · Runtime smoke-test pending  
**Goal**: Restore Pet equipment equip validation and its level/color/bind operations, then make Awakening bonuses cumulative, immediately visible, and able to authorize the transformed `exSid` skills that the Flash client sends into combat.

Research base (verified 2026-10-24):

- [`docs/research/2026-10-24_01_PET_EQUIPMENT_AWAKENING_RESEARCH.md`](../research/2026-10-24_01_PET_EQUIPMENT_AWAKENING_RESEARCH.md) — client RPC/payload contract, cost tables, class-ID root cause, cumulative Awakening rule, and transformed-skill ownership algorithm.
- [`docs/memory/pet-system.md`](../memory/pet-system.md) — authoritative Pet equipment persistence, SID, DTO, and equip callback ordering.
- [`docs/memory/stat-feature-foundation.md`](../memory/stat-feature-foundation.md) — Awakening progression and feature-state ownership.

## 1. Confirmed mechanics

### Pet equipment

- Pet equipment is `kind=9`, `useType=2`, and `position=50..57`.
- `reqClass` matches creature `classId`, not extended `classIds`; `reqClassId` matches the creature template ID.
- Both drag-and-drop and double-click call `petEquipOn(petId, itemInstanceId)`.
- Equipped Pet items persist at `slot_type=6`, SID `1001..1200`, and Pet `equ1..equ8`/`equipGroup`.
- Existing equip callback order remains unchanged.
- The missing modification RPCs and exact casing are:
  - `petEquLevelUp(equipmentID, materialItemID)` → `{flag,num,i,tid,sid,next,msg?}`.
  - `petEquModColor(equipmentID, materialItemID)` → `{flag,num,i,c,sid,msg?}`.
  - `petEquModBind(equipmentID, materialItemID)` → `{f,ii,n,b1,b11,b2,b12,mainProp1,mainProp2}`.
  - `surepetEquModBind(decision)` → `{f:"sure",b1,b2}` or `{f:"false"}`.
- Level cost is `next reqLevel² × 10` silver plus item `2333` using the checked level/multiplier table.
- Color cost is `current reqLevel² × 10` silver; material/rates are `2334 × 1/2/4 at 80/50/30%` and `2335 × 8 at 20%`, with checked special multipliers. A failed roll still consumes cost.
- Bind preview costs `30,000` silver plus item `2809` using the checked level/multiplier table; confirmation alone persists the rolled values.
- Mutation of an equipped Pet item recalculates/persists the owning Pet and emits `onRefreshPetProp({id,s})`.

### Awakening

- `awaken_level` remains authoritative in `player.character_progression`; allocation remains in feature key `awakening`.
- Awakening stats are cumulative across rows `1..awakenLevel`, matching `AwakenPanel.updateTotal()`.
- A successful `awakening` or `ensureBuyAwaken` emits one full `onUPP` stat payload; failure and `needGold` do not.
- The client sends `base.exSidN` when `awakenPointDict[base.codeName] == N`; it does not learn/persist that transformed ID.
- Exact learned-skill ownership remains valid. Fallback authorization requires an owned base row, matching allocated rank, exact `exSidN`, and an existing transformed template.
- Unauthorized or wrong-rank variants remain rejected. Authorized commands execute through the transformed game-data template.

## 2. Design

### 2.1 Single sources of truth

Keep Pet modification costs in one package-local table in the item handler, with helpers for the position `56..57` multiplier. Do not duplicate client tables across individual RPC methods.

Awakening authorization derives only from the persisted `pointDict`, learned base skill rows, and each base template's `ExSid1..3`. No transformed skill IDs are persisted as learned skills.

### 2.2 Pet equipment mutation flow

Add item-service mutation methods that load and persist the existing item instance while preserving its ID, SID, bind state, star level, and properties:

- replace `TemplateID` for level upgrade;
- set equipment display color through the canonical color-code conversion;
- persist bind values after confirmation.

The handler validates ownership, item/template category, exact material template and quantity, next-template compatibility, money, and terminal color/level before consuming anything. It then follows the existing consume-and-wallet callback conventions. Bind previews are stored per character under the existing handler mutex pattern and cleared on either decision.

Add a Pet-service method that locates the Pet owning an item ID in `equ1..equ8`, reruns equipment/skill aggregation and `RecalculateStats`, saves it, and returns the enriched Pet for the callback. Bag mutations return no Pet.

### 2.3 Awakening stat refresh

Change `applyAwakeningBonus` to iterate `1..awakenLevel`, skip missing rows safely, and apply both property pairs through `applyCharacterPropBonus`.

Inject the existing character repository and item-stat aggregator into the Awakening handler. On successful level-up, reload the character, call `AggregateEquipmentStats`, build `domain/character.BuildUPPPayloadWithEquipment`, and send `onUPP`.

### 2.4 Awakening skill authorization

Define a narrow authorizer interface in the skill application package. `skill.Service.HasSkill` first checks exact persistence; only when false does it load learned base IDs and ask the authorizer whether the requested ID is the allocated `exSidN` of one of those exact rows.

Implement the authorizer on `awaken.Service` using its existing feature-state loader and `gamedata.Manager`. Wire it into `skillService` after `awakenService` is constructed. The combat handler remains unchanged and continues to call `HasSkill`, preserving one ownership gate.

### 2.5 Touch points

| Touch point | Change |
|---|---|
| `internal/application/pet/skill_slots.go` | Resolve equipment class from `classId`/`ClassID`. |
| `internal/application/pet/equipment.go` | Add owning-Pet equipment refresh/recalculation method. |
| `internal/application/pet/equipment_test.go` | Add differing `classId`/`classIds` equip regression and refresh tests. |
| `internal/application/item/service.go` | Add narrowly scoped template/color mutation persistence helpers if existing helpers cannot preserve the required fields. |
| `internal/presentation/rtmp/handlers/item/pet_equipment_modify.go` | Implement the four exact RPCs, shared cost rules, pending bind state, payloads, inventory/money callbacks, and Pet refresh. |
| `internal/presentation/rtmp/handlers/item/handler.go` | Register RPCs and initialize pending state. |
| `internal/presentation/rtmp/handlers/item/pet_equipment_modify_test.go` | Cover validation, costs, success/failure, confirm/reject, exact payloads, and equipped-Pet refresh. |
| `internal/application/statfeature/character.go` | Aggregate all Awakening rows through the current level. |
| `internal/application/statfeature/character_test.go` | Prove cumulative mixed flat/percent channels and missing-row safety. |
| `internal/presentation/rtmp/handlers/awaken/handler.go` | Push one full stat refresh after successful level-up paths. |
| `internal/presentation/rtmp/handlers/awaken/handler_test.go` | Assert success-only callback and payload. |
| `internal/application/awaken/skill_variant.go` | Implement persisted-allocation/base-template `exSidN` authorization. |
| `internal/application/awaken/skill_variant_test.go` | Cover matching, wrong rank, unallocated, missing template, and foreign-base rejection. |
| `internal/application/skill/service.go` | Add exact-first variant-aware `HasSkill`. |
| `internal/application/skill/service_test.go` | Prove exact ownership compatibility and fallback behavior. |
| `internal/presentation/rtmp/handlers/combat/battle_test.go` | Prove an authorized transformed command survives command parsing and an unauthorized one is discarded. |
| `cmd/gameserver/main.go` | Wire Awakening stat refresh dependencies and the skill variant authorizer. |
| `docs/memory/pet-system.md` | Record modification RPCs, costs, and refresh contract. |
| `docs/memory/stat-feature-foundation.md` | Record cumulative stats and transformed-skill authority. |
| `docs/ai-memory/WORK_STATE.md`, `docs/ai-memory/FIX_HISTORY.md` | Record checkpoints and verification state. |

No schema or migration change is required. Implementation uses a narrow direct update on the existing `player.character_progression` row so immediate stat aggregation cannot observe stale write-behind state.

## 3. Non-goals

- Do not edit decompiled ActionScript.
- Do not implement `petEquModPre`/`surePetEquModPre`; the reported missing operations are level, color, and bind.
- Do not change normal character-equipment modification RPCs.
- Do not invent semantics for undocumented legacy skill `targetType=13` or recreate original-server script-only effects without evidence. Authorized variants still execute all fields supported by the existing generic adapter.
- Do not deploy, restart services, mutate catalog/player data, or test with real consumable items/Pets.
- Do not add the new RPC names to `quietMethods` before Flash confirmation.

## 4. Verification

1. Focused tests:
   - `go test ./internal/application/pet -run 'Test.*Pet.*Equip'`
   - `go test ./internal/presentation/rtmp/handlers/item -run 'TestPetEqu'`
   - `go test ./internal/application/statfeature -run 'Test.*Awaken'`
   - `go test ./internal/application/awaken ./internal/application/skill`
   - `go test ./internal/presentation/rtmp/handlers/awaken ./internal/presentation/rtmp/handlers/combat`
2. Broad checks:
   - `go test ./internal/application/pet ./internal/application/item ./internal/application/statfeature ./internal/application/awaken ./internal/application/skill ./internal/presentation/rtmp/handlers/item ./internal/presentation/rtmp/handlers/awaken ./internal/presentation/rtmp/handlers/combat`
   - `go vet ./...`
   - `go build -o bin/gameserver.exe ./cmd/gameserver`
   - `git diff --check`
3. User-driven Flash smoke test after an authorized restart:
   1. Equip compatible Pet gear by drag-and-drop, unequip it, then equip by double-click; verify immediate Pet stat changes.
   2. Try an incompatible `classId` and insufficient-level Pet; verify rejection without movement.
   3. Run level, color success/failure, bind preview reject, and bind preview accept; verify exact material/silver deltas and immediate equipped-Pet stat refresh.
   4. Perform one successful Awakening level increase; verify the displayed total and character properties update without relogin.
   5. Allocate one Awakening skill, enter combat, use the transformed skill, and verify it is accepted and uses the transformed template; remove allocation and verify rejection/fallback.
4. Update memory and `FIX_HISTORY.md`; keep all new RPC logging enabled until the user confirms the smoke test.
