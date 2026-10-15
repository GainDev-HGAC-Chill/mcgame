# Map Boss and Beast Handbook Rewards Plan

Status: **Implemented 2026-10-16 — pending Flash smoke-test/deploy**

Research: `docs/research/2026-10-16_01_MAP_BOSS_HANDBOOK_REWARD_AUDIT.md`

## Goal

Make every configured map boss and Beast Handbook boss start a real, correctly attributed battle and grant level-appropriate rewards matching the handbook descriptions.

## Implementation

### 1. Schedule-boss domain and scene visibility

- Change schedule eligibility to accept only active `ground` and `flying` rows; `daily_only` rows must never bootstrap or respawn.
- Add map/channel visibility checks:
  - normal map bosses follow persisted alive state;
  - mythic bosses are visible only on their configured channel scope (`channel:1`);
  - map ID must match the registry before engagement.
- Extend scene NPC assembly to inject registry bosses missing from the static map index, primarily the 13 mythic rows whose template map is 200.
- Deduplicate injected NPCs by NID and preserve template position/resource/type fields expected by `NPCView`.

### 2. Shared schedule-boss battle path

- Add a combat application method that:
  - validates schedule NID, map, channel, alive/scope state;
  - loads all `TBL_NPC_CREATURE` rows for the NID;
  - builds each creature at the configured encounter level with boss flags and appearance;
  - starts one PVE battle with the complete roster;
  - tags `battle.BossCtx` with the schedule NID, source `schedule`, and tier.
- Route both effective wires through this path:
  - `npc.ClickNpc` handles schedule bosses before generic NPC fallthrough;
  - `combat.HitNpc` recognizes schedule bosses before its legacy single-NPC path.
- Reuse the existing `onBattleStart`, Pet HP/MP, watcher, battle ID, and character-state callbacks.
- Keep non-boss NPC, quest battle, random encounter, and farm behavior unchanged.

### 3. Death and loot attribution

- Make the schedule kill hook prefer a `BossContext` whose source is `schedule`.
- Keep the enemy-entity fallback for legacy battles.
- Preserve de-duplication so one victory marks one schedule NID dead and rolls its loot once.

### 4. Reward seed generator

- Add `cmd/seed-boss-loot` to read the local static catalogs and emit deterministic SQL.
- Validate every selected item/equipment ID and every one of the 150 registry rows before emitting.
- Generate `supabase/seeds/data_tbl_schedule_boss_loot.sql`, which sorts after `data_tbl_schedule_boss.sql` so the latter's truncate cannot erase rewards.
- Emit bound drops on the 0–50000 scale with profile/level notes.

Proposed rates per victory:

| Reward | Ground normal | Flying normal | Mythic |
| --- | ---: | ---: | ---: |
| Level-scaled material | 35% | 35% | 45% |
| Gem | 20% | — | 25% |
| Feather | — | 20% | — |
| Rare progression item | 5% | 5% | 10% |
| Sub magic weapon/equipment | 3% from level 30 | 3% | 6% |
| Infinite Dust | 15% from level 55 | 15% | 25% |
| Magic Dust | 10% from level 55 | 10% | 20% |
| Wings of Light | — | — | 2% |

Quantities and material/gem/feather grades increase at level bands 1–29, 30–54, 55–89, 90–119, 120–149, and 150+.

Handbook profile selection:

- normal `num=2`: ground profile;
- normal `num=4`: flying profile;
- `isSuper=1`: mythic profile.

Map bosses use their registry `kind` and `tier`. This gives logical world/handbook equivalents the same reward family without depending on fragile name matching.

### 5. Tests

- Domain tests for `daily_only` exclusion and channel-scope matching.
- Scene tests for mythic injection, channel-1 filtering, normal deduplication, and dead normal-boss filtering.
- Combat service tests for complete roster construction, configured levels, explicit `BossContext`, invalid map/channel/dead rejection, and no generic fallback.
- Handler tests proving both `clickNpc` and `hitNpc` start a battle and send `onBattleStart`.
- Kill/loot tests proving one schedule NID is marked dead and rolled once even when enemy IDs are creature CIDs.
- Generator tests proving all 71 map and 79 handbook bosses receive at least one row and only valid catalog IDs are emitted.

### 6. Verification and documentation

- Run focused scheduleboss/combat/NPC/scene/activity/bossloot tests.
- Run targeted `go vet` for changed packages.
- Build with `go build -o bin/gameserver ./cmd/gameserver` and build the generator into `bin/`.
- Run `git diff --check`.
- Apply the generated seed to the local compose database only after a checkpoint, then verify counts and FK/item validity with read-only queries.
- Update `docs/memory/schedule-boss.md`, `docs/memory/boss-daily.md`, `docs/memory/boss-loot.md`, `docs/ai-memory/WORK_STATE.md`, and `docs/ai-memory/FIX_HISTORY.md`.

## Safety

- No ActionScript changes.
- No production/player-data changes.
- No deployment or restart.
- All ambiguous boss IDs, missing rosters, map mismatches, dead states, and wrong channels fail closed.
