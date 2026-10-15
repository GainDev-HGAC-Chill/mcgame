# Map Boss and Beast Handbook Audit

## Scope

- World-map ground and flying bosses in `data.data_tbl_schedule_boss`.
- Beast Handbook bosses driven by `BossDailyPanel` and `bossDailyBattle`.
- Battle creation, schedule lifecycle, boss attribution, and reward coverage.
- Read-only inspection of the local compose database and decompiled Flash client.

## Flash contract

- Map NPCs with `npcType == 15` send `clickBoss(id)`.
- Map battle NPCs with `npcType == 10` send `clickNpc(id)` when clicked and `hitNpc(id)` after approaching within 50 pixels.
- All current map-boss registry NPCs use type 10, so their primary click wire is `clickNpc`, not `clickBoss`.
- Beast Handbook cards send `bossDailyBattle(bid)` and expect `onBattleStart` asynchronously.
- The handbook contains 79 boss IDs. Its reward descriptions use these categories:
  - ground/normal: materials, gems, rare items; from level 30, sub magic weapons; from level 55, Infinite Dust and Magic Dust;
  - flying/normal: materials, feathers, rare items, sub magic weapons, and the two dusts;
  - mythic: materials, rare items, optional gems/sub magic weapons, Wings of Light, and the two dusts.

Authoritative client references:

- `docs/client/view/view/compGameStage/NPCView.as`
- `docs/client/view/view/comp/BossDailyRect.as`
- `docs/client/view/view/compDragable/BossDailyPanel.as`
- `docs/client/predef/GamePredef.as`

## Database inventory

Local compose inspection on 2026-10-16:

| Area | Count | Validation |
| --- | ---: | --- |
| Map bosses | 71 | All have NPC rows and valid `TBL_NPC_CREATURE` rosters of at most 10 entries |
| Normal map bosses | 58 | NPC map matches registry map; 35 ground + 23 flying |
| Mythic map bosses | 13 | Registry map is valid, but NPC `pos_map_id` is the sentinel map 200 |
| Handbook bosses | 79 | All have NPC rows and exactly five valid encounter rows |
| Schedule runtime rows | 360 | All alive at inspection time |
| Boss loot rows | 0 | No map or handbook boss can receive rewards from the unified boss-loot path |

The 360 runtime rows are not 58 map bosses across channels. They also include 62 normal-tier `daily_only` rows because `BossConfig.Eligible()` checks only tier and active state, not kind. Daily-only rows therefore pollute schedule state even though they have map 0.

## Map-boss battle findings

### RPC overwrite blocks direct click

`combat.Handler.RegisterHandlers` registers `clickNpc` to `HitNpc`, but `npc.Handler.RegisterHandlers` runs later and registers the same method to `ClickNpc`. `RPCDispatcher.Register` replaces existing handlers. The effective `clickNpc` handler is therefore the NPC interaction handler.

For a type-10 map boss within click distance, Flash calls `clickNpc`. The NPC handler has no schedule-boss branch, so the click falls through without starting combat. This matches the reported symptom.

`hitNpc` still points to the combat handler, but Flash only sends it through the proximity/approach path. It is not a reliable replacement for direct click.

### Current encounter construction is incorrect

`BuildNPCTemplateForCharacter` looks for a map-creature row whose `cid == npcID`. Schedule boss IDs are NPC IDs and their encounter creatures are defined by `TBL_NPC_CREATURE.nid`. The lookup normally misses and falls back to one generic level-10 enemy.

The intended rosters are present and valid:

- ground bosses generally use repeated boss rows;
- flying bosses use a boss plus supporting creatures;
- roster sizes fit the ten enemy battle slots.

### Boss attribution is lost

When a real creature template is built, `CalculateCreatureStats` stores the creature CID in `NPCTemplateData.ID`. Existing schedule death and loot detection compare enemy `EntityID` with schedule NPC NIDs. A correctly built roster would therefore still fail to mark the schedule boss dead or resolve boss loot unless the battle carries an explicit schedule `BossContext`.

### Thirteen channel-1 bosses are absent

The 13 mythic ground bosses from `activity.xml` target real maps, but their NPC templates have `pos_map_id=200`. Scene creation reads NPCs by `pos_map_id`, while schedule visibility only filters existing scene NPCs. These bosses are never injected into their registry maps.

Their template coordinates and resources are populated and can be used for registry-driven scene injection. Their `channel_scope='channel:1'` must be enforced server-side.

## Beast Handbook findings

- All 79 client config IDs are accepted by the server domain registry.
- All 79 resolve to valid five-enemy rosters; no missing creature templates or non-positive encounter levels were found.
- Runtime logs prove successful battle creation for IDs 2204, 2205, 2206, and 2262.
- Daily battle attribution resolves because every daily row has `daily_boss_id`, although the current seed keeps all 79 as separate `daily_only` rows instead of pairing logical world equivalents.
- The unified reward table is empty, so victory grants no handbook-specific reward.
- The server increments the handbook attempt state when battle creation succeeds, before victory. This is existing behavior and is not required for the reported click/reward repair.

## Reward source assessment

`TBL_PLAN` has 591 legacy plan rows across 55 of the 71 map bosses, but it is not safe to import those rows mechanically:

- only 55 map bosses are covered;
- many entries are seasonal or unrelated to the current handbook description;
- `r=1` often appears on many rows in the same plan and does not safely translate to independent guaranteed drops;
- handbook-only creatures generally have no matching plan rows.

A curated deterministic profile is safer and satisfies the visible client text. Rewards should be bound, use the existing 0–50000 rate scale, and increase by boss level/profile:

| Profile | Required categories |
| --- | --- |
| Ground normal | level-scaled materials and gems; rare item; sub magic weapon from level 30; both dusts from level 55 |
| Flying normal | level-scaled materials and feathers; rare item; sub magic weapon; both dusts |
| Mythic | stronger material/rare pool; optional gem/sub weapon; Wings of Light; both dusts |

For handbook normal bosses, `num=2` maps to the ground profile and `num=4` maps to the flying profile. Mythic handbook entries use the mythic profile. This directly follows the three distinct tooltip families in `BOSS_DAILY_CONFIG`.

## Required correction surface

1. Make schedule eligibility exclude `daily_only` rows and add map/channel engagement validation.
2. Inject registry-backed mythic bosses into their configured maps on channel 1.
3. Route both `clickNpc` and `hitNpc` schedule-boss interactions into one roster-aware battle start path.
4. Build enemies from `TBL_NPC_CREATURE`, preserve the schedule NID through `BossContext`, and use it for death and loot attribution.
5. Generate a checked-in boss-loot seed after the schedule-boss seed, covering all 71 map bosses and all 79 handbook bosses.
6. Add full coverage tests so a future catalog change fails if any configured boss lacks a valid roster or reward profile.

## Out of scope

- No ActionScript edits.
- No deployment, container restart, or real-player smoke test.
- No reward claim by quick-battle card beyond the existing stub behavior.
- No change to handbook attempt timing unless separately approved.
