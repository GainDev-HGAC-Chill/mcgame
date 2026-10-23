# Training, Shop Quality, Pet Soul, and KP Rewards — Research

**Date:** 2026-10-23  
**Status:** Complete for Shop, Pet Soul, and KP reward; Training has partial official magnitude authority but remains blocked for full implementation

## Scope and constraints

- No ActionScript changes.
- No database, catalog, or player-state mutation during research.
- Flash call sites define wire behavior; checked-in game-data defines IDs and magnitudes.
- Missing progression formulas remain fail-closed.

## 1. Training skills

### Confirmed

- Flash learns the first level through `addTrainingSkill(skillId)` and upgrades through `skillLearnByClient(nextSkillId)` (`SkillUseSlot.as:420-469`).
- `GamePredef.TRAINING_SKILL_ORDER` declares 12 families (`GamePredef.as:2013-2026`).
- The authoritative catalog fingerprint is `TBL_SKILL.useEnv=3, kind=5`: 120 rows, 12 families, 10 levels each. Every row points to a distinct `buffId`.
- The linked 120 `TBL_BUFF` rows are permanent training descriptors (`effectNum/effectTime=9999999`) but every `prop1..prop6` and `propNum1..propNum6` is zero.
- Descriptions identify four stat groups affecting both character and Pet: HP, physical defense, magic defense, and speed, each with basic/intermediate/advanced families.

### Authority gap

- Only buff `1270` contains a numeric effect description: character HP `220 × class coefficient`, Pet HP `110 × element coefficient`. The other 119 rows do not provide magnitudes.
- The official 4399 character-training page at `http://web.4399.com/mlxt/gzfgj/rwxl.html` supplies all ten basic Speed, Magic Defense, and Physical Defense base values for character/Pet. Its level requirements exactly match the corresponding Vietnamese basic-tier catalog rows. The page labels the effect column `技能效果*职业系数` (skill effect × class coefficient), so the listed character magnitudes are base values that still require the class coefficient.
- Official basic Speed character/Pet values by level 1..10 are `20/10, 40/20, 60/30, 120/60, 180/90, 240/120, 320/160, 480/240, 700/350, 950/480`.
- Official basic Physical Defense and Magic Defense character/Pet values by level 1..10 are both `400/200, 800/400, 1200/600, 1600/800, 2200/1100, 3000/1500, 4000/2000, 5200/2600, 6800/3400, 8800/4400`. Defense levels 8/9/10 additionally list `0.5%/0.5%/1%` physical or magic immunity; the checked-in training buff rows do not encode these fields.
- A player-authored Baidu table independently matches the Vietnamese numeric HP row and supplies basic HP levels 1..8 as character/Pet bases `100/50, 220/110, 360/180, 520/260, 700/350, 900/450, 1120/560, 1360/680`. The answer describes the Pet side as multiplied by Pet quality coefficient, while the Vietnamese buff says Pet family coefficient; this conflict and the source's player-authored status keep it as corroborating evidence only.
- The original developer's Lezi `策划面对面` answer at `https://mc.lezi.com/plan.php?id=2835` confirms that HP training increases base maximum HP for the character and Pet, applies to Pets acquired after learning, and can subsequently be amplified by skills/equipment. A scan of all 626 preserved Q&A entries found no coefficient or missing-tier table.
- Neither Flash source, Go source, seeds, prior research, git history, nor the audited `I:\VPT` archives contains the missing class/Pet-family coefficient tables, intermediate/advanced curves, or the remaining basic HP values.
- `expSkill` is the learning cost and `dexSkill=-1`; neither is a stat magnitude.

### Conclusion

Learning persistence is implemented. The official page is enough to preserve 30 base rows, but applying the feature only partially or silently guessing coefficients/remaining curves would violate the fail-closed rule. The user explicitly authorized designing a replacement curve on 2026-10-23; its proposed constants, coefficient derivation, fail-closed boundaries, and non-original status are recorded in the implementation plan.

## 2. Gathering gloves and shop quality

### Confirmed catalog and shop rows

- Equipment `1123..1127` are the gathering gloves (`Thu Thập Hộ Thủ ...`) in `TBL_EQUIPT_TEMPLATE`; all have template `color=-1`, while `colorCode` contains asset-like values such as `165`, `89`, `50`, `180`, and `102`, not a runtime quality tier.
- NPC shop `32` (NPC `275`, `Tiệm Tạp Hóa Ông Mộ`) and mirror shop `82` sell `1123..1126` with `quality=0`, which is authoritative white/no-prefix intent.
- Both NPC editor and System Shop editor already expose the raw `quality` field, but only as an unlabeled number.

### Root cause

- Shop purchase converts `quality=0` to `colorCode=0` and calls `AddItemWithBindAndColor`.
- `AddItemWithBindAndColor` treats zero as “fall back to equipment template color,” normalizes the gathering glove template `colorCode` to the maximum runtime tier `5`, and therefore emits orange/`Trác Việt`.
- Shop `quality=0` must instead be an explicit white/no-prefix grant, not a request for template fallback.

### Safe correction

- Add an explicit-color item grant path for shop purchases so zero remains zero.
- Keep the existing fallback path for callers that intentionally request template defaults.
- Replace raw quality inputs in both shop editors with a labeled `0..25` quality selector/validated control. `0` means white/no prefix; positive values retain the established quality-to-color contract.

## 3. Pet Soul (Mệnh Hồn)

### Wire contract and current defects

- Single bag dissolve: `transformExp(slot, false)` (`PetSoulPanel.as:1856-1871`).
- Quick bag dissolve: `transformExp(colorThreshold, true)`, where UI thresholds are white `1`, green `2`, blue `3` (`PetSoulPanel.as:2490-2519`).
- Single temporary-bag dissolve: `transformExpInPanel(index)` (`PetSoulCanvas.as:202-230`).
- Quick temporary-bag dissolve: `transformExpInPanel(colorThreshold, true)` (`PetSoulProductPanel.as:858-887`). The server does not register this RPC.
- Manual EXP input sends a fraction `slider.value / needExp`, not an absolute amount (`SoulExpPanel.as:423-429`). The server currently truncates it to `int64`, so most inputs become zero.
- The client displays total soul EXP as `template.exp + slot.exp` and the next threshold as `template.exp + template.upExp` (`TipSoul.as:201-217`). Current dissolve returns only `slot.exp`, losing template base EXP.
- Level-up eligibility is `player.soulExp + slot.exp >= template.upExp`, and next-level metadata is the next sequential soul row (`sid+1`) (`PetSoulIcon.as:342-355`, `TipSoul.as:212-217`). Current level-up spends the full `upExp`, leaves the same `sid`, and accumulates EXP instead of advancing the soul level.
- Quick dissolve currently deletes every unlocked bag soul and ignores the threshold argument.

### Stat authority

- Every `TBL_PET_SOUL` row has exact `propType` and `propVal`; equipped Pet soul state stores the exact row `sid`.
- Base properties map directly to the existing Pet stat IDs (HP `1`, physical attack `4`, magic attack `5`, physical defense `6`, magic defense `7`, accuracy `8`, dodge `9`, counter `10`, speed `11`, critical `13`, and supported rate properties).
- Pet service currently attaches `SoulInfo` for display but never aggregates these values before `RecalculateStats`, so neither sheet nor battle stats change.

### Translation defect

- Soul rows `751..760` use Chinese name `永生` and mixed description `Tăng免死 ...`.
- Existing Vietnamese stat labels translate property `34` as `Miễn tử` (`Language.as:7866,8596`). The consistent localized name is `Vĩnh Sinh`, with description `Tăng Miễn Tử <value>`.

## 4. Box reward currency “Văn Chỉ”

### Confirmed ownership

- Văn Chỉ is PPVE `KP`, persisted in feature state `feature_key='pet_pve'`, field `state.kp` (`PPVEState.KP`).
- It is not `CurrencyPvePoint=238`; that scalar is a separate Pet Talent/PVE currency.
- PPVE panel refresh uses `onUpdatePPVEPanel(kpData, ppveData, null, false)`, where `kpData.kp` is the displayed balance.

### Required design

- Reserve a box-award game-currency ID for feature-owned KP without adding it to `character_currencies` or a `Character` field.
- Route that ID through a PPVE reward-granter interface which loads and persists the full PPVE state, increments `KP`, and returns panel payloads.
- Emit `onUpdatePPVEPanel` after use so an open panel is synchronized; do not alias the award to `pvePoint` or rely on generic `onAddMoney`.
- Add the exact option to the Dashboard box editor and regression tests for read/write/award behavior.

## Evidence boundaries

- Pet Soul callback method bodies are absent from the checked-in decompilation because `CallBack.as` exceeded the decompiler trait limit. The inbound RPC names and argument shapes above are directly observed at all UI call sites; existing server callback names should be retained unless shipped-artifact analysis supplies a stricter correction.
- Training stat magnitudes are only partially resolved: 30/120 base rows are official, eight basic HP rows are corroborated, and all coefficients plus the other tiers remain unresolved. Missing values are not safe to infer.
