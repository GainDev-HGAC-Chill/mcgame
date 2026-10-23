# Training, Shop Quality, Pet Soul, and KP Reward — Implementation Plan

**Date:** 2026-10-23 · **Status:** Approved for implementation on 2026-10-23  
**Goal:** Repair all four runtime paths end-to-end. Training uses a clearly labeled replacement curve that preserves every recovered original magnitude and fills only the missing values.

Research base:
- [2026-10-23_01_TRAINING_SHOP_SOUL_KP_RESEARCH.md](../research/2026-10-23_01_TRAINING_SHOP_SOUL_KP_RESEARCH.md) — verified Flash calls, catalog IDs, root causes, persistence ownership, and remaining Training gap.
- [Gameplay progression memory](../memory/gameplay-progression.md) — existing Training and Pet Soul contracts.
- [Item inventory memory](../memory/item-inventory.md) and [Dashboard memory](../memory/dashboard.md) — established quality and box-award behavior.

## 1. Confirmed mechanics

- Training consists of 12 families × 10 levels with `useEnv=3, kind=5`; learned rows affect character and Pet, but 119/120 linked buffs contain no numeric magnitude. An official 4399 table now supplies 30 basic-tier base rows, while coefficients and the other tiers remain absent.
- NPC shops `32/82` sell gathering gloves `1123..1126` with `quality=0`, meaning explicit white/no prefix.
- Pet Soul EXP, level advancement, dissolve thresholds, and temporary-bag RPC shapes are defined by the Flash call sites and `TBL_PET_SOUL`.
- Pet Soul bonuses are the exact `propType/propVal` of each equipped `sid`.
- Văn Chỉ is PPVE `state.kp`, not character currency `238` (`pvePoint`).

## 2. Authority table

| Decision | Source | Status |
|---|---|---|
| Shop quality `0` stays white/no prefix | `TBL_SHOP_SLOT` rows `3439..3446`; item display contract | Confirmed |
| Soul level advances to sequential `sid+1` | `TipSoul.as` next-row lookup; level-up eligibility | Confirmed |
| Soul dissolve returns template base EXP plus invested EXP | `TipSoul.as` total EXP display | Confirmed |
| Quick dissolve filters by selected color threshold | `PetSoulPanel.as` and `PetSoulProductPanel.as` | Confirmed |
| Soul stat magnitude | `TBL_PET_SOUL.propType/propVal` | Confirmed |
| `永生` localization | property 34 Vietnamese label + catalog context | Confirmed: `Vĩnh Sinh`, `Miễn Tử` |
| Văn Chỉ persistence | `PPVEState.KP` and panel payload | Confirmed |
| Basic Speed/Physical Defense/Magic Defense base curves | Official 4399 `gzfgj/rwxl.html`, level requirements match catalog | Confirmed for 30/120 rows |
| Basic HP levels 1..8 | Player table matching buff `1270`; Lezi confirms base-HP semantics | Corroborated only |
| Training coefficients, remaining HP, intermediate/advanced curves | No complete source found after client/archive/web/626-Q&A audit | Blocked |

## 3. Unit A — Training stat application

### Approved design direction

The user authorized a replacement balance curve on 2026-10-23. It is not represented as recovered original behavior.

### Replacement curve

- Basic character base values by level 1..10:
  - HP: `100, 220, 360, 520, 700, 900, 1120, 1360, 1620, 1900`. Levels 1..8 preserve the corroborated table; levels 9..10 continue its exact `+20` increment-growth pattern.
  - Speed: `20, 40, 60, 120, 180, 240, 320, 480, 700, 950` from the official 4399 table.
  - Physical and Magic Defense: `400, 800, 1200, 1600, 2200, 3000, 4000, 5200, 6800, 8800` from the official 4399 table.
- Basic Pet base values by level 1..10:
  - HP: `50, 110, 180, 260, 350, 450, 560, 680, 810, 950`; levels 1..8 preserve the corroborated table and levels 9..10 remain exactly half the replacement character values.
  - Speed: `10, 20, 30, 60, 90, 120, 160, 240, 350, 480` from the official 4399 table.
  - Physical and Magic Defense: `200, 400, 600, 800, 1100, 1500, 2000, 2600, 3400, 4400` from the official 4399 table.
- Tier multipliers are Basic `1.00`, Intermediate `1.50`, Advanced `2.25`. Each of the 12 catalog families remains independent; only the highest learned level inside a family applies, while learned tiers of the same stat stack additively.
- Character coefficients use canonical `TBL_CLASS` aptitudes normalized around `2000`, clamped to `[0.75, 1.25]`:
  - HP: `aptStamina / 2000`.
  - Speed: `aptAgility / 2000`.
  - Physical Defense: `(2.57*aptStamina + 0.85*aptAgility) / (3.42*2000)`.
  - Magic Defense: `(2.71*aptIntelligence + 4.60*aptEnergy) / (7.31*2000)`.
  - These weights deliberately reuse `internal/domain/stats/profile.go`; the resulting coefficients for classes 1..6 are HP `1.25/0.80/0.95/0.85/1.00/1.10`, Physical Defense `1.20/0.875/0.90/0.85/1.00/1.10`, Magic Defense `0.781/1.194/1.20/1.224/1.00/0.781`, and Speed `1.05/1.10/0.75/0.85/1.00/1.10`.
- Pet coefficients are a frozen family table derived from the median aptitudes of the 195 catchable `TBL_CREATURE` rows, using the same normalized formulas and clamp. For Pet `classId` 1..6, the HP/Physical Defense/Magic Defense/Speed coefficients are:
  - Human: `1.040/1.030/1.088/1.000`.
  - Beast: `1.026/1.074/0.750/1.220`.
  - Plant: `1.250/1.136/1.185/0.750`.
  - Machine: `1.250/1.250/1.210/1.150`.
  - Devil: `1.200/1.188/1.250/1.150`.
  - Dragon: `0.750/0.812/1.136/1.000`.
- Each family contribution is `round(base[level] * tierMultiplier * coefficient)` and all applicable family contributions are summed as flat bonuses. Missing/invalid class, Pet family, skill, level, or catalog fingerprint contributes zero and is logged; it never falls back to a guessed value.
- The official table's defense-immunity annotations are intentionally excluded because the Vietnamese training catalog has no property authority for them and this replacement is limited to the four stated stats.

### Implementation

- Add a Training bonus provider over learned character skills, keyed only by the exact 120-row catalog fingerprint.
- Aggregate only the highest learned level per family.
- Apply character and owned-Pet bonuses in the existing stat recalculation paths, before DTO and battle snapshots.
- Register Training as an additional character bonus provider and add composable Pet bonus providers rather than replacing the existing stat-feature provider.
- Add table-driven tests for family/level/character class/Pet family, upgrades replacing rather than stacking old levels, relog persistence, and battle-sheet parity.

## 4. Unit B — Explicit shop quality

- Add an item-service grant API whose color argument is explicit, including zero; leave current template-fallback behavior unchanged for other callers.
- Route NPC and System Shop equipment purchases through the explicit API.
- Validate shop quality as `0..25`; map through the shared quality contract rather than ad-hoc `math.Ceil`.
- Replace raw quality fields in NPC and System Shop editors with labeled quality controls and explanatory text. Preserve existing optimistic revision, audit, and special-currency guards.
- Regression: gathering gloves from slots `3439..3446` produce `colorCode=0`, display color white, and no `preNameType`/`Trác Việt`; positive quality still maps correctly.

## 5. Unit C — Pet Soul correctness

- Introduce a shared soul mutation lock per character/Pet ownership boundary to prevent lost updates during EXP/dissolve/move operations.
- Parse `putInToExp` as the Flash fraction and calculate a bounded absolute amount from the current remaining requirement.
- On `soulLevelUp`, spend only `upExp-slot.exp`, advance to the validated same-family `sid+1`, and reset invested EXP. Reject level 10 or catalog drift without spending.
- Dissolve single bag souls for `template.exp + slot.exp` and the authoritative chip return.
- Dissolve quick bag souls only when unlocked and `template.color < threshold` for threshold `1..3`.
- Register `transformExpInPanel`; implement single and threshold-filtered bulk temporary-bag dissolve using template base EXP/chip values.
- Preserve existing callback names for current bag operations; add focused handler tests around the temporary-panel callbacks and payloads. Keep unknown callback details isolated behind handler helpers.
- Aggregate equipped Pet Soul rows into Pet recalculation before DTO/battle creation. Apply exact integral/decimal `propVal` through the existing Pet bonus/stat property model; unsupported combat properties remain visible in canonical Pet properties and are added to `EffectiveStats` only where a domain mapping exists.
- Normalize rows `751..760` to `Vĩnh Sinh` / `Tăng Miễn Tử ...` in canonical game-data and a narrow idempotent data migration; do not modify ActionScript.
- Regression: partial EXP, full level-up, level-10 refusal, single/quick bag dissolve, single/quick temp dissolve, locked/color exclusion, stat sheet/battle parity, ownership rejection, and translation rows.

## 6. Unit D — Box reward “Văn Chỉ”

- Reserve a non-conflicting box game-currency award ID named `Văn Chỉ (KP)`; do not register it as a character scalar currency.
- Add an item reward-granter interface implemented by `PPVEService`. It loads PPVE state, increments `KP`, persists the full state, and returns `KPData`/`PPVEData`.
- Wire the granter in `cmd/gameserver/main.go` and fail closed when unavailable.
- Extend item reward results and RTMP callbacks to send `onUpdatePPVEPanel` after a successful KP grant; never map it to `pvePoint` or generic `onAddMoney`.
- Add the option to `BOX_ITEM_GAME_CURRENCY_OPTIONS`, API validation, display labels, and tests. Existing selectable-bag constraints remain unchanged.

## 7. Verification

- Focused Go tests: `internal/application/soul`, `internal/presentation/rtmp/handlers/soul`, `internal/application/pet`, `internal/domain/pet`, `internal/application/item`, and item handlers.
- Focused Shop tests plus Dashboard Node tests and `pnpm typecheck`.
- Targeted `go vet` for changed packages.
- Windows and Linux gameserver builds into `bin/`.
- `docker compose -f docker/docker-compose.yml build mcgame-server dashboard`.
- `git diff --check`.
- No automatic restart, deploy, live item use, player/catalog mutation, commit, or push.

## 8. Rollback and residual risks

- Shop explicit-color path is isolated and can be reverted without changing item persistence.
- Soul state remains compatible (`sid`, `exp`, `lock`); corrected semantics do not require destructive state migration.
- Translation migration is narrow to IDs `751..760` and must have an inverse statement documented before application.
- Mixed box rewards are not globally transactional in the current architecture. KP grant ordering must avoid consuming the box when PPVE persistence fails, and tests must cover this failure path.
- Training curve constants are replacement balance data. Keep them isolated and table-driven so recovered original values can replace them without changing persistence or RPC behavior.
