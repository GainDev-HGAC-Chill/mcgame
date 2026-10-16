# Character Skills, Fairy Skills, and Magic-Crystal Items — Audit

**Date**: 2026-10-16  
**Status**: Complete  
**Scope**: Character skill learning/upgrading authority, Fairy learning/upgrading/gameplay effects, and item templates 5947/5948/5949/5993.

No player data, item stacks, database rows, ActionScript files, or runtime services were changed during this audit.

## 1. Sources

- Decompiled Flash client under `docs/client/`, read-only.
- Checked game-data catalogs under `docs/database/game_data/` and seed files under `supabase/seeds/`.
- Current Go handlers, application services, domain models, repositories, and combat runtime.
- Read-only inspection of the local compose database for current `data_tbl_item_award` rows.
- Existing memory: `docs/memory/battle-combat.md`, `docs/memory/stat-feature-foundation.md`, and `docs/research/0000-00-00_00_CURRENCY_FEATURE_MAPPING_RESEARCH.md`.

## 2. Character skill wire contract

### Initial learning

Ordinary character skills are learned by using the matching type-505 skill-book item:

```text
useItem(1, -1, itemInstanceID)
```

The client does not call `skillLearnByClient` for the initial ordinary-skill learn. Special environments use their own RPCs: `addTrainingSkill`, `addGuildSkill`, and `addMagicWeaponSkill`.

### Upgrading

```text
skillLearnByClient(nextSkillID [, position]) -> Boolean
```

The Boolean is only an acknowledgement. Successful server state must be refreshed through:

```text
onSkillUpdate(skillList)
onMWeaponSkillUpdate(skillList, true)
```

The client locates the next row in the same skill family and checks `reqLevel`, `reqClass`, `reqCL`, `expSkill`, `price`, `costGuildContrib`, `dexSkill`, and context-specific `gold`. These client checks are advisory and must be repeated server-side.

## 3. Character skill server defects

### 3.1 Initial skill books are consumed without granting a skill

`internal/application/item/effect_skill_book.go` handles all type-505 items by returning a success message only. `NewEffectResult` remains consumable, so `UseItem` removes the book but no character-skill row is created. The current item catalog commonly stores `skillId=0`; the client resolves ordinary books by the catalog family/name rather than treating that zero as the learned skill.

### 3.2 Upgrade RPC is not authoritative

`skill.Service.UpgradeSkill` currently checks only character level, class, class rank, and rebirth. It does not:

- require an existing previous family level;
- require exactly `current level + 1`;
- reject unrelated training/guild/magic-weapon/life/Pet/Fairy skill rows;
- validate or debit skill EXP, silver, gold, guild contribution, or life mastery;
- require the initial skill book.

If no family row exists, the requested row is learned directly. If a lower family row exists, any higher requested row replaces it. A modified client can therefore learn unsupported skills, skip levels, and avoid all costs.

### 3.3 Battle trusts client skill level

`battleUpdateCmd` accepts the command's `level` field and the executor applies extra level scaling. The server only checks exact skill-ID ownership for a character actor; it does not replace the supplied level with the persisted entry. A client can therefore increase damage by sending an arbitrary level, and level-specific catalog rows may be scaled twice.

Pet actors also bypass the character ownership branch and currently have no equivalent slot-ownership check.

### 3.4 Legacy character-skill damage is based on progression fields

The generic gamedata adapter maps `expSkill` to base damage and `dexSkill` to multiplier even though the client and table reference identify them as learning/mastery requirements. This is not an authoritative combat formula. The audited Pet active-skill registry already bypasses this heuristic; unaudited character skills do not.

### 3.5 Persistence and recovery gaps

- Character creation persists the character before starter-skill seeding and only logs seed failures; there is no login repair.
- Skill-slot replacement clears the old occupant before persisting the new occupant, without a transaction or rollback.
- Life-skill upgrade persists skill and resource state in separate operations.
- Callback loading converts repository failure into an empty successful list.
- The database enforces `(character_id, skill_id)` uniqueness but not one row per family or one owner per `slot_position`.

### 3.6 Ordinary skill-book roster

The checked catalog contains 197 type-505 items, but they are not one homogeneous system. Matching a normalized book title against level-1 `TBL_SKILL` rows produces:

- 53 ordinary active-skill books (`useEnv=0`, `kind=1`), including book 1419 disambiguated to class `|6|` skill 3424 rather than unrelated class `|6@|` skill 60601;
- 12 ordinary passive-skill books (`useEnv=1`, `kind=2`);
- 6 life-skill books (`useEnv=6`), of which only 2409/2410/2411 are currently routed by the server;
- 1 special `useEnv=2` book;
- 125 type-505 items that cannot safely enter the ordinary character-skill path by title alone.

Therefore the normal book path must use a fail-closed catalog resolver constrained by item type, normalized title, item/skill class fingerprint, skill level 1, and allowed ordinary environments `{0,1}`. It must reject zero or multiple candidates. A global title-only lookup is unsafe.

The 65 ordinary books span all six classes. Each resolved family has a ten-level chain. Books 1414/1416/1418/1419 intentionally have item `reqLevel=1` while their level-1 skills require character level 80; skill requirements remain authoritative after book resolution.

The first learn consumes one book and has zero skill EXP/silver cost. Upgrades do not consume another book: the requested next row supplies `expSkill` and `price`. `expSkill` is the player's current `Experience`/client `expSkill` pool and is spent. `price` is silver; the server must use the selected silver wallet consistently and publish both the changed wallet field and `expSkill` after persistence.

### 3.7 Life-skill coverage gap

The server tracks five base life-skill types, but item routing and upgrading only cover three:

| Book | Skill | Type | Current state |
|---:|---:|---:|---|
| 2400 | 4931 | 14 Fishing | Unrouted; cannot safely learn/upgrade |
| 2408 | 4975 | 16 Herb gathering | Unrouted; cannot safely learn/upgrade |
| 2409 | 4965 | 15 Planting | Routed |
| 2410 | 4976 | 17 Cooking | Routed |
| 2411 | 4986 | 18 Medicine | Routed |

Item 2412 maps to skill type 19, which the client defines as Hidden Weapon, not a sixth life skill. It must not be admitted through the life-skill resolver.

Fishing and Herb rows currently fall through `skillLearnByClient` to the insecure normal upgrader, bypassing their intended life-skill costs. They require explicit life-skill definitions in both book routing and upgrade classification.

## 4. Fairy learning and upgrade contract

### Learn

```text
onSkill(fairyID, itemInstanceID)
```

The item must be type 521. The canonical level-1 TBL_SKILL ID is the book's `propType`; every checked Fairy book has `skillId=0`. The server must require the selected Fairy's growth-derived level to meet the book's `reqLevel`, reject duplicate families, require an available slot among `s1..s20`, consume one book, persist, and return:

```text
{flag:true, id:fairyID, sindex:"s1".."s20", skill:tblSkillID}
```

Failure uses `{flag:false,sindex:1|2}` according to the client contract.

### Upgrade

```text
upFairySkill(fairyID, skillIndex)
```

`skillIndex` is 1-based. Level 1→2 consumes one item 3583; level 2→3 consumes three; level 3 is maximum. The next row must be the same intended family and the next exact level. A global `codeName` match is insufficient because `skill_elf_13` collides with unrelated skill 6407.

### Config

```text
fairySkillConfigChange(fairyID, config)
```

The fire-and-forget config contains groups `0`, `1`, `2`, each with `c0..c4`, and `select` in `-1..2`. Only learned active Fairy skills are legal; duplicate family/name entries in one group are rejected by the client and must also be rejected server-side.

## 5. Fairy server defects

- The 22 book→level-1 mappings are present and the ordinary upgrade costs are correct.
- Learning does not enforce the book's Fairy `reqLevel`.
- Duplicate detection compares only exact skill ID, allowing a level-1 copy after the same family was upgraded.
- Upgrade searches globally by `codeName` and can select a collision.
- Missing `skillData` returns a false success with the unchanged ID.
- Book consumption occurs before whole-state persistence, with no rollback.
- Whole-roster JSON read/modify/write has no per-character lock or optimistic revision.
- Skill config is stored without shape, ownership, kind, or selected-group validation.
- Multiple Fairies can remain in guard state simultaneously.
- Deleting the guarded Fairy does not emit `onFairyOff`.

Most importantly, no stats or combat system reads Fairy `SkillFlag` or `Config`. The Flash client only manages UI and cosmetic rendering. Persisted Fairy skills therefore have no gameplay effect today.

The server also does not apply the active Fairy's base/growth stat contribution to the owning character. The client formula is display-only and cannot be treated as authoritative server mutation.

### 5.1 Exact Fairy family catalog

Every family has three skill levels. All target the Fairy owner. Passive skills are automatic; active skills are selected in a five-turn rotation (`c0` on turns 1/6/11, ..., `c4` on turns 5/10/15).

| Book | Skill IDs | Buff IDs | Kind | Book level | Effect by skill level | Engine status |
|---:|---|---|---|---:|---|---|
| 3577 | 5917–5919 | 2417–2419 | Passive | 11 | Physical attack +5/7/10% | Existing stat projection reusable |
| 3578 | 5920–5922 | 2420–2422 | Passive | 11 | Magic attack +5/7/10% | Existing stat projection reusable |
| 3579 | 5923–5925 | 2423–2425 | Passive | 11 | Speed +5/7/10% | Existing stat projection reusable |
| 3580 | 5926–5928 | 2426–2428 | Passive | 11 | Magic defense +5/7/10% | Existing stat projection reusable |
| 3581 | 5929–5931 | 2429–2431 | Passive | 11 | Physical defense +5/7/10% | Existing stat projection reusable |
| 3582 | 5932–5934 | 2432–2434 | Passive | 11 | Max HP +5/7/10% | Reusable; reconcile current/max HP |
| 3986 | 6227–6229 | 2721–2723 | Active | 12 | Periodic heal from 10/20/30% attack | Periodic heal exists; exact tick timing unresolved |
| 3987 | 6230–6232 | 2724–2726 | Active | 13 | Magic attack +5/8/13%, 2 rounds | Existing battle stat status reusable |
| 3995 | 6236–6238 | 2732–2734 | Active | 13 | Physical attack +5/8/13%, 2 rounds | Existing battle stat status reusable |
| 3996 | 6239–6241 | 2735–2737 | Active | 12 | Magic defense +10/20/35%, 2 rounds | Existing battle stat status reusable |
| 3997 | 6242–6244 | 2738–2740 | Active | 12 | Physical defense +10/20/35%, 2 rounds | Existing battle stat status reusable |
| 3998 | 6245–6247 | 2741–2743 | Active | 19 | Debuff resistance +5/7/12; Danh Y +30% | Values are prose-only; class modifier ambiguous |
| 4000 | 6248–6250 | 2744–2746 | Active | 14 | 60/70/80% on-hit defense reduction 10/15/20% | Trigger reusable; defense side/duration unresolved |
| 4001 | 6251–6253 | 2747–2749 | Active | 14 | 70/80/90% poison, 10/15/20% increasing damage | Damage basis/duration/stacking unresolved |
| 4002 | 6254–6256 | 2750–2752 | Active | 12 | 10/20/30% proc, drain 5/8/13% MP per turn | Needs negative-percent MP status; duration unresolved |
| 4003 | 6257–6259 | 2753–2755 | Active | 20 | Debuff accuracy +5/7/12% | Stat exists; combat accuracy integration missing |
| 4004 | 6260–6262 | 2756–2758 | Active | 15 | Speed +7/10/15%, 3 rounds | Existing battle stat status reusable |
| 4005 | 6263–6265 | 2759–2761 | Active | 18 | Each 20% HP lost grants attack +2/3/5% | Attack side, stacking, and cap unresolved |
| 4006 | 6266–6268 | 2762–2764 | Active | 17 | Reflect 5/8/13% incoming damage | Needs reflect hook and recursion/lethal rules |
| 4007 | 6269–6271 | 2765–2767 | Active | 17 | Final incoming damage −5/8/13% | Existing incoming-damage modifier reusable |
| 4008 | 6272–6274 | 2768–2770 | Active | 16 | 3/5/7% on-hit silence | Silence exists; duration unresolved |
| 4009 | 6275–6277 | 2771–2773 | Active | 12 | Convert 10/20/30% dealt damage to HP | Proc chance and damage basis unresolved |

`skill_elf_13` is not globally unique: intended Fairy skills 6266–6268 collide with unrelated skill 6407. Buff code names also collide. Family resolution must be constrained to the canonical book chain, not global `codeName` alone.

### 5.2 Fairy base/growth contribution

For the one guarding Fairy, the Flash panel displays each owner attribute contribution as:

```text
raw = base + (fairyLevel - 1) × (growthRate + 0.2 × growthLevel)
qualityPct = [20,25,30,35,40][floor((max(growthLevel,1)-1)/10)]
satiety = round(doh / 100)
satietyFactor = 1 - max(75 - satiety, 0) / 75
ownerBonus = round(raw × qualityPct × satietyFactor / 100)
```

The five Fairy fields map to Strength, Stamina, Agility, Intelligence, and Spirit. The current server persists all inputs but applies none of these contributions. Server authority first requires exactly one guarding Fairy; current whole-roster state can contain several.

### 5.3 Effects that remain fail-closed

Catalog fields are sufficient for six passive families and the straightforward active stat modifiers. They are not sufficient to invent timing or formulas for Thánh Ca, Đề Phúc, Chú Phọc, Độc Đao, Bí Diệm, Cổ Vũ, Chi Nộ, Chi Hoặc, or Chi Ngụy. These effects must remain disabled in authoritative combat until an operator decision or live/reference evidence resolves the listed ambiguities.

## 6. Crystal award IDs 73–75

Canonical mapping:

| ID | Character field | Client key | Meaning |
|---:|---|---|---|
| 73 | `MagicCrystalRec` | `magiccystalrec` | Bụi Ma Thuật / recovery dust |
| 74 | `MagicCrystalPre` | `magiccystalpre` | Pha Lê Vĩnh Hằng |
| 75 | `MagicCrystalLimit` | `magiccystallimit` | Pha Lê Cực Hạn |

Current local award rows:

| Item | Award |
|---:|---|
| 5947 | type 35, ID 75 × 200 |
| 5949 | type 35, ID 74 × 5 |
| 5993 | type 35, ID 75 × 10 |

The award IDs are correct. The error occurs because 73–75 are registered only through inline persistence accessors and are absent from `gameCurrencyDescriptors`. `effect_award.go` validates type-35 awards through `GameCurrencyDescriptor`, so it rejects 74/75 before mutation, consumption, or callback.

Additional defects:

- Item 5948 describes 20 recovery dust but has no award row.
- Stale wallet fallback entries for 5948/5949 would grant silver if reached.
- Generic `UseMultiItem` aggregates only HP/MP and discards structured currency grants, so multi-use can persist balances without sending `onAddMoney`.
- Checked `data_tbl_item_award.sql` is incomplete relative to the local compose catalog; reset durability requires a seed reconciliation, not a schema migration.

## 7. Safety conclusions

- Do not use a real skill book, Fairy upgrade book, or crystal item for verification.
- Character learning/upgrading must fail closed on missing family, context, cost, or repository state.
- Book/resource consumption and progression persistence need atomic or compensating behavior before exposing the fixed paths.
- Fairy active effects cannot be inferred from generic `propNum` fields alone; nonstandard families require explicit combat mechanics.
- No schema migration is known to be required for the crystal descriptor fix. Seed corrections must use the seed workflow.

## 8. Recommended delivery boundary

The safe implementation boundary is staged:

1. close character-skill and battle trust exploits;
2. repair ordinary/life skill-book learning and exact next-level cost handling;
3. repair crystal descriptors, deterministic award seed rows, and multi-use callbacks;
4. harden Fairy learning/config/guard persistence and apply base stats plus six passive families;
5. add only catalog-complete active Fairy effects first, keeping ambiguous effects fail-closed until their mechanics are approved.
