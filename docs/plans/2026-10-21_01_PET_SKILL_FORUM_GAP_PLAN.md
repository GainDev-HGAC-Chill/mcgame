# Pet Skill Forum Gap — Implementation Plan

**Date**: 2026-10-21 · **Status**: Implemented (CODE_FIXED)  
**Goal**: Extend the server's audited Pet skill coverage with every KingGame-listed mechanic that is corroborated by checked-in seed data, while preserving the Flash playback contract and failing closed on incomplete or conflicting community descriptions.

Research base (verified 2026-10-21):

- `docs/research/2026-10-21_01_PET_SKILL_FORUM_GAP_RESEARCH.md` — forum-to-seed-to-runtime gap matrix and exact IDs.
- `docs/research/2026-10-14_02_PET_ACTIVE_SKILL_AUDIT.md` — approved 144-template/52-family baseline and conflict policy.
- `docs/memory/battle-combat.md` — current skill-plan, status, targeting, playback, and passive contracts.
- Decompiled Flash paths cited in the research addendum — one-selected-target command and server-owned expansion/playback.

## 1. Confirmed mechanics

- Flash sends one `tid`; the server remains authoritative for target expansion, repeated hits, independent per-target proc rolls, status lifecycle, and passive hooks.
- Exact seed descriptions and fields take precedence over incomplete or uncertain forum values.
- Forum values may fill a seed omission only when the level sequence is coherent and does not contradict another authoritative runtime decision.
- Existing approved plans remain unchanged except for the Tự Nhiên Thuẫn Đơn Thể HoT defect identified by this audit.
- Unknown/conflicting families remain unsupported rather than receiving guessed legacy damage.

### 1.1 Exact active plans

| IDs | Family | Plan |
|---|---|---|
| 2961–2963 | Cao Tần Long Hống Trung Cấp | E1; M95/110/125%; 15/25/35% MagicAttack −10/16/22%, 2 turns. |
| 2666, 2735, 2736 | Đột Thích Xung Phong | E1; P80/90/100%; 10/15/20% Attack −15/24/33%, 2 turns. |
| 2845, 2839, 2846 | Kích Trạng Loạn Tâm | E1; 5/10/15% confusion, 2 turns. |
| 2692, 2777, 2784 | Lạc Thạch Vẫn Tinh | Enemy horizontal 3; M30/40/50%. |
| 4739, 4741, 4742 | Mê Tâm | Random enemy 2/3/4; 60/65/70% confusion, 2/2/3 turns. |
| 2838, 2843, 2844 | Phản Ứng Trì Độn | E1; 5/10/15% Speed −5/7/10%, 2 turns. |
| 3542, 3544, 3546 | Táng Thần Quần Thể | Enemy cross 4; P90/108/128%; caster rests one turn. |
| 3540, 3566, 3567 | Trào Phúng Quần Thể | Vertical 2/2 then cross 4; 66% taunt; 1/2/2 turns. |
| 2725, 2690, 2727 | Uy Nhiếp Ý Chí | Enemy horizontal 3; M30/40/50%. |
| 2615 | Thạch Hóa Công Kích | E1; 5% stone control, 2 turns. No synthetic levels 2/3. |

### 1.2 Passive and existing-plan corrections

- `Da Kháng Tính` 3808/3812/3813: before the owner's status tick/control check, roll 50/75/100%; on success remove one random removable harmful status and emit the matching Flash removal state.
- `Cân Bằng Chi Thược` 3763/3771/3772: MagicAttack bonus percentage is final Pet Energy × 0.05/0.10/0.15. The calculation is derived from the Pet's pre-skill `finalEnergy` and must not recursively compound during refreshes.
- `Tự Nhiên Thuẫn Đơn Thể`: HoT coefficients become 0.25/0.45/0.65 for levels 1/2/3. Defense remains +1000/+2000/+3000 and level 3 remains horizontal-five, matching seed.
- Add stone to planned control semantics so it prevents actions/skills, is resistance-aware, removable, and emits `buffAdd`/`buffDel` using buff ID 299.

## 2. Design and touch points

| Touch point | Change |
|---|---|
| `internal/application/combat/pet_active_skill_templates.go` or a new adjacent exact-plan file | Register the 28 exact templates. Keep exact IDs for special-source skills rather than broad name inference. |
| `internal/application/combat/pet_active_skill_specials.go` | Correct Tự Nhiên Thuẫn Đơn Thể HoT coefficients only. |
| `internal/domain/combat/skill_plan.go` | Add a stone control flag if the existing status type cannot represent planned petrification. |
| `internal/domain/combat/skill_executor.go` and status runtime | Apply stone using the same resistance, duration, icon, dispel, and turn-blocking pipeline as other planned controls. |
| `internal/domain/combat/pet_turn_start_passives.go` | Generalize exact turn-start passive ownership handling to include Da Kháng Tính without changing Di Hình Hoán Vị semantics. |
| `internal/application/pet/potential_skill_bonuses.go` or a focused passive-scaling helper | Add Cân Bằng Chi Thược Energy-based MagicAttack scaling without mixing it into fixed potential-skill coefficients. |
| `internal/application/combat/pet_active_skill_catalog_test.go` | Assert every listed exact ID resolves to the expected target, effect, coefficient, chance, duration, and rest rule. |
| Domain/application Pet and combat tests | Cover turn cleanse, stone lifecycle, independent AoE rolls, Energy scaling idempotence, and corrected HoT levels. |

## 3. Implementation order

1. Add exact active plans that use existing effect primitives; run focused catalog and executor tests.
2. Add planned stone control and its lifecycle regression; register ID 2615.
3. Add Da Kháng Tính turn-start cleanse with deterministic RNG tests at 0% boundary, level-1 failure/success, and level-3 guaranteed success.
4. Add Cân Bằng Chi Thược scaling with table tests for all levels and repeated recalculation idempotence.
5. Correct Tự Nhiên Thuẫn Đơn Thể HoT and update its regression.
6. Run full affected combat/Pet/handler tests, targeted vet, Windows and Linux builds, and diff check.
7. Update battle/Pet memory, fix history, and work-state checkpoints after each meaningful batch.

## 4. Verification

- Catalog coverage: all 28 exact active templates resolve and no unrelated skill is captured.
- Damage: physical/magic coefficients use the audited Pet stat pipeline; Táng Thần group rests exactly one following turn.
- Targets: horizontal/cross/random counts respect living legal targets and selected-target anchoring.
- Statuses: each AoE target rolls independently; confusion, taunt, slow, Attack/MagicAttack reductions, and stone expire and dispel correctly.
- Passive hooks: Da Kháng Tính runs before harmful status processing and removes at most one effect per turn; Cân Bằng scaling is stable across repeated refreshes.
- Playback: every affected target receives damage/status state; icon add/remove keys remain Flash-compatible.
- Commands:
  - `go test ./internal/domain/combat ./internal/domain/pet ./internal/application/combat ./internal/application/pet ./internal/presentation/rtmp/handlers/combat -count=1`
  - `go vet` on the same Go packages
  - `go build -o bin/gameserver.exe ./cmd/gameserver`
  - Linux `CGO_ENABLED=0` build into `bin/`
  - `git diff --check`

## 5. Non-goals and deferred decisions

- No ActionScript edits, migration, seed mutation, restart/deploy, player-data mutation, commit, or push.
- Do not implement Cận Vệ Thánh Nữ, Cung Chúc Tân Hỷ, Phản Xạ, Sủng Vật Hiệp Hộ, Thích Vũ, Tiếng Sét Phẫn Nộ, or nonexistent levels until their conflicts/missing values are resolved.
- Do not change NĐTT level 1 from seed 120% to forum 130%, Ma Pháp Thuẫn from four to three turns, Thuốc Kích Thích costs, Ảo Hoa level-3 duration, or Trào Phúng Đơn Thể level-1 duration as part of this batch.
- Do not infer behavior from `multiAttack` or other unproven fields.

## 6. Approval gate

Approved by the user on 2026-10-21 with the deferred list unchanged and seed-authoritative NĐTT level 1 at 120%.

## 7. Implementation outcome

- Registered all 28 approved exact active templates and pinned family, target, coefficient, proc, duration, buff ID, and rest behavior in table tests.
- Added planned stone control and exact buff 299 lifecycle without changing the Flash wire shape.
- Added Da Kháng Tính turn-start ownership rules at 50/75/100%, one removable harmful effect per successful roll, before periodic/control processing.
- Added non-recursive Cân Bằng Chi Thược scaling through `SkillStatBonuses.EnergyMagicAttackPerPoint`; repeated Pet stat refresh produces the same result.
- Corrected Tự Nhiên Thuẫn Đơn Thể HoT to 25/45/65% MagicAttack.
- Focused domain/application combat, Pet, and combat-handler tests PASS; targeted vet PASS; Windows and Linux gameserver builds PASS; diff check has no whitespace errors (Windows line-ending warnings only).
- No deferred family, ActionScript, seed, migration, database/player data, runtime deployment, commit, or push was changed.
