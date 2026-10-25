# Pet Equipment and Awakening Repair Research

**Date**: 2026-10-24  
**Status**: Complete for implementation planning

## Scope

- Pet equipment equip/unequip through drag-and-drop and double-click.
- Pet-equipment upgrade, color/prefix change, and bind-property change RPC routing.
- Awakening stat application, immediate character refresh, and awakening-skill combat effects.

No ActionScript changes, live gameplay mutations, deployment, or service restart are in scope.

## Confirmed catalog facts

### Pet equipment

`docs/database/game_data/TBL_EQUIPT_TEMPLATE.json` contains 572 pet-equipment templates. Every row has `kind=9` and `useType=2`.

| Position | Type | Rows |
|---:|---:|---:|
| 50 | 900 | 18 |
| 51 | 901 | 18 |
| 52 | 902 | 18 |
| 53 | 903 | 18 |
| 54 | 904 | 18 |
| 55 | 905 | 18 |
| 56 | 906 | 232 |
| 57 | 907 | 232 |

The existing server/client contract recorded in `docs/memory/pet-system.md` assigns these items to persistent item `slot_type=6`, real client-visible SIDs `1001..1200`, and Pet property fields `equ1..equ8` plus `equipGroup`.

## Pet equipment findings

### Equip rejection

The Flash equip gate compares `TBL_EQUIPT_TEMPLATE.reqClass` with the Pet creature `classId`. `classIds` is a separate extended classification used by guardian/evolution features. The server helper `petEquipmentClassID` currently reads `CreatureData["classIds"]` and `TBL_CREATURE.class_ids`, so a Pet whose two classifications differ is rejected even when the client permits the equipment. The fix is to use `CreatureData["classId"]` and `TBL_CREATURE.class_id`; `reqClassId` continues to match the Pet creature template ID.

Drag-and-drop and double-click both reach the same client RPC, `petEquipOn(petId, itemInstanceId)`, so no second server route is required.

### Modification wire contract

All four missing methods are Responder-driven and use item instance IDs:

| RPC | Arguments | Result consumed by Flash |
|---|---|---|
| `petEquLevelUp` | `(equipmentID, materialItemID)` | `{flag, num, i, tid, sid, next, msg?}`; `num` is remaining material stack |
| `petEquModColor` | `(equipmentID, materialItemID)` | `{flag, num, i, c, sid, msg?}`; `num` is the consumed quantity and `c` is the new display color |
| `petEquModBind` | `(equipmentID, materialItemID)` | `{f, ii, n, b1, b11, b2, b12, mainProp1, mainProp2}` |
| `surepetEquModBind` | `(decision)` where `1=accept`, `-1=reject` | accept `{f:"sure", b1, b2}`, reject `{f:"false"}` |

The lowercase `p` in `surepetEquModBind` is intentional. The client also has separate `petEquModPre`/`surePetEquModPre` prefix RPCs, which are outside this reported three-operation scope.

Catalog/client costs:

- Level upgrade consumes item `2333` by current required level: `40:5`, `60:10`, `80:15`, `100:20`, `120:25`, `140:50`, `160:50`. Positions `56..57` multiply this by `2` when `reqClassId` is pipe-wrapped and otherwise by `1.5` rounded. Silver is `nextTemplate.reqLevel² × 10`. The visible production controls use the fixed required quantity; the old 1–5/20% controls are hidden.
- Color upgrade uses current display colors `0/1/2 → item 2334 × 1/2/4 at 80/50/30%`; color `3 → item 2335 × 8 at 20%`; color `4` is terminal. Positions `56..57` multiply quantity by `4` for pipe-wrapped `reqClassId`, otherwise by `2`. Silver is `currentTemplate.reqLevel² × 10`. Both success and failed rolls consume material and silver.
- Bind reroll consumes item `2809` by required level: `40:1`, `80:2`, `100:4`, `120:8`, `140:16`, `160:24`, with the same special-position multiplier `4` or `2`, plus `30,000` silver. The attempt produces a pending preview; only the confirmation RPC persists the selected bind values.

If the mutated item has `slot_type=6`, its owning Pet must be re-enriched, recalculated, persisted, and pushed through `onRefreshPetProp({id,s})`. Bag and pet-equipped items keep their existing SID and instance ID.

### Awakening

`supabase/seeds/data_tbl_awakening.sql` contains 77 progression rows. They carry actual stat channels in `prop1`, `prop2`, `prop_num1`, and `prop_num2`; the catalog is not display-only.

`supabase/seeds/data_tbl_awakening_skill.sql` contains 36 class-specific skill augmentations: six positions for each class `1..6`. Each row names one or more base skill code names whose combat behavior is modified.

`supabase/seeds/data_tbl_stat_feature.sql` marks `awakening` as an enabled character feature with `template_level` bonus strategy and `bonus_ready=1`. Its authoritative numeric level remains the character progression `awaken_level`; skill allocations belong to the `awakening` feature-state payload.

## Awakening findings

### Cumulative stats and immediate refresh

`AwakenPanel.updateTotal()` iterates every row from `1` through `player.awakenLevel` and sums both property columns by property ID. The server currently applies only `GetAwakening(awakenLevel)`, which contradicts that cumulative client model. `applyAwakeningBonus` must therefore fold all existing rows `1..awakenLevel` through the normal character property accumulator.

On a successful `awakening` or `ensureBuyAwaken` result, the client updates `awakenLevel` and `awakenPoint` but does not recalculate the character's derived property view. The handler needs to push one full equipment-plus-feature `onUPP` payload after persistence succeeds. Failed attempts and `{needGold}` responses do not emit a stat refresh.

### Skill transformation and ownership

`SkillManager` does not add a new learned skill. For each learned base skill it reads the base template's `codeName`, looks up `awakenPointDict[codeName]`, and replaces the displayed/sent skill ID with that base row's `exSidN`. Combat therefore receives the transformed ID even though persistence owns only the base ID.

The current combat command gate calls exact `HasSkill(characterID, requestedSkillID)`, so valid transformed commands are discarded before the game-data skill adapter can execute them. Authorization must match the client algorithm exactly:

1. exact ownership remains valid;
2. otherwise load the character's learned base rows;
3. for each base row, read allocation rank `N` from awakening `pointDict[base.codeName]`;
4. authorize only when `base.exSidN == requestedSkillID` and the requested template exists.

Wrong-rank variants, variants from another base row, and variants without allocation remain rejected. The checked catalog has 36 allocation rows covering 41 distinct base code names; all have `maxLevel=1`. Across the learnable level rows those code names produce 392 distinct `exSid1` templates, and every referenced variant exists. Their types are within the existing game-data adapter's type range; the repair routes execution through the transformed template rather than inventing a second effect table.

One legacy variant family (`Hộ Vệ`, IDs `10401..10410`) uses undocumented `targetType=13`, which is not named by `Battle.as` and is already outside the generic target map. This repair does not guess new target semantics; it fixes authorization and ensures all catalog-supported fields are read from the transformed row. Exact bespoke parity for undocumented legacy target/effect scripts requires separate original-server evidence.

## Server touch points confirmed

- Pet equip validation: `internal/application/pet/skill_slots.go`, with regression coverage in `equipment_test.go`.
- Pet equipment mutations: item application persistence helpers plus a dedicated item-handler file registered from `handlers/item/handler.go`; Pet refresh reuses `pet.Service` equipment aggregation.
- Awakening cumulative stats: `internal/application/statfeature/character.go`.
- Immediate stat callback: `internal/presentation/rtmp/handlers/awaken/handler.go` plus bootstrap dependencies.
- Transformed-skill ownership: `internal/application/skill/service.go` with a narrow awakening variant authorizer implemented by `internal/application/awaken` and wired after both services exist.
