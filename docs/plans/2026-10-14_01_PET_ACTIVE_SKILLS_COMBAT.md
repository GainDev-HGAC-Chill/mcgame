# Pet Active Skills Combat — Implementation Plan

**Date**: 2026-10-14 · **Status**: Approved
**Goal**: Make all 144 audited templates in 52 active Pet skill families select the described targets, calculate damage/healing from Pet combat stats, and execute their described buffs, debuffs, controls, periodic effects, costs, and special mechanics.

Research base (verified 2026-10-14):
- `docs/research/2026-10-14_02_PET_ACTIVE_SKILL_AUDIT.md` — complete family audit, current defects, target shapes, and user-approved conflict resolutions.
- `docs/memory/battle-combat.md` — battle command, target, playback, damage, and skill adapter contracts.
- `docs/database/details/TBL_SKILL.md` and `TBL_BUFF.md` — seed field meanings and client-visible descriptions.

## 1. Confirmed mechanics

- Scope is the 144 `TBL_SKILL.kind=1` templates whose names match Pet books (`TBL_ITEM_TEMPLATE.type=506,useType=2`), grouped into 52 families.
- Physical and magic `X% damage` means `X% × Pet Attack/MagicAttack` before defense, critical, element, variance, and existing damage modifiers.
- `attackNum` is the number of full per-hit applications. Multi-hit stops after no legal enemy remains.
- Pure support/state/remove skills deal no direct damage unless their rule explicitly includes damage.
- Proc chance is rolled independently for each target.
- Visible skill descriptions override conflicting `TBL_SKILL`/`TBL_BUFF` values.
- Existing RPC, `battleUpdateCmd`, and `onBattlePlayList` wire shapes do not change.
- No migration or persistent state is required. Every active effect is battle-local and disappears with the battle.
- Preserve the current uncommitted external normal-attack target-resolution change in `internal/domain/combat/turn_executor.go`; this feature only changes the skill branch and shared effect handling.

### 1.1 Target rules

- Existing target geometry remains authoritative for normal single, vertical, horizontal, cross, and all-side shapes.
- `Đoạt Mệnh`, `Liêu Hồn`: one selected living enemy, overriding catalog `targetType=11`.
- `Nuốt Chửng`: one selected participant. Ally means consume one removable debuff; enemy means consume one removable beneficial buff.
- `Thanh Phong Phục Sinh`: one selected dead ally.
- `Tự Nhiên Thuẫn Đơn Thể` level 3: allied horizontal row, following visible description.
- Taunt forces the affected target's offensive single-target action toward the caster while the caster remains legal.

### 1.2 Direct damage and healing registry

The catalog is keyed by stable family name and template level. Values are percentages of Pet Attack (`P`), MagicAttack (`M`), or healing from MagicAttack (`H`).

| Family | Levels | Rule |
|---|---|---|
| Ác Ma Chi Kích | 1/2/3 | E1, M 100/110/120% |
| Cự Mộc Áp Thân | 1/2/3 | E1, M 80/90/100% |
| Cự Mộc Áp Thân Cao Cấp | 1/2/3 | E1, M 115/135/155% |
| Dã Thú Cuồng Vũ | 1/2/3 | E1, P 100%, 2/2/3 hits |
| Diệp Vũ Chi Thuật Cao Cấp | 1/2/3 | enemy horizontal 3, M 55/65/75% |
| Diệt Vong Chi Ca | 1 | E1, M 80% |
| Lời Ngọt Ngào | 1/2/3 | all enemies max 10, M 100/110/120% |
| Phong Hôn Chi Nhẫn | 1/2/3 | enemy horizontal 3/3/4, M 60/75/60% |
| Quyến Luyến Không Rời | 1/2/3 | enemy horizontal 3, M 36/48/60% |
| Tấn Công Cấp Tốc | 1/2/3 | E1, P 100% |
| Thích Nhĩ Hô Khiếu | 2/3 | E1, P 120/130% |
| Toái Băng Phong Bạo | 1/2/3 | E1, M 80/90/100% |
| Bát Diệp Trảm | 1/2/3 | E1, P 80/90/100%; 10/15/20%, 2 rounds, physical Attack −15/−24/−25% |
| Dập Hỏa Thuật | 1/2/3 | enemy vertical 2, M 60/70/90%; 10/15/25%, 2 rounds, both Defense −5/−10/−15% |
| Độc Huyết Trảo | 2/3 | E1, P 75/105%; 34/40%, 2/3 rounds, physical Attack −24/−33% |
| Hắc Dạ Ma Ảnh | 1/2/3 | E1, M 55/75/105%; 30/34/40%, 2/2/3 rounds, MagicAttack −10/−16/−22% |
| Kí Sinh Chủng Tử Cao Cấp | 1/2/3 | E1, M 115/135/155%; 25/40/55%, 2 rounds, MagicAttack −10% |
| Ký Sinh Chủng Tử | 1/2 | E1, M 80/90%; 10/15%, 2 rounds, MagicAttack −10/−16% |
| Long Khiếu | 1/2/3 | all enemies max 10, M 20/22/26%; 10/15/25%, 1 round confusion |
| Pháo Kích | 1/2/3 | enemy cross 4, P 100%; 60/70/80%, 1 round hard stun |
| Sương Đông Đơn Thể | 1/2/3 | E1, M 100/110/120%; 97%, 3 rounds Speed −10/−20/−30% |
| Sương Đông Quần Thể | 1/2/3 | enemy cross 4, M 70/75/80%; independent 97%, 3 rounds Speed −10/−20/−30% |
| Suy Tiến Trang Trí | 1/2/3 | enemy vertical 2, P 30/40/60%; 30/34/40%, 1/1/2 rounds stun |
| Tàn Nhẫn Tư Liệt | 1/2/3 | E1, P 55/75/105%; 20/25/30%, 2/2/3 rounds bleed tick = 60/90/140% caster Energy |
| Thủy Cam Lộ | 1/2/3 | A1, H 110/130/160%; 2 rounds fixed HoT 75/145/238 HP |

### 1.3 Special damage, costs, recovery, and state rules

| Family | Confirmed rule |
|---|---|
| Đoạt Mệnh | E1; P 130/160/190%; independent 10/20/30% proc adds fixed 5/10/15% of target current HP sampled before the hit |
| Liêu Hồn | E1; M 120/140/160%; same proc/current-HP addition as Đoạt Mệnh |
| Hủy Diệt | Requires caster HP >90%; enemy horizontal 3; fixed 35/45/55% target max HP; caster is sacrificed after resolution |
| Phóng Thích Năng Lượng | E1; fixed 30/45/60% target max HP; caster cannot act on its next turn |
| Táng Thần Đơn Thể | E1; P 140/160/190%; caster skips next turn |
| Độc Thích Đơn Thể | No immediate hit; 88%, 3 rounds; magic DoT 30/60/90% MagicAttack each tick |
| Độc Thích Quần Thể | enemy horizontal 4/5 or all enemies 10; independent 77%, 3 rounds; magic DoT 30/40/50% MagicAttack each tick |
| Ma Pháp Thuẫn | A1; costs 60% caster max MP; four rounds; magic-only shield for at most 3 hits with pool 50/75/100% caster max MP |
| Nuốt Chửng | One selected participant; randomly consume one eligible effect; on success restore caster 10/15/20% max HP and max MP once |
| Quang Hợp | Owner; spend 20/30/40% max MP and gain the same amount as HP; 5 rounds restore 2/3.6/5.6% max MP per round |
| Thanh Phong Phục Sinh | One selected dead ally; level 2/3 revives at 20/30% target max HP |
| Thiên Hựu | One allied character; 2 rounds invulnerable to damage and unable to act |
| Thuốc Kích Thích | Owner; costs 30/40/50% max HP and requires current HP greater than cost; 3 rounds Speed +30/40/50%, Combo and Critical +14/21/30% |

### 1.4 Pure buff, debuff, and control registry

| Family | Confirmed rule |
|---|---|
| Ảo Hoa Kỳ Hương | E1; 30/40%, 2/3 rounds sleep; blocks actions/items and breaks on damage |
| Đế Vương Nộ | Owner; requires caster HP >40%; 2 rounds physical/magic Attack +50/75/100% |
| Duy Trì Tốc Độ | A1; 3 rounds Dodge +10/20/30% |
| Hấp Thu Năng Lượng | E1 non-boss; 2/2/3 rounds; randomly choose 1/2/3 distinct stats among both Attack and both Defense; transfer 10/15/20% from target to caster |
| Hiệp Ước Tử Thần | Owner; level 1; 3 rounds both Attack and both Defense +50% |
| Khóa Mục Tiêu | enemy vertical 2; independent 85%; 3 rounds incoming damage +15/25/40% |
| Suy Yếu Linh Hồn | E1; 3 rounds physical/magic damage reduction −10/20/30 percentage points |
| Thú Tính Gào Thét | all enemies max 10; 5 rounds physical Defense −10/20/30% |
| Tích Lực Sơ Cấp | Owner; 1 round Critical +48/67/88% |
| Tích Lực Trung Cấp | Owner; 2 rounds Critical +31/48/67% |
| Trào Phúng Đơn Thể | E1; 78%; 1/3/4 rounds taunt |
| Tự Nhiên Thuẫn Đơn Thể | A1/A1/allied horizontal 5; 4 rounds physical Defense +1000/+2000/+3000 and HoT = 65% caster MagicAttack per round |
| Tự Nhiên Thuẫn Quần Thể | allied horizontal 4/5 or all allies 10; 4 rounds physical Defense +1000 and HoT = 25% caster MagicAttack per round |
| Vạn Thú Tề Phát | A1; levels 2/3 for 2/3 rounds: physical Attack and Speed +14/20%, both Defense −14/20% |

## 2. Design

### 2.1 Single source of truth

- Add a static, read-only Pet active skill catalog in `internal/application/combat/pet_active_skill_catalog.go`, keyed by exact family name and template level.
- The catalog compiles into domain-owned `SkillPlan` values. It is the only place containing per-family coefficients and special rule selection.
- `GamedataSkillAdapter` returns a plan only for catalog matches. Non-Pet and unknown skills retain the existing legacy adapter behavior.
- Catalog validation tests load checked-in `TBL_SKILL`/`TBL_ITEM_TEMPLATE` fixtures and assert all 144 audited templates resolve to a plan.

### 2.2 Reusable battle-local effects

- Extend `SkillInfo` with an optional `Plan` containing target override, costs, and ordered effects.
- Add domain effect primitives for physical/magic/fixed damage, heal, resource restore/cost, stat modifier, periodic damage/heal, shield, control, taunt, revive, sacrifice, and remove-one-effect.
- Extend battle-local status instances with source/template ID, polarity, removability, stat modifiers, periodic effects, shield pool/hit count, and control flags.
- Do not mutate base Participant stats. Damage, order, accuracy, and target selection read effective stats overlaid with active effects.
- Same-family reapplication replaces/refreshes the existing instance; unrelated effects coexist. Effects expire after the configured number of affected participant turns.
- A dispel removes one random eligible effect using `TurnProcessor.rng`.

### 2.3 Execution order

1. Resolve target override and alive/dead eligibility.
2. Validate all HP/MP requirements without mutation.
3. Pay costs.
4. For each target, roll that target's proc independently.
5. Execute ordered direct effects, then attach timed effects.
6. Apply post-use sacrifice/rest.
7. Build existing-compatible playback actions and recheck death/victory through current turn flow.

Damage application must pass through one shared path so invulnerability and shields also protect against normal attacks, counters, direct skills, and periodic damage. Fixed-percent HP damage bypasses defense/crit/element/variance. Healing caps at MaxHP/MaxMP.

### 2.4 Touch points

| Touch point | Change |
|---|---|
| `internal/application/combat/pet_active_skill_catalog.go` | Add the complete family+level registry and compile it into domain plans. |
| `internal/application/combat/skill_adapter.go` | Use the Pet registry before legacy `expSkill/dexSkill` heuristics; retain fallback for other skills. |
| `internal/domain/combat/turn_processor.go` | Extend `SkillInfo`, expose deterministic RNG-based plan execution, and unify turn-start effect processing. |
| `internal/domain/combat/skill_plan.go` | Add battle-domain plan/effect/cost/target types. |
| `internal/domain/combat/skill_executor.go` | Execute plans and produce existing playback actions. |
| `internal/domain/combat/status_effect.go` | Generalize battle-local timed effects, modifiers, shields, controls, polarity, and dispel. |
| `internal/domain/combat/target_resolver.go` | Support target overrides, dead ally selection, and taunt-forced targets. |
| `internal/domain/combat/damage.go` | Apply effective stat overlays and shared shield/invulnerability damage. |
| `internal/domain/combat/resolve.go` | Apply the same plans/modifiers in the optional per-mille pipeline. |
| `internal/domain/combat/turn_builder.go` and `turn_builder_permille.go` | Route HP mutation through shared damage application without changing wire shape. |
| Focused `_test.go` files in application/combat and domain/combat | Table coverage and behavior tests described below. |

## 3. Non-goals

- No Flash/ActionScript changes.
- No schema, migration, or persistent out-of-battle buff state.
- No changes to Pet skill learning/upgrading or inventory consumption.
- No redesign of non-Pet character/NPC skill formulas; they remain on the legacy adapter unless they use shared defensive effects.
- No inferred extra effect for the corrupted tail of `Pháo Kích`; implement only P 100% plus confirmed stun.
- No container restart, production data mutation, or live asset consumption.

## 4. Verification

1. Application catalog table test: all 144 audited templates/52 families produce a plan; unknown/non-Pet skill does not.
2. Damage tests: physical/magic multiplier before mitigation; fixed current/max HP; multi-hit; optional per-mille mode.
3. Target tests: type-11 overrides, all AoE geometries, dead ally revive, non-boss, taunt.
4. Effect tests: independent per-target proc; stat overlays; duration/refresh; sleep break; confusion; DoT/HoT; shield hit/pool limits; invulnerability; revive; rest; dispel; sacrifice; HP/MP costs; transfer.
5. Regression: pure support skills cause zero damage; normal attack external target fix remains intact; existing combat tests pass.
6. Commands:
   - `go test ./internal/application/combat ./internal/domain/combat`
   - `go vet ./internal/application/combat ./internal/domain/combat`
   - `go build -o bin/gameserver ./cmd/gameserver`
   - Compare broader `go test ./internal/...` failures against documented baseline; do not fix unrelated failures.
7. Manual Flash smoke test after separately approved deploy: one single-target damage skill, one AoE, one multi-hit, one buff, one DoT, one heal, revive, shield, Nuốt Chửng, and one special HP-percent skill; verify target count, HP deltas, durations, and no disconnect.
8. Update `docs/memory/battle-combat.md`, `docs/ai-memory/WORK_STATE.md`, and `FIX_HISTORY.md` after implementation and verification.

## 5. Implementation notes

- Implemented the registry for all 144 checked-in templates in 52 families. The fixture test derives the roster from checked-in `TBL_ITEM_TEMPLATE` and `TBL_SKILL` JSON rather than duplicating template IDs.
- Source-reality correction: the optional per-mille path does not need a separate plan implementation in `resolve.go`; the plan executor calls the existing `ResolveHits` pipeline with the planned multiplier and effective battle-local stats.
- Source-reality correction: generalized runtime status helpers were split into `status_effect_runtime.go` to keep `status_effect.go` within the repository file-size limit.
- Source-reality correction: the domain has no separate Energy field, so `Tàn Nhẫn Tư Liệt` snapshots `Participant.CurrentSP`, the battle participant's energy resource, for its periodic coefficient.
- Focused application/domain tests and vet passed before concurrent unrelated item/combat-service edits made the full application package uncompilable. The final domain and isolated catalog/adapter reruns pass with vet; the gameserver build is blocked by unrelated `rewards.go` and `handlers/item/use.go` errors.
