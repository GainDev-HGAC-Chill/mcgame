# Battle & Combat System

## Battle System

Turn-based combat with action constants: ATTACK=-10, DEFENCE=-20, ESCAPE=-30, TIMEOUT=-40, SKILL=-50, ITEM=-60, CATCH=-70, POSITION=-80, RETURN=-90, PET=-100, AUTO=-110.

Battle Position Grid (5x4, 20 positions across 4 rows):

- Row 0 (Enemy Characters): positions 4, 2, 0, 1, 3
- Row 1 (Enemy Pets): 9, 7, 5, 6, 8
- Row 2 (Player Pets): 18, 16, 15, 17, 19
- Row 3 (Player Characters): 13, 11, 10, 12, 14
- Enemy side: 0-9, Player side: 10-19

Skill Target Types (12): SELF=1, SELF_PLAYER=2, SELF_PET=3, ENEMY=4, ENEMY_PLAYER=5, ENEMY_CRE=6, ENEMY_NOBOSS=7, TEAM=8, TEAM_PLAYER=9, TEAM_PET=10, ALL=11, PLAYER=12. Skill Area Types: V=1 (vertical), H=2 (horizontal), C=3 (cross), R=4 (random).

Battle Status Effects: NORMAL=0, DEFENCE=1, DIZZY=10, CONFUSION=20, SLEEP=30, POISON=40, FIRE=50, ICE=60, LIGHT=70, SILENCE=80, REBELLION=90, STONE=100.

Behavior IDs: Move=1 (300ms), Attack=3 (500ms), Die=4 (800ms), Hurt=5, Defend=6 (400ms), Skill/Item=7 (500ms), Back=10080 (300ms), TurnBack=10100 (100ms).

Action sequence per attack: MoveToTarget (bid=1) -> Attack (bid=3, includes tSObj.hHp) -> Hurt (bid=5) -> Die (bid=4, only if target dies) -> Back (bid=10080) -> TurnBack (bid=10100).

Battle command flow: Player selects actions for player + pet, builds command objects `{tid, action, id, level}`, queues in `_cmdAry`, and sends them in one `battleUpdateCmd(_cmdAry)` batch when both selections are ready. The client still has a 4-second send-delay gate, but `battleOnStart()` backdates `_newRoundTime`, so the first manual round can send immediately once choices are made.

Pet battle empty ping: the Flash client can emit zero-arg `battleUpdateCmd` round-start pings (`AMF0 body = 0x05`, transaction_id=0`) when a pet is present. The server should not synthesize defend commands from those pings. Empty pings should wait only while at least one live player character still owes a command; if no live player-character input is required, the server must advance the round with pending or server-generated commands so pet-only or already-lost states can resolve cleanly.

Pet battle follow-up: `BattleUpdateCmd` accumulates partial player-side command submissions in `Battle.PendingCommands` and waits until every alive `SidePlayer` character (pets excluded from this check) has a command before calling `ProcessRound()`. Pet commands are auto-generated server-side via `generatePetCommands()` in `turn_processor.go` if not provided by the client.

Battle action `PET=-100` is a player-side pet roster command, not a playback action. The client sends it from bag-panel pet summon and recall flows with `id=<pet instance id>`. The server must resolve that pet ID to a real owned pet and translate it into `onBattlePlayList` behavior IDs `BH_RETURN(15000)`, `BH_RETURNED(17000)`, `BH_SUMMON(16000)`, and `BH_SUMMONED(18000)`. Treating `-100` like a normal attack makes pet switching appear broken even though the wire command arrives correctly.

Pet switch playback also needs two follow-throughs the Flash client depends on: append a trailing `BH_TURNBACK(10100)` after the cast sequence so the player returns to idle, and push `onUpdatePet(petID, "state", value)` updates so `_core.battlePet` flips to the newly active pet (or clears to rest when the current pet is recalled).

When a queued attack loses its target because that participant was removed earlier in the round, the miss fallback must use `tid=-1` instead of the default zero value. Leaving the missing-target attack at `tid=0` makes the Flash client animate enemy slot `0`, which looks like an unrelated creature was targeted.

## Battle Flow

Start: Client walks on map, `cbom` RPC triggers random encounter, server creates Battle, sends `onBattleStart`. onBattleStart fields: `battleFieldId`, `nid`, `battleType` (1=PVE, 2=PVP, 3=Arena, 4=Boss), `mapId`, `battleData`, `guest`, `sneakFlag`.

Turn: Client sends `battleUpdateCmd`. Server accumulates live player-side commands until the round is complete, then calls `ProcessRound()`: generates NPC commands, sorts actors by Speed (descending), executes each command, generates action sequence, sends `onBattlePlayList`. The payload is an array of sequential action groups; actions inside the same inner array run together, and `BattleStage.nextActionRound()` waits on the `delay` value of the last action in that group before advancing.

Timeout turns must finalize through the same combat end path as manual rounds. If timeout processing produces a dead-all-players or dead-all-enemies state, the server must recheck `CheckVictory()`, persist `EndBattle(...)`, clear battle deadlines, and send the same `{end, type, result}` semantics as the normal `battleUpdateCmd` handler. Leaving timeout results as playback-only causes battles to stay active until disconnect even when the character is already dead.

End: When all enemies or all players dead. Server adds `{end: true, type: winner, result: 1}` (`type=1` for player win as truthy, `type=0` for enemy win as falsy), sends `onEndBattle`. After battle ends, the follow-up `onUPP` must use the same equipment/stat-feature bonus pipeline as normal character refreshes; rebuilding the payload from base stats makes boosted HP snap down to the unbuffed max after `battlePlayEnd`.

Participant (cList) fields: `id`, `type` (2=character, 12=creature, 39=pet), `name`, `side` (0=player, 1=enemy), `level`, `hpMax`, `hp`, `mpMax`, `mp`, `spMax`, `sp`, `battleId` (position 0-19), `resCode`, `attack`, `defense`, `speed`, `isAlive`.

State object fields (sSObj/tSObj): `hHp` (damage playback HP value), `rHp` (new HP after healing), `uMp` (new MP), `cri` (critical hit), `frontEff`, `skillEff`, `bullet`. The Flash client renders hurt popups as `oldHp - hHp`, so lethal overkill hits must send an unclamped display `hHp` value even though server-side participant HP still clamps to `0`.

## Skill System Backend Flow

The current backend skill-system slice is driven by seeded gamedata plus per-character ownership rows:

- `data.data_tbl_class.start_skill` stores pipe-delimited starter skill IDs per class. `internal/application/character/service.go` now learns those skills immediately after a successful character create by calling the existing skill service.
- `data.data_tbl_creature_skill` stores `(cid, position, sid)` rows for creature-template skills. `internal/gamedata/cache.go` exposes `GetCreatureSkills(cid)` sorted by `position`, and both pet contract flow and combat AI reuse that lookup.
- New pets still normalize slots through `ensurePetSkillSlots`, then the first five open slots are seeded from `GetCreatureSkills(templateID)`. Closed slots remain `-1`.
- Combat skill metadata still flows `gamedata.Manager -> application/combat.GamedataSkillAdapter -> domain/combat.SkillGetter -> TurnProcessor`.
- NPC AI skill-pool resolution now prefers creature-template skills from gamedata, but `NPCAISelector.SelectCommand` still short-circuits to basic attack today; only the skill-pool plumbing changed.
- Player `battleUpdateCmd` skill actions now pass through a `HasSkill` ownership check in the combat handler. Unknown player skills are rejected before turn processing. Pet skill commands are intentionally not checked there because pet skills live in pet property slots rather than `player.character_skills`.
- `skillLearnByClient` is Flash-responder based: the client expects a `Boolean`, not a map payload. Normal skill learn/upgrade now runs through `internal/application/skill`, preserves the learned-entry family by `code_name`, optionally reapplies `position`, and refreshes the client via `onSkillUpdate` plus `onMWeaponSkillUpdate`. Flash may encode the requested skill ID as a numeric string and send `null` for an empty position; both are valid wire inputs, while non-numeric strings remain rejected.
- Normal upgrades are server-authoritative: only `useEnv` 0/1 families, exact `current+1`, class/rank/rebirth, `expSkill`, and selected silver can mutate the row. Initial ordinary learning is exclusively through one of 65 uniquely resolved type-505 books. Battle commands no longer apply client-supplied `level` damage scaling; character and Pet actors must own the exact submitted skill ID.
- Skill hotbar and auto-battle settings now round-trip across reconnects without a schema change. Character-side `interfaceData` keys persist under `player.characters.pm_process_data["interfaceData"]`, while active-pet battle settings persist in `player.character_pets.property`. Login rebuilds `interfaceData` from defaults plus the character bucket and the active pet bucket, `uif` saves generic interface keys, `skillSetUserBar` stores `stN/sidN`, `skillSetBattle` stores character `btN/bsN` or pet `btN/bsN`, and pet DTO `pi` now mirrors persisted `p6-p9`, `bs2/4/6/8/10`, and `bt2/4/6`.
- Combat skill playback now distinguishes close skills from ranged casts through `TBL_SKILL.type`-derived range metadata. Close skills (`type = 1`) approach the target for single-target playback, while remote/magic/support battle skills stay in place. The same range metadata also drives melee-vs-ranged row modifiers so remote physical skills are not penalized like melee attacks.
- Random map encounters now pick `TBL_MAP_CREATURE` rows by rolling against each row's `[startRate, endRate)` range per enemy slot, so duplicate creatures remain possible and rare tail ranges are preserved.
- Battle loot now comes from `TBL_CREATURE_LOOT`, filtered by active quest `qid`, persisted immediately into inventory/pet storage during `EndBattle()`, and sent to each player only from the post-battle `battlePlayEnd` callback path.
- Scene player payloads now go through one enrichment path for `sceneLogin`, `sceneChange`, `createChars`, and room-changing `toMovable`: active-pet display data is attached consistently, `getCharStateClient` returns real normal/battle state instead of `0`, scene payloads include `inBattle` and `state`, and grouped players also carry `inGroup`, `isLeader`, `groupAfk`, and `mList`. Movement broadcasts (`udcp` / `udcr`) now include `mList` too, because the Flash client's `onCR` follow logic uses that chain to move grouped characters correctly.
- `TBL_SKILL` is split across several different concerns rather than a single combat-formula contract. High-confidence battle fields are `type`, `targetType`, `targetNum`, `areaAttack`, `useMp`, `useSp`, `useHp`, `buffId`, `buffRate`, `buffRound`, and `attackNum` (`attackNum` = how many attack-play / damage-hit repetitions the skill should perform; each play deals damage once, and combat now repeats skill-hit playback/damage from it until the count is exhausted or the target dies); learn/progression fields include `reqLevel`, `reqClass`, `reqCL`, `price`, `expSkill`, `dexSkill`, `costGuildContrib`, `gold`, `guildDevExp`, `guildDevMoney`, and `useEnv`; pet/special-case fields include `creKind`, `scriptReqL`, `restoreSid`, `restoreStoneSid`, `exStoneSid`, `useItemId`, and `useItemType`. Fields like `multiAttack`, `element`, `exSid1/2/3`, `useItemNum`, and `isTest` still need deeper evidence before they should drive new server behavior.
- Battle skill-name display uses `sSObj["skill"]`. The Flash client handles `GamePredef.BS_SKILL` inside `BattleCreatureView.battleState()`, forwards it through `BattleStage.showSkill()`, and renders it with `SkillCanvas.flash()`. Server-side skill playback should keep the name on the opening action of the skill sequence so close-range skills flash their name reliably too.
- Skill playback now uses category-aware visuals: physical skills serialize attack behavior (`bid = 3`) instead of the generic magic-cast behavior, while magic/support skills stay on skill behavior (`bid = 7`). The live client battle-state path clearly consumes `frontEff`, `attackEff`, `skillEff`, `bullet`, `gfe`, and `gbe`, but current evidence still does not prove live battle-state handling for `buffEff` or plain `backEff`.
- Battle effects are also gated by the player setting `interfaceData["he"]`: login applies `useEffect = !GLOBAL_SETTING["he"]`, `BattleStage.useEffect` sets `CreatureView.LOAD_EFFECT`, and the effect loaders early-return when that flag is off. Missing battle effects can therefore be a client setting issue even when the server payload is correct.
- The previous short placeholder defaults such as `3004001` and `3001002` are not reliable battle-effect asset IDs. A real captured battle payload uses `attackEff: 2080130010008`, so the server now uses that observed encoded ID as the conservative default for physical hit playback and stops inventing default `frontEff` / `skillEff` / `bullet` IDs.
- `BattleStage` applies `sSObj` to the source view and `tSObj` to the target view. If a hit effect should appear on the target, it must live on `tSObj`; putting it on `sSObj` makes the client render it on the attacker.
- Skill targeting must treat client `tid` as either a participant ID or a battle position. Normal attacks already had that fallback, but `TargetResolver.ResolveTargets()` originally did not, which let skill commands like `tid: 0` fall through to random enemy selection. The resolver now falls back to `GetParticipantByPosition()` when numeric position lookup is needed.
- Generic character skill damage no longer treats progression fields `expSkill/dexSkill` as combat values. The adapter parses only description clauses that explicitly say gây/tạo `% sát thương` or `% tấn công`; buff-only percentages are ignored, dot/comma decimals are accepted, and multiple conflicting damage percentages fail closed to the neutral `1.0` multiplier because conditional branch semantics are not encoded in the generic adapter. Audited Pet active skills continue to bypass this generic path through exact plans.
- Audited Pet active skills now bypass those legacy heuristics through a family+level registry. It covers 144 active templates across 52 Pet-book families plus nine special-source templates for `Hổ Khiếu Sơn Lâm`, `Thợ Săn Sát Thủ`, and `Tứ Phương Khiếp Đảm` (153/55 total), compiling them to domain `SkillPlan` values for target overrides, Attack/MagicAttack multipliers, fixed/current/max-HP mechanics, costs, healing, revive, periodic effects, stat effects, controls, taunt, shields, one/all dispel, transfer, rest, and sacrifice.
- Đại Thánh Chí Tôn's `Bách Biến Trùng Kích` is another creature/evolution-only active chain outside the Pet-book roster. Exact IDs `6712..6720` use physical Attack multipliers `1.4/1.7/2.0`, evolved branch `1.6/1.9/2.5`, and alternate evolved branch `1.9/2.2/3.0`. These plans are keyed by exact template ID because catalog names vary by trailing spaces and `∮`/`ㄨ` decorations. The ambiguous `buffId=-2` Choáng/Độc/Suy Nhược selection remains fail-closed rather than guessed.
- `Vô Song Loạn Vũ` is the on-kill passive companion to that chain. Battle participants snapshot canonical Pet `skill1..skill15` ownership. If and only if a `Bách Biến Trùng Kích` plan kills its selected target, passive IDs `6721/6722/6723` invoke the matching `Trùng Kích Truy Kích` template against up to `1/2/3` distinct random living enemies without a second MP cost. Follow-up IDs `6724..6750` and physical multipliers `2.5..4.0` are selected from the exact Bách Biến branch and level; every wave executes that exact follow-up plan rather than a passive-owned synthetic/basic-attack effect. Defeated targets are excluded. If a follow-up wave kills at least one target, another same-size random wave fires; the chain ends when a wave kills nobody or no living enemy remains. Playback waves are separated so Flash replays each cast instead of merging them into one oversized AoE. A non-lethal initial hit, another killing skill, or a Pet without the passive never starts the chain.
- Thỏ Baby `Nhịp Đập Trái Tim` exact IDs `5363/5362/5361` start at MagicAttack coefficients `1.20/1.50/1.80`. Every pulse goes through the normal planned magic-damage pipeline, including defense, critical, shields, playback and reflection. After each pulse, continuation chance is the casting Pet's effective `Combo` percentage, including battle-local buffs such as `Thuốc Kích Thích`; it is not a fixed 30%. A successful roll casts the same NĐTT skill again without another MP cost. The current target is retained while alive; after a kill, the next pulse selects another living enemy, and the chain ends only on a failed roll, no living enemy, or the level-specific cap of `5/10/15` casts. Each pulse adds `0.20` to the MagicAttack coefficient and enforces damage greater than the preceding successful pulse; playback separators keep retargeted pulses as distinct Flash casts instead of one AoE. `Thuốc Kích Thích` levels 1–3 target the casting Pet itself: HP percentage cost and Speed/Combo/Critical buff apply to the Pet, never its owner character.
- `Di Hình Hoán Vị` is a turn-start passive keyed by exact owned IDs `6751/6752/6753`. Its helper rows `6754/6755/6756` define level 1/2/3 as a `20/30/40%` roll that removes every removable harmful status before control/periodic processing and grants `80/120/150` per-mille debuff resistance for that turn. The resistance is battle-local and participates in harmful planned-status probability as `base chance + attacker debuff hit - defender debuff resistance`; even a base-guaranteed status can be resisted. A failed next-turn roll removes the previous one-turn resistance instead of leaving it stale.
- Huyền Minh Linh Vũ's `U Minh Huyền Vũ` uses exact IDs `7002..7010` because the base/`∮`/`ㄨ` branches have different MaxHP ratios and base control chances. Each hit starts from `25% Attack + 15..30% caster MaxHP`, then enters the normal defense/critical/element/position/modifier pipeline. It keeps the selected target and expands horizontally across the paired enemy rows up to `4/6/8` living targets. `Thất Hồn` is buff ID `14`, lasts one turn, and rolls the catalog's `50..70%` base chance before generic debuff and stun-specific hit/resistance adjustments; the `ㄨ` branch is `60/65/70%`.
- Planned battle statuses with authoritative buff IDs now emit Flash's exact `buffAdd`, `buffDel`, and `buffColDel` state keys. Multi-target playback preserves a dedicated target-state round instead of dropping status state while clustering AoE actions. `Thợ Săn Sát Thủ` uses IDs `80118/80119/80120` and increases both Speed and Accuracy by `10/15/20%` for five turns. `Tứ Phương Khiếp Đảm` uses IDs `80121/80122/80123`, applies its Attack/MagicAttack/Speed reductions for one turn, and removes icons when effects expire or are dispelled.
- The KingGame cross-audit added 28 exact special-source templates that previously fell into legacy `expSkill/dexSkill`: Cao Tần Long Hống Trung Cấp, Đột Thích Xung Phong, Kích Trạng Loạn Tâm, Lạc Thạch Vẫn Tinh, Mê Tâm, Phản Ứng Trì Độn, Táng Thần Quần Thể, Trào Phúng Quần Thể, Uy Nhiếp Ý Chí, and level-1 Thạch Hóa Công Kích. Planned petrification uses `StatusStone`, buff 299, generic debuff resistance, normal dispel/icon lifecycle, and blocks both actions and skills. `Da Kháng Tính` exact IDs `3808/3812/3813` run before turn-start status processing, roll `50/75/100%`, and remove at most one random removable harmful effect. `Tự Nhiên Thuẫn Đơn Thể` keeps seed defense/targets but heals `25/45/65%` MagicAttack per tick by level.
- `Cận Vệ Thánh Nữ` exact owned IDs `3848/3849/3850` activate before every owner turn according to checked-in seed text. The passive replaces its own existing shield with a fresh three-hit pool equal to `50/75/100%` MaxMP and emits self `buffAdd` using `2015/2014/2013`, which all carry icon `4140200100250`. The server follows seed's every-turn behavior rather than the forum's conflicting first-three-round wording.
- Pet plan statuses are battle-local and overlay effective participant stats without mutating base stats. Same-family applications replace the existing instance, unrelated effects coexist, and normal attacks, counters, planned direct damage, periodic damage, invulnerability, sleep break, and magic shields share the status-aware damage path.
- `Thanh Phong Phục Sinh` uses selected-dead-ally resolution; `Đoạt Mệnh`/`Liêu Hồn` force selected single enemies; `Nuốt Chửng` accepts one selected living participant and consumes one removable effect of the appropriate polarity. Taunt is enforced by the shared single-enemy resolver while its caster remains legal.
- Pet direct skill base damage is exactly `Attack/MagicAttack × family multiplier` before defense, critical, element, variance, position, and existing damage modifiers; unlike the legacy skill path, it does not add the generic level damage bonus. Dodge, Combo, and Critical buffs add percentage points, while Attack, Defense, Speed, and stat-transfer percentages are relative modifiers.
- Pet plan damage playback must emit the same hit → Hurt → Die/Idle sequence as ordinary combat. The AoE round builder removes `hHp` from cast openers and moves it to the shared Hurt round; without a Hurt action, server HP changes and deaths occur but Flash receives no damage popup. Direct and fixed Pet damage now always carries one `hHp` playback result per positive applied hit/target before any death action. Misses, pure support skills, and initial application of DoT-only skills do not fabricate immediate damage; periodic damage emits its popup when the tick is actually applied.
- Periodic Pet damage must address the affected participant's real battle position and put `hHp` in the Hurt actor's `sSObj`. Flash calculates the popup as that actor view's previous HP minus the supplied new HP. Sending a default `tid=0` with `hHp` in `tSObj` routes poison/bleed ticks to slot 0 and can display a wrong number or no number. Positive ticks now emit Hurt → Die/Idle at the affected slot; fully absorbed ticks emit no fake popup.
- Area target caps are applied after dead targets are filtered. A dead nearer slot in a horizontal/random pattern can no longer consume the target allowance and reduce a configured multi-target skill to one visible/live victim when farther legal targets exist. Geometry remains authoritative: vertical/horizontal/cross skills can still hit fewer than their configured maximum when the relevant formation cells are empty.

## Battle Animation Effects

Effect constants (in `internal/domain/combat/battle_effect.go`): `BS_FRONT_EFFECT` = `"frontEff"`, `BS_ATTACK_EFFECT` = `"attackEff"`, `BS_SKILL_EFFECT` = `"skillEff"`, `BS_BULLET` = `"bullet"`, `BS_GLOBAL_FRONT_EFFECT` = `"gfe"`, `BS_GLOBAL_BACK_EFFECT` = `"gbe"`.

Database table `data.data_tbl_skill_effect`: columns `id`, `skill_id`, `cast_effect` (default 3000001), `hit_effect` (default 3001002), `bullet_effect` (default 0), `aoe_effect` (default 0), `buff_effect` (default 0).

Effect resource ID ranges: 3000001-3000999 (Attack), 3001001-3001999 (Hit/impact), 3002001-3002999 (Buff), 3003001-3003999 (Bullet), 3004001-3004999 (Cast/front), 3005001-3005999 (Back).

Client callback format (`onBattlePlayList`): Actions contain `sSObj` with `frontEff` and `bullet` fields, and `tSObj` with `skillEff` and `hHp` fields.

## BattleService (Background Worker)

Runs two tickers: Check Ticker (every 5s) calling `checkTurnTimeouts()`, and Cleanup Ticker (every 1m) calling `cleanupStaleBattles()`. Config: `TurnTimeout`: 30s, `CheckInterval`: 5s, `StaleTimeout`: 10m, `CleanupInterval`: 1m.

API: `SetTurnDeadline(battleID, playerID)`, `ClearTurnDeadline(battleID, playerID)`, `ClearBattleDeadlines(battleID)`, `GetRemainingTime(battleID, playerID)`, `GetStats()`, `OnRoundProcessed()`.

## Damage Formula

Damage calculation in `internal/domain/combat/damage.go` uses all character combat stats:

1. **Dodge**: `clamp((defender.Dodge - attacker.Hit) / 100, 0, 0.5)`
2. **Base damage**: `(attack * multiplier + skill.BaseDamage) * (1 + level * 0.02)`
3. **Defense penetration**: PraDef/PraMagDef reduced by defender ResiDefy (max 80%)
4. **Defense reduction**: `C / (C + effectiveDef)` where `C = 300 + level * 5`
5. **Enhancement**: `+EnhPhyHurt%` or `+EnhMagicHurt%`
6. **Damage reduction**: `-ReduceHurt1%` or `-ReduceHurt2%` (max 75%)
7. **Critical**: `clamp((crit - resiCrit) / 100, 0, 0.5)` -> `damage * (1 + critDmg/100)`
8. **Variance**: `* [0.9, 1.1]`, floor 1

**Combo**: After primary hit, chance = `min(attacker.Combo / 100, 25%)` for a second strike at 60% damage. Cannot chain.

**Counter-attack**: After all damage, chance = `min(defender.Counter / 100, 30%)` for defender to strike back at 50% multiplier. Cannot chain.

Participant struct carries 10 additional combat stats from equipment: Counter, Combo, PraDef, PraMagDef, ReduceHurt1, ReduceHurt2, ResiCritical, ResiDefy, EnhPhyHurt, EnhMagicHurt.

## Combat Encounter Balance

`basicBattleRate` interpretation: Client stores values like `50` for normal field maps. Server treats whole-number map rates as per-thousand values, so `50` becomes `5%` (previously treated as 50%). Standard `50` rate maps tuned to approximately 6%.

Encounter template normalization: Random encounter enemies built from capped level. Creature combat stats use template `att*` fields instead of pet-quality `apt*` values. Large raw `life` values normalized before mapping to battle HP.

## Revive System

Implemented all revive handlers: `ReliveUseItem`, `FreeRelive`, `BBGoldRelive`, `BBNormalRelive`. All restore HP/MP, save to DB, send `onUPP`. HP restoration also added in `ToSafe` (scene_handler.go) for cancel-on-revive-dialog.

## Battle System: Solo vs Pet Mechanics

Full client-side research from decompiled ActionScript: `docs/research/2026-04-11_00_BATTLE_SOLO_VS_PET_RESEARCH.md`.

**Battle grid**: 4x5 grid with 20 positions (0-19). Enemy slots are `0..9`, player slots are `10..19`. On the player side, `10..14` is the back row and `15..19` is the front row. The `setBp(bool)` RPC controls which participant gets the front row for the current connection: `bP=true` puts the player character in front (`15..19`) and the pet behind (`10..14`); `bP=false` does the reverse.

**Solo vs Pet**: Solo sends 1 command per round (immediately). With pet sends 2 commands (batched after both select actions). If player dies with pet alive, player auto-defends and pet continues.

**Battle start protocol**: Server sends `onBattleStart` RTMP client callback (not RPC response). Triggers: `hitNpc` (proximity), `clickBoss` (boss click), `clickNpc` (NPC click), or server-initiated random encounter on `CheckBattleOnMove`.

**`onBattleStart` data**: Top-level fields: `battleFieldId`, `cList`, `guest`, `sneakFlag`, `bossFlag`, `airBattle`. `cList` is keyed by raw battle slot. Participant `type` values follow the Flash constants: `2=character`, `12=creature`, `39=pet`. With `bP=true`, reference pet battles place the pet at battle slot `10` and the player character at battle slot `15`; with `bP=false`, those two player-side rows swap. Participant DTO fields are type-specific: creatures include `bossFlag` and `withCloud`; pets include `cid` and optional `c`; players include `sp`, `spMax`, `star`, `ee`, `ef`, `en`, and optional `wp`.

**Battle-start callback contract**: Match the original Flash flow before `onBattleStart`. Send `onUpdatePet(petId, "currentHp", value)` for the active player pet, then `onSetCharState {cid, state: 5}`, then `onBattleStart`. Do not mutate the pet `id` at battle start. The client uses `onSetCharState` to move the player object into `ST_BATTLE`, and missing that callback can push the client onto the empty/no-arg `battleUpdateCmd` path.

**Grouped PVE start/playback**: For grouped PVE, `onBattleStart` is sent to every player-character participant individually, then `onBattlePlayList` is fanned out through battle watchers so all grouped members and spectators receive the same playback stream.

**Grouped round readiness**: Shared PVE rounds wait for live player-character commands only. Pet actions may still arrive from the client, but the turn processor can auto-generate pet behavior, so pets no longer block grouped round execution.

**Grouped timeout and cleanup**: Timeout resolution now merges already-submitted commands and fills timeout commands for every still-missing player character in the battle once per expired battle. If no live player characters are missing, timeout processing must use the already-submitted commands as-is instead of inventing a fallback timeout command. Ended battles are cached briefly so each participant can still complete `battlePlayEnd` after the active battle is removed.

**Grouped reward split**: Shared PVE EXP and money are stored as per-survivor shares before reward application, so each surviving grouped character receives one equal split of the total character reward pool.

**Grouped disconnect handling**: When one grouped participant disconnects and other live player characters remain, keep the battle active. Persist the disconnecting character's live HP/MP/SP, queue `ClientActionEscape` for that actor in `PendingCommands` as the grouped-leave variant, remove only that participant's deadline, and let the normal shared round emit `BH_ESCAPE` (`10130`) then `BH_ESCAPED` (`10170`) so the remaining clients see the member leave in turn order instead of through an out-of-band callback.

**Battle watcher cleanup**: Grouped battle playback fans out through `battle.Watchers`. Disconnecting members must be removed from that watcher list before later `onBattlePlayList` broadcasts or the server will keep trying to write callbacks to dead RTMP connections and spam timeout errors in Docker logs.

**Action playback**: Server sends `startSequence(actionGroups)` — array of action groups executed sequentially, actions within each group run simultaneously. Each action: `{sid, tid, bid, sSObj, tSObj, delay, end, result}`. The Flash client advances to the next group after the final action's `delay`, so end-of-turn pacing belongs on the terminal action in that group chain (for the current server flow, `BH_TURNBACK` is the safe place to add inter-actor delay). Behavior IDs: 0=idle, 1=run, 3=attack, 7=magic, 10080=return, 10100=turnback, 12000=revive, 14000=swap, 16000=summon.

**Auto-battle**: Separate priority chains for player (`battleAuto`) and pet (`battlePetAuto`): HP threshold → MP threshold → team heal → pet heal → auto-defence → main attack skill → fallback normal attack.

**Fairy battle effects**: PVE and PVP snapshot the sole guarding Fairy's validated selected five-slot preset at encounter creation. The battle-local cycle advances on owner attack/skill/auto turns. Six catalog-complete stat/reduction families use the status engine; ambiguous families remain explicit unsupported entries and log instead of applying guessed effects. See [Fairy System](fairy-system.md).

**Existing server implementation**: `internal/presentation/rtmp/handlers/combat/battle.go` (HitNpc), `encounters.go` (CheckBattleOnMove), `internal/domain/combat/battle.go` (ToDTO with cList).

## Battle System: Client Protocol Alignment (2026-04-11)

Fixed 11 gaps between server implementation and Flash client expectations based on decompiled ActionScript research.

**ParticipantType constants**: Research doc incorrectly listed 1/2/3. Actual Flash client GamePredef.as values: TBL_CHARACTOR=2, TBL_CREATURE=12, TBL_PET=39. Server values were already correct — do NOT change them.

**DTO field types**: Flash client reads resCode, bossFlag, withCloud, iconCode, colorCode as Numbers. Server was sending formatted strings. Now sends int values directly. bossFlag is always present (not conditional). Pet DTOs include cid (owner character ID).

**Behavior IDs**: Keep existing defend/defended mappings for client compatibility: defend stays 0 and defended stays 6. Added TurnTo (BH_TURNTO) = 10110 for melee sequencing without changing the older defend animation behavior.

**Melee vs Magic movement**: Melee attacks (ActionTypeAttack) follow: TurnTo -> MoveToTarget -> Hit -> Hurt -> Back -> TurnBack. Stationary magic/heal skill casts do not return to idle on the client by themselves, so they must end with `TurnBack` after the cast resolves. Counter attacks also include TurnTo before movement.

**Skill name display**: sSObj.skill field set on ActionTypeSkill actions so client can flash skill name on screen.

**CheckVictory with pets**: Battle continues as long as ANY participant on a side is alive. If player character dies but pet lives, battle continues (pet fights, dead player auto-defends). Battle ends only when all participants on one side are dead.

**Battle end result semantics**: result: 1 = Win (show battle report), result: 0 = Lose, result: 2 = Fled. Server previously sent result=1 for all non-flee endings.

**Battle trigger RPCs**: `hitNpc` and `clickBoss` route through the combat handler. `clickNpc` is owned by the NPC handler; normal NPCs open interaction UI there, while quest-battle NPCs such as type `20` bridge back into combat by starting a standard PVE battle from the active quest kill target.

Files changed: internal/domain/combat/battle.go, turn_builder.go, turn_executor.go, internal/presentation/rtmp/handlers/combat/handler.go, commands.go.
