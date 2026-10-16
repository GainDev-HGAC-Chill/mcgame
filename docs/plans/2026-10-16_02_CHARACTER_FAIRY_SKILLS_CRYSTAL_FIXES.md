# Character Skills, Fairy Skills, and Crystal Items — Implementation Plan

**Date**: 2026-10-16 · **Status**: Implemented — safe scope (Units A–H and unambiguous Unit I subset); runtime QA pending
**Goal**: Make character skill learning/upgrading server-authoritative, prevent client-forged battle skill levels, make Fairy learning/configuration safe and give catalog-complete Fairy bonuses real gameplay effects, and repair magic-crystal item awards 5947/5948/5949/5993 without touching ActionScript or player data.

Research base:

- `docs/research/2026-10-16_02_CHARACTER_FAIRY_SKILLS_AND_CRYSTAL_RESEARCH.md` — verified Flash wire contracts, 65-book ordinary roster, life-skill gaps, all 22 Fairy families, and crystal award root cause.
- `docs/memory/battle-combat.md` — existing skill ownership/callback and combat playback contracts.
- `docs/memory/farm-life-skill.md` — current three-life-skill path and callback requirements.
- `docs/memory/currencies.md` and `docs/memory/magic-crystal.md` — canonical currency registry and IDs 73–75.

## 1. Confirmed mechanics

### Character skills

- Initial ordinary learn: `useItem(1, -1, itemInstanceID)` with a type-505 book.
- Upgrade: `skillLearnByClient(nextSkillID[, position]) -> Boolean`.
- Success refreshes `onSkillUpdate(skillList)` and `onMWeaponSkillUpdate(skillList, true)`.
- Initial learn requires exactly one matching book and the level-1 skill requirements; it has no `expSkill`/`price` debit.
- Upgrade requires the already-owned preceding level in the same canonical family, exact `current+1`, `reqLevel`, `reqClass`, `reqCL`/rebirth, `expSkill`, and `price`.
- Ordinary book/upgrade paths accept only `useEnv in {0,1}`. Training, guild, magic-weapon, life, Pet, Fairy, and special skill rows remain in their owning subsystems.
- `expSkill` is `Character.Experience`; `price` is silver. The selected silver wallet is authoritative.
- Battle skill ID and level are server-owned. Client `level` is not authoritative.

### Fairy

- Learn: `onSkill(fairyID, itemInstanceID)`; item `propType` identifies the canonical level-1 skill.
- Upgrade: `upFairySkill(fairyID, skillIndex)`, one-based slot; costs item 3583 ×1 then ×3.
- Config: three groups `0..2`, five entries `c0..c4`, `select=-1..2`; only learned `kind=1` skills are legal.
- Exactly one Fairy may have `state=1`; its base/growth contribution and six passive families affect the owner.
- Active slots rotate every five owner turns. Only effects with complete catalog mechanics may be enabled without an operator formula decision.

### Crystal items

- 5947 grants currency 75 ×200; 5948 grants 73 ×20; 5949 grants 74 ×5; 5993 grants 75 ×10.
- Callback keys preserve client spelling: `magiccystalrec`, `magiccystalpre`, `magiccystallimit`.
- Multi-use must aggregate currency `Delta` and retain the latest absolute `Total`.

## 2. Implementation units

### Unit A — Canonical currency registry and item results

Files:

- `internal/domain/character/currency_keys.go`
- `internal/application/item/service.go`
- `internal/application/item/effect_wallet.go`
- focused tests under the same packages

Changes:

1. Add IDs 73–75 to `gameCurrencyDescriptors` and register them through the normal `registerIntGameKV` path; remove the parallel inline registration helper if unused.
2. Remove stale wallet fallback entries 5948/5949.
3. Aggregate `GrantedCurrencies` in `UseMultiItem` by client key: sum delta, retain the final total and label, and preserve `RefreshCharacterView`.
4. Keep unknown award IDs fail-closed.

Acceptance:

- IDs 73/74/75 resolve through both descriptor and persistence accessor registries.
- Single and multi-use expose exact `granted_currencies` results.
- 5947/5949/5993 no longer return `unsupported game currency id`.

### Unit B — Static award data durability

Files/data:

- compose database table `data.data_tbl_item_award`
- `supabase/seeds/data_tbl_item_award.sql`

Changes:

1. Write a checkpoint before any local catalog mutation.
2. Through the sanctioned compose database, add/verify deterministic rows for 5947/5948/5949/5993.
3. Export the full table seed through the repository seed workflow; do not hand-write a schema migration and do not touch player schemas.
4. Verify row counts and exact award IDs/amounts from the exported seed.

Acceptance:

- All four rows survive a seed review/reset path.
- No schema migration or player-data mutation is introduced.
- If the compose DB is unavailable, mark this unit blocked rather than synthesizing a partial seed.

### Unit C — Character skill catalog authority

Files:

- new focused catalog/resolver file under `internal/application/skill/`
- `internal/application/skill/service.go`
- `internal/application/skill/normal_skill_test.go`

Changes:

1. Classify skill rows by owning subsystem and reject non-ordinary rows in ordinary learn/upgrade methods.
2. Resolve initial books fail-closed from the checked catalog using normalized book title, type 505, item/skill class fingerprint, allowed environment `{0,1}`, `kind`, and level 1.
3. Require one canonical candidate; reject zero/multiple matches. Explicitly disambiguate book 1419 by class fingerprint, never by first global name/codeName match.
4. Require exact next level in the same constrained family. Remove the fallback that learns an arbitrary requested row when no family is owned.
5. Add a per-character mutation lock around learn/upgrade/position mutations.

Acceptance:

- All 65 ordinary books resolve; unrelated type-505 items do not.
- Direct level-2/10 requests, family skips, wrong class/rank/rebirth, and foreign subsystem rows fail closed.
- Duplicate exact/family learns do not consume a book.

### Unit D — Ordinary skill-book item flow and upgrade costs

Files:

- `internal/presentation/rtmp/handlers/item/use_books.go`
- `internal/application/item/effect_skill_book.go`
- `internal/presentation/rtmp/handlers/combat/battle.go`
- skill/item handler tests

Changes:

1. Route ordinary type-505 books through the skill service before generic item consumption, using the same pending/finalize/rollback shape as life-skill books.
2. Emit skill-list refresh only after book consumption succeeds; do not return a success message without a learned row.
3. For upgrades, validate and debit `Experience` plus the selected silver wallet using the requested next row's `expSkill` and `price`; persist skill/resource state with compensation on a later write failure.
4. Return the Flash Boolean responder and send existing skill refresh callbacks only after the complete mutation succeeds.
5. Push absolute `expSkill` and changed silver balance so the client does not remain stale.

Acceptance:

- A successful first learn consumes exactly one matching book and creates one level-1 family row.
- Failures consume neither book nor currency/EXP.
- Upgrade spends exact costs once; retries are idempotent and cannot skip levels.

### Unit E — Complete life-skill classification

Files:

- `internal/application/skill/service.go`
- `internal/application/skill/life_skill.go`
- `internal/presentation/rtmp/handlers/item/use_books.go`
- life-skill tests and `docs/memory/farm-life-skill.md`

Changes:

1. Add book routes 2400→type 14 Fishing and 2408→type 16 Herb.
2. Include types 14 and 16 in the authoritative life-skill upgrade path with their existing `fishDex`/`herbDex` definitions.
3. Keep item 2412/type 19 Hidden Weapon outside life skills.
4. Preserve the existing life-skill callback wire shape.

Acceptance:

- Five base life skills can be learned and upgraded only through the life-skill service.
- Fishing/Herb can no longer fall through to the ordinary free-upgrade path.

### Unit F — Combat command authority

Files:

- `internal/presentation/rtmp/handlers/combat/commands.go`
- `internal/domain/combat/turn_executor.go` only if needed to remove client-level influence
- Pet ownership helpers/tests

Changes:

1. Ignore the client-supplied `level` for damage authority.
2. Resolve character skill ownership from the exact persisted skill row and use its server level only where legacy behavior still requires a level.
3. Resolve Pet commands against the acting owned Pet's learned slots; reject arbitrary catalog IDs.
4. Keep basic attacks and existing catalog-driven Pet plans unchanged.
5. Do not invent new character damage formulas from `expSkill`/`dexSkill`; isolate that legacy adapter debt for a separate formula audit.

Acceptance:

- `level=999` produces no advantage over the same owned skill request with its normal level.
- A character/Pet cannot execute an unowned or wrong-actor skill.
- Existing Pet active-skill focused suites remain green.

### Unit G — Fairy mutation integrity

Files:

- `internal/application/fairy/service.go`
- `internal/application/fairy/skill.go`
- `internal/application/fairy/state.go`
- Fairy handler and application tests

Changes:

1. Add a per-character lock for whole-roster mutations.
2. Validate book type 521, canonical `propType` mapping, Fairy growth level against book `reqLevel`, and duplicate family rather than exact ID.
3. Resolve upgrades through the canonical 22-family chain, not global `codeName`; missing skill data becomes failure.
4. Validate config shape, selected group, five slots, learned ownership, active `kind=1`, and duplicate families.
5. Enforce one guarding Fairy by resting all others in the same saved mutation.
6. Return enough state from delete/change operations for handlers to send `onFairyOff` when required.
7. Use compensating item/state mutation behavior and tests so persistence errors do not silently lose books.

Acceptance:

- Under-level, duplicate-family, forged config, malformed slot, and collision requests fail closed.
- Concurrent learn/upgrade/config requests cannot overwrite each other in-process.
- Exactly one Fairy is authoritative for stats/combat.

### Unit H — Fairy base attributes and passive skills

Files:

- new Fairy bonus provider under `internal/application/fairy/`
- `cmd/gameserver/main.go`
- stat aggregation and encounter tests

Changes:

1. Implement the verified Fairy level/growth/satiety contribution for the one guarding Fairy.
2. Project the five raw Fairy attributes into character Strength/Stamina/Agility/Intelligence/Spirit.
3. Apply the six passive skill buffs through the existing percent-stat projection.
4. Register the Fairy provider with `itemService.AddCharacterBonusProvider` in both bootstrap paths.
5. Recompute effective stats on guard, rest, growth, feed, learn/upgrade, and delete refresh paths.

Acceptance:

- No guarding Fairy means zero Fairy bonus.
- One guarding Fairy produces formula-exact attribute and passive bonuses.
- Upgrading passive level changes only the intended stat percentage.

### Unit I — Catalog-complete Fairy active effects

Files:

- Fairy battle profile/rotation adapter under `internal/application/combat/` or `internal/application/fairy/`
- existing battle status/effect engine
- domain/application combat tests

Changes:

1. Snapshot the guarding Fairy and selected legal preset when constructing an encounter.
2. Advance `c0..c4` on the owner's attack turns only.
3. Implement only effects whose target, magnitude, and duration are complete in the catalog: magic/physical attack, magic/physical defense, speed, and final incoming-damage reduction.
4. Keep ambiguous families fail-closed and log a structured diagnostic rather than applying guessed behavior.

Acceptance:

- Rotation is deterministic and battle-local.
- Straightforward active buffs apply for exact rounds and expire through existing status ticking.
- Unresolved effects never become silent no-ops presented as successful combat effects; they remain explicitly unsupported pending decisions.

## 3. Deferred decisions

The following Fairy effects need operator mechanics or stronger reference evidence before implementation: Thánh Ca timing, Đề Phúc class bonus, Chú Phọc defense side/duration, Độc Đao damage progression, Bí Diệm duration, Cổ Vũ attack side/stack cap, Chi Nộ reflect basis/recursion, Chi Hoặc silence duration, and Chi Ngụy proc/basis.

The generic character-skill combat adapter's use of `expSkill`/`dexSkill` as damage is confirmed wrong semantically, but replacing it requires a separate per-family formula catalog. This plan removes client trust and prevents exploitation without inventing replacement damage numbers.

## 4. Verification

Automated:

1. Focused domain/application/handler tests for character skill, life skill, Fairy, item awards, item multi-use, currency registry, and combat commands.
2. Existing Pet active-skill suites to detect regressions.
3. `go vet` on changed packages.
4. `go build -o bin/gameserver.exe ./cmd/gameserver`.
5. `git diff --check`, documenting unrelated ActionScript/backup baseline separately.
6. Broad `go test ./internal/... -count=1`, compared against the recorded baseline rather than fixing unrelated failures.

Catalog invariants:

- 65/65 ordinary books resolve uniquely.
- 22/22 Fairy families contain exactly levels 1–3 and canonical buff IDs.
- 73/74/75 descriptor/accessor registries agree.
- 5947/5948/5949/5993 award rows are exact.

Runtime smoke tests are deferred until explicit deploy/restart permission. They must use test characters/items only and must not use Quest 166, Pet Nấm Lục, or real player inventory.

## 5. Documentation/checkpoints

- Update `docs/ai-memory/WORK_STATE.md` after every meaningful unit.
- Update `docs/ai-memory/FIX_HISTORY.md` after each completed defect group and record tests not run.
- Update/create feature memory for character skills and Fairy, plus `currencies.md`, `magic-crystal.md`, `farm-life-skill.md`, and `battle-combat.md` as applicable.
- Do not commit, push, deploy, restart, or modify player data.

## 6. Approval boundary

Approval of this plan authorizes Units A–H and the unambiguous subset of Unit I. It does not authorize guessed mechanics for the deferred Fairy families, deployment/restart, player-data testing, or unrelated cleanup.
