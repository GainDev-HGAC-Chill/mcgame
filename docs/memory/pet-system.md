# Pet System

## Overview

Properties: petList, activePetObject, contractPet, evolutionPetObject, petMaxNum. Auto-battle settings: bs2/4/6/8/10 (skill/item assignments), p6/7/8/9 (HP/MP thresholds). Auto-battle logic checks thresholds before choosing heal, defense, or attack.

## Pet Leveling

Pet leveling now follows the Flash client `GamePredef.PET_LEVEL_EXP` table instead of the old quadratic formula. Server-side pet `Experience` remains the in-level remainder, while client-facing pet `exp` callbacks/DTOs add the cumulative base from `PetLogic.lvToExp(level)` so Flash progress and level-up thresholds stay aligned.

Pet growth medicines are pet-target `type=501`, `useType=2` items handled before ordinary HP/MP potion behavior. The exact `GamePredef.PET_GROW_ITEM_NUM` mapping is `961=30000`, `962=150000`, `964=300000`, `1585=300000`, `1586=2000000`, `1587=6400000`, `1836=12500000`, and `3339=100000000` EXP. `useItem(2, petId, itemInstanceId)` and multi-use must propagate the Pet ID, validate ownership and the template's required level against the Pet rather than the character, reject a max-level Pet without consuming, persist through `pet.Service.GainExperience`, then emit cumulative client `exp`, changed `level`, and `onRefreshPetProp({id,s})`. Only these eight checked IDs use this rule; other type-501 items retain normal potion behavior.

## Pet Attribute Points

Pet attribute-point flow now mirrors the character path: each pet level grants `5` distributable points, `Pet.RecalculateStats()` mirrors the remaining pool into `property.lastPoint`, and `changePetProperty(petId, {addStrength, addAgility, addStamina, addIntelligence, addEnergy})` spends those points into `Apt*Ex` plus `onUpdatePet`/`onRefreshPetProp` callbacks for the Flash pet panel.

`petXd(petId, itemInstanceId)` uses Quy Nguyên Đơn template `1526` only for bound-shop class `classIds=10` Pets. New allocations carry a per-stat ledger; legacy rows can reset only when the sum of all five `Apt*Ex` values exactly equals persisted `distributed_attr_points`, so the item is never consumed on ambiguous data. PostgreSQL readers return `SETOF player.character_pets`, whose physical tail order is `attr_points, distributed_attr_points, max_attr_points, life`; all three repository read paths must preserve that order. Scanning max before distributed caused legacy Pet 300 to appear as distributed `430` instead of `440`, making an otherwise unambiguous reset fail with `Invalid input`.

## Pet Illusion Contract

`petEat(mainPetId, costPetId, luckItemId, luckCount, aptitudeType, passwordHash)` is the Flash Ảo hóa flow. The responder reads exact keys `succ`, `msg`, `costPet`, `costItem`, `propName`, `value`, `aptType`, and `petId`; returning only `{success, pet}` mutates server state but leaves the panel looking unchanged. The third argument is the actual lucky-stone item instance ID, not a boolean.

Both Pets must be purple or orange and the main Pet must be bound. Base success rate mirrors `PetFuncPanel.setEatInfo`: each Pet contributes 15%/30% when divine (`classIds=10`) or 10%/15% when ordinary at purple/orange; matching `classId` adds 20%, otherwise the subtotal is halved; each valid propType-7 lucky stone adds 20%, capped at 100%. The cost Pet and selected stones are consumed on both success and failure. Success replaces only the previously tracked illusion bonus for the selected aptitude with 10–15% of a purple cost Pet's base aptitude or 25–30% for orange, preserving allocated/cultivation bonuses in the same `Apt*Ex` field. Persistence uses one `IllusionRepository.ApplyIllusion` transaction that locks both Pets and the optional stone stack, then updates the main Pet, deletes the cost Pet, and decrements/deletes the stones atomically; cache state changes only after commit.

## Pet Life Meter

Pet `life` is now a persisted battle-condition meter on `player.character_pets.life`, not an HP-derived display field. New pets start at `10000`, battles drain `10` from each owner-side pet participant on battle end, zero-life pets cannot enter battle or be swapped in, rest/follow states still work at zero life, and pet functional items restore this stored meter back up to the `10000` cap via `onUpdatePet(..., "life", ...)`.

## Pet Cache Read Order

`internal/infrastructure/cache/cached_pet_repo.go` must prefer `activeSessions` over `coldLoader` for pet reads (`FindByCharacterID`, `FindFollowingPet`, `Count`) and for `ClearFollowing`.

Reason: pet cold cache can be stale relative to the active session. If cold data is checked first, two visible bugs appear:

- `getFinalPraDefPet` / `finalPraMagDefPet` can fail with `following pet not found` even right after `onRefreshPetProp` shows `state:1`
- `initViewPetMngP` can return an old or empty pet list after relogin or after in-session pet changes

`CachedPetRepository.Save` and `Delete` should also keep cold pet cache in sync when possible so Redis cold data does not lag behind hot session state.

`initViewPetMngP` must return wrapped pet DTOs from `Pet.ToDTO()` and the top-level payload must be keyed by pet ID, not a positional slice. The Flash `PetManagerPanel.onInitViewPetMngP()` expects `pet.data`, flattens it client-side, then stores the result in `_core.player.petList` and later indexes that structure by `petId` in callbacks like `onUpdatePet` and `onRefreshPetProp`. Returning flattened `petDTOData()` breaks `_local_2.data.tid`, and returning a slice breaks `_core.player.petList[petId]`, so list display may work while pet callbacks silently miss.

Pet slots resolve hover data separately through `gdc(39, petId)`. That path must return the current character's flat `Pet.ToDetailDTO()` as `{type:39,index:petId,data:...}`; an empty TBL_PET compatibility response leaves `TipCre` without skills and attributes. The detail DTO must include root `skill1..skill15`, `attStrength`, `attAgility`, `attStamina`, `attIntelligence`, and `attEnergy`, because `TipCre.setInst()` reads those fields directly. Ownership remains enforced by `pet.Service.GetPetDetail`.

## Pet Star Enhancement Contract

`petStar(petId, itemInstanceId, safeItemInstanceId)` consumes a Pet-function Nội Đơn with item template `type=507`, `useType=4`, `propType=3`. The base success rates must match `GamePredef.PET_STAR_SUCCESS` as indexed by target star: `100/100/90/80/70/60/50/30/30/30/30/20%` for stars 1→12. On success it increments `upgradeNum` and adds `proplNum / 100` to persisted `GrowRateAdd`, rounded to two decimal places, then recalculates Pet stats. On failure below 8 stars both current stars and growth addition are retained. At 8 stars or above without protection, both `upgradeNum` and `GrowRateAdd` reset to 0 and Pet stats are recalculated; valid Pet-function protection `propType=6` (checked template 2276) retains both values. Attempt and protection items are consumed, and the responder uses `flag` plus `starSafeNum` for the Flash result flow. The checked-in enhancement templates currently encode growth additions from `0.02` through `0.13`; the value must come from game data rather than an item-ID switch.

The responder returns the updated root `growRateAdd`, while normal inventory callbacks remove or decrement consumed instances. The handler must additionally push `onRefreshPetProp({id,s})` after `petStar` and `petStarClear`, because `PetPanel`/`PetManagerPanel` calculate effective growth from nested `property.growRate + property.growRateAdd`; updating only the root leaves the panel showing stale base growth (for example `1.56` instead of effective `1.56 + 0.40 = 1.96`).

`petStarClear(petId, itemInstanceId)` accepts only Nội Đơn Phụng Hoàng template `2245` whose catalog contract remains `type=507,useType=4,propType=5`. A valid use resets both `upgradeNum` and `GrowRateAdd` to zero and recalculates stats before consuming one pill. Any other item, including a lookalike Pet-function item with a different `propType`, is rejected without Pet or inventory mutation.

## Pet Client State Contract

Flash pet UI does not treat `state` as the same thing as server-side `is_following`.

- `PetManagerPanel` and `MAIN_PET` read root `pet.state`, not just `pet.property.state`
- `state=1` is the active/battle pet shown in `MAIN_PET`
- `state=2` is the follow icon path in the pet list
- `state=3` is resting

Because of that, server DTOs must emit `state` at the root level and also keep `property.state` populated for refresh flows like `onRefreshPetProp`.

`PetPanel` also reads `pet.property.aptStrength`, `aptAgility`, `aptStamina`, `aptIntelligence`, and `aptEnergy` when building tooltip/final aptitude displays. Those aptitude keys need to be present in the property block, not only at the root DTO level.

`changePetState` only controls the active battle pet (`state=1`). It must not change `is_following` or send scene follow callbacks. Visible follow pets still belong to `startPetFollow` / `cancelPetFollow`, while combat setup and pet EXP rewards must resolve the battle pet from the pet list entry with `state=1`.

`initViewPetMngP` and `changePetState` should both refresh `creatureData` through `onUpdatePet(petId, "creatureData", data)` for every pet that needs panel rendering. `PetManagerPanel.updateView()` only sets `_core.battlePet` when both `state == 1` and `creatureData` are present.

## Pet Slot Opening Contract

`petOpenSlot` is a bag/pet-panel flow, not a responder-driven panel RPC. The Flash client calls `_core.remote.petOpenSlot()` directly and expects state changes through callbacks.

- Slot price is sourced client-side from `getTemplateData(29, 326).gold`, which is currently `20`
- Currency behavior is `goldBind` first, then `gold`
- `BagPanel` only blocks the call when `gold` would be needed but `gold` is still locked
- The required callbacks are `onMinusMoney(playerId, moneyType, cost, currentNum)` and `onUPP({petMaxNum: newSlotCount})`
- `CallBack.onUPP` already refreshes both the pet manager panel and the bag pet tab when `petMaxNum` changes
- The supported UI cap is `180` pet slots because `GamePredef.MAX_PET_PER_PAGE = 18` and the bag exposes 10 pet pages

## Pet Follow Scene Contract

Visible follow-pet rendering is scene-driven, not responder-driven.

- `startPetFollow` responder only updates `_core.showPetId` in `PetManagerPanel`
- the actual pet sprite in the map is controlled by scene callbacks
- `cancelPetFollow` must send `onCancelPet(ownerCid)`
- `startPetFollow` must send `onCancelPet(ownerCid)` first, then `onCreatePet({cid: ownerCid, petData})`

The key for `onCancelPet` is the character id of the pet owner, because `ViewManager.addP()` stores pets by `_arg_1.cid`.

`petData` for `onCreatePet` must be a flat scene object compatible with `Core.createPet()`, not the wrapped panel DTO from `Pet.ToDTO()`. A dedicated scene DTO should include at minimum:

- pet identity: `id`, `cid`, `tid`, `name`, `level`
- combat bars: `currentHp`, `currentMp`, `hpMax`, `mpMax`, `life`
- appearance: `resCode`, `iconCode`, `colorCode`, `element`
- follow stats: `property`

Login and scene-enter payloads also need `showPetObj` when `showPetId > 0`; sending only `showPetId` is not enough for client rendering after relogin or when another player enters the room.

Summon books enter this same follow contract through item use. Flash additionally gates follow rendering on long-buff 1709 (standard orange summon) or 1979 (advanced purple/orange summon), so consuming a book without adding its timed buff is not sufficient. Exact item-to-buff durations are 2514→1709/1h, 2543→1709/30d, and 2798→1979/30d. Only bound Pets qualify; standard requires `qLevel >= 3`, advanced requires `qLevel >= 2`. The item handler preflights eligibility before consumption, persists follow state, then broadcasts the normal hide/create scene callbacks.

## Pet Skill Slot Opening Contract

`petOpenSkill` is a direct pet-panel RPC. The Flash client calls `_core.remote.petOpenSkill(petId, index)` and expects callback-driven updates instead of a responder payload.

- only `skill6..skill15` are paid unlock slots
- `skill1..skill5` are the default open slots and should normalize to `0`
- slot state meanings are `-1` = locked, `0` = open but empty, `>0` = learned skill template id
- base open costs by slot are `10, 20, 40, 70, 110, 160, 220, 290, 370, 460`
- the client multiplies those base costs by pet quality through `GamePredef.GOLD_PET_SKILLOPEN_Q`, which is `1x`, `3x`, or `10x` for `qLevel` `1`, `2`, or `3`
- the required success callbacks are `onMinusMoney(playerId, moneyType, cost, currentNum)` and `onUpdatePet(petId, "skillN", 0)`
- after `onUpdatePet`, the pet panel itself triggers `getFinalPraDefPet` and `finalPraMagDefPet`, so `petOpenSkill` does not need to push an extra pet property refresh callback

## Pet Guard Contract

`getPetGuardData`, `putDownPet`, and `movePetToPetGuardSid` are panel-driven pet guard RPCs. The shared refresh callback is `onUpdatePetGuardData(payload)`.

- `getPetGuardData` should both return the payload and emit `onUpdatePetGuardData`, because one guard panel uses a responder and the other calls it with `null`
- payload shape is `{lvData, petData}`
- `lvData` keys are `0..4`
- `petData` keys are `0..4`, `11..15`, `21..25`, `31..35`, `41..45`
- outer level key `0` is the shared outer guard level
- inside level keys `1..4` correspond to the four inner guard groups
- valid guard pets must satisfy creature `classIds == 10` and `useLv >= 50`
- one pet cannot appear in multiple guard slots at the same time
- current persistence is `player.characters.pet_guard_data jsonb`

`petguardout` and `petguardin` are separate client-visible currencies/items for the guard-upgrade UI, not placement-state fields. Their acquisition/spend flow is still unimplemented in the current Go server.

## Pet Skill Book Contract

`petBook` is a responder-driven pet panel RPC. The Flash client calls `_core.remote.call("petBook", new Responder(onBook), petId, bookItemId)`.

- the second argument is the dragged pet skill book item instance id from inventory, not a skill template id
- responder success is `f = 1`, `rf = true`, with `n` as the remaining stack count
- learn failure is `f = 1`, `rf = false`
- invalid or unusable skill book is `f = 2`
- invalid pet id, invalid drag source, or other bad input is `f = 3`
- usable pet skill books come from `data.data_tbl_item_template` with `type = 506` and `use_type = 2`
- ordinary books resolve by exact item name against `data.data_tbl_skill`, choosing the lowest skill level in that name family; the 11 Super books use the exact replacement contract below instead
- ordinary `petBook` learning uses the first skill slot whose value is `0`; it must not write into locked slots with value `-1`
- duplicate and conflict checks need to compare the learned skill family, not only the exact skill id; the current server matches by shared `Name`, `CodeName`, or `ScriptReqL`
- after a successful learn, the server must emit `onUpdatePet(petId, "skillN", skillTemplateId)` and refresh pet derived properties so the panel updates final stats immediately
- passive pet skill bonuses are derived from `TBL_SKILL.buff_id` into `TBL_BUFF.prop*` and `prop_num*`, then merged into pet `final*` combat stats
- example mapping:
  - item template `3216` = `Thụ Bì Ngoại Sáo Siêu Cấp`
  - learned skill template `5527`
  - buff template `2113`
  - effect: `hpMax` and `finalHp` increase by `22%`

Battle skill ownership must use these same canonical `skill1..skill15` slots through `Pet.LearnedSkillSlots()`. The older `property.skills` array is not populated by the current learn/upgrade flows and must not be used as combat authority. After exact skill-ID ownership succeeds, the authoritative level comes from that skill row in `TBL_SKILL`; a different level-row ID that is not present in a Pet slot remains rejected.

Name-based Pet-book resolution accepts canonical Pet environments `useEnv=0/1/2`, excludes special rows such as `useEnv=-1`, and fails closed if the lowest eligible level is still ambiguous. Environment 2 is required by valid active/status books such as item 1622 `Thuốc Kích Thích`, which resolves to `3590→3672→3673`. In particular, book 2242 `Man Lực` resolves to `4172→4171→4170`, not special row `60903`; book 1883 `Da Kháng Tính` starts at `3812→3813`, not row `3808`. Legacy slots produced by the old resolver use those two exact aliases as upgrade sources so the next legitimate `petUpSkill` repairs the slot while preserving normal manual cost and rollback behavior.

### Pet Super-skill learning

The exact Super books `3216..3226` use the existing `petBook` RPC but do not follow ordinary empty-slot learning.

- the Pet must own the matching Sơ, Trung, and Cao rows at level 3; lower levels do not qualify
- learning always grants the exact Super level-1 row, never the lowest prerequisite level or numerically lowest candidate
- no empty slot is needed: the lowest-numbered prerequisite slot becomes Super level 1 and the other two prerequisite slots become `0`
- owning any level of a Super chain blocks relearning that family's Sơ/Trung/Cao books
- Super items require `qLevel >= 3`; their exact `reqClass` is checked against creature `classId`, which is separate from both `classIds` and the family segment of `creKind`
- the 11 exact mappings are: `3216: 2830+3345+5156→5527`, `3217: 2749+3348+5159→5530`, `3218: 2743+3351+5162→5533`, `3219: 2821+3354+5165→5536`, `3220: 2699+3357+5168→5539`, `3221: 2842+3360+5171→5542`, `3222: 3329+3363+5174→5545`, `3223: 2746+3366+5177→5548`, `3224: 2748+3369+5180→5551`, `3225: 2711+3372+5183→5554`, and `3226: 2818+3375+5186→5557`
- successful persistence locks and revalidates the current 15-slot snapshot and book stack, then writes the Pet and consumes one book in one PostgreSQL transaction; active/cold/warm caches update only after commit
- after commit the handler emits the inventory callback, all three `onUpdatePet` slot changes in ascending order, and one property refresh; responder keys remain `f/rf/n`

The official 2013 guide's “lowest prerequisite level” rule is intentionally overridden by the operator contract above. Full evidence is recorded in `docs/research/2026-10-21_03_PET_SUPER_SKILL_LEARNING_RESEARCH.md`.

## Pet Skill Eligibility Rule

For book learning, `TBL_SKILL.cre_kind` uses the shape `family|minimumQLevel`.

- the first segment identifies the skill's source family and is not a Pet eligibility gate
- the second segment is the minimum Pet `qLevel`
- `petBook` rejects the book with responder failure `f = 2` only when the Pet quality is below that second segment
- runtime validation reads quality from enriched `pet.CreatureData.qLevel`, then falls back to `TBL_CREATURE.q_level`
- live evidence: Thỏ Baby 278 has `classIds=10,qLevel=3` and already owns Trí Tuệ Thiên Nhiên rows whose `creKind` first segment is `1`; it must also be allowed to learn Cổ Long (`6|...`) and Ngạnh Xác (`4|...`) books
- the checked-in catalog currently has 199 `type=506,useType=2` rows: 154 deterministically resolve through canonical `useEnv=0..2`, while 45 container/test/special/localized rows remain fail-closed. The full classification is in `docs/research/2026-10-21_02_PET_SKILL_BOOK_CATALOG_AUDIT.md`; do not turn fuzzy names into aliases without authoritative item→skill evidence.

## Pet Equipment Contract

`petEquipOn` and `petEquipOff` are callback-driven pet panel RPCs. They use dedicated pet-equip item slots, not the normal character equipment range.

- `petEquipOn(petId, itemId)` takes the pet id and the dragged pet equipment item instance id from the bag
- `petEquipOff(petId, itemId)` takes the pet id and the equipped pet item instance id currently attached to that pet
- valid pet equipment templates come from `TBL_EQUIPT_TEMPLATE` with `kind = 9` and `position = 50..57`
- pet gear validation should check `req_level`, `req_class`, and `req_class_id`
- the client uses `GamePredef.SLOT_SID_PET_EQUIP = [1000, 1200]`, so pet-equipped item instances must move into real SIDs `1001..1200`
- server persistence now uses:
  - item `slot_type = 6` for pet-equipped items
  - pet `property.equ1..equ8` for slot ownership
  - pet `property.equipGroup` for a stable pet-equip SID group
- `Pet.ToDTO()` and `Pet.ToSceneDTO()` must both expose root `equ1..equ8`, because the pet manager panel and battle HUD read those fields directly
- callback order matters:
  - `petEquipOn`: `onUpdatePet`, then `onMoveItem`, then `onRefreshPetProp`, then `onPetEquipOn`
  - `petEquipOff`: `onMoveItem`, then `onUpdatePet`, then `onRefreshPetProp`, then `onPetEquipOff`
- `onPetEquipOn` payload shape is `{pid, pos, oldSid, newSid}`
- `onPetEquipOff` payload shape is `{pid, pos, sid}`
- in those pet-specific payloads, `oldSid`, `newSid`, and `sid` are item instance ids, not slot ids
- `player.character_items` has a unique constraint on `(character_id, slot_type, slot_index)`, so replace-equip flows need a temporary slot hop before moving the new item into the pet slot
- deleting a pet should move equipped pet items back into the bag before deleting the pet row, otherwise those items become unreachable

### Pet equipment repair and modification (2026-10-24)

- `reqClass` is checked against creature `classId`/`ClassID`, not extended `classIds`; `reqClassId` continues to match the Pet template ID. Drag-and-drop and double-click share `petEquipOn`.
- Exact modification RPCs are `petEquLevelUp`, `petEquModColor`, `petEquModBind`, and intentionally lowercase `surepetEquModBind`. They remain logged until Flash smoke-test confirmation.
- Level upgrade consumes item `2333` by current required level (`40/60/80/100/120/140/160 → 5/10/15/20/25/50/50`) and silver equal to next template `reqLevel² × 10`.
- Color reroll uses item `2334` for display colors `0..2` with quantity/rate `1/80`, `2/50`, `4/30`; display color `3` uses item `2335 × 8` at `20%`. Silver is current template `reqLevel² × 10`; failed rolls still consume the cost.
- `petEquModColor` returns `num` as the quantity consumed, and Flash subtracts it in `onPetEquModColor`; do not also push an inventory stack callback for that RPC or the in-session stack is decremented twice. Level and bind responders instead carry the remaining stack.
- Bind reroll uses item `2809` by required level (`40/80/100/120/140/160 → 1/2/4/8/16/24`) plus `30,000` silver. The first RPC stores a per-character preview; confirmation alone persists the new bind values.
- Positions `56..57` use the checked special multipliers: level `×2` for pipe-wrapped `reqClassId`, otherwise rounded `×1.5`; color/bind `×4` or `×2` respectively.
- Mutations preserve item instance ID, pet-equip SID/slot, bind/star/custom properties. If the item is equipped, the owning Pet is re-enriched, recalculated and saved while preserving current HP/MP, then `onRefreshPetProp({id,s})` is pushed immediately.
- Template upgrades must also synchronize any persisted legacy identity properties (`tid`, `cid`, `giid`, `tplId`). `Item.ToDTO()` reasserts those four fields from canonical `Item.TemplateID` after merging properties, so old rows such as `template_id=1690` with `properties.tid=1682` cannot make Flash render or validate the obsolete template. This wire repair does not mutate existing player rows.

## Pet Bind And Skill Delete Contract

`bindedPetByPlayer` and `petDelSkill` are linked pet-panel RPCs.

- pet DTO `binded` must no longer be hardcoded; the server now resolves it from:
  - persisted `pet.property.binded`
  - fallback creature template `is_bind`
  - default `0`
- `bindedPetByPlayer(petId)` persists the lock state and emits `onUpdatePet(petId, "binded", 1)`
- `petDelSkill(petId, slotIndex, passwordHash)` uses the real skill slot index `1..15`
- `petDelSkill` requires the pet to be bound; when not bound, the server emits `onSystemSay("Phải khóa pet mới xóa được")`
- the delete cost item is `ItemConfig.ITEM_PET_SKILL_DEL = 1364`
- successful delete consumes one stack of that item and emits:
  - `onUpChaSlotStackNum(itemInstanceId, remainingStack, sid)` when the stack remains
  - `onDelCharactorSlot(itemInstanceId, sid)` when the stack is exhausted
  - `onUpdatePet(petId, "skillN", 0)`
  - `onRefreshPetProp`
- `petDelSkill` reuses the secondary password flow already used by protected pet RPCs and should cache the accepted hash with `setClientDP`
- clearing a pet skill must recompute passive pet skill bonuses immediately, because pet skill buffs are merged into final pet `final*` combat stats

## Pet Bag Item Summon Contract

Pet bag summon items are part of the normal item RPC flow, not a separate pet RPC.

- supported server entry point is `useItem(targetType, targetId, slotId)`
- supported pet-summon bag templates currently come from `data.data_tbl_item_template` with:
  - `use_type = 1`
  - `type = 508` or `550`
- the server resolves the summoned pet from normalized `item name + description`, not from `i1/i2`
- supported description patterns:
  - single pet: `Chứa pet <name>`
  - random one-of-many: descriptions containing `hoặc`
  - fixed-count multi-pet: `N pet <name>`
- unsupported for now:
  - choose-one descriptions such as `Có thể chọn nhận 1 trong 3`, `Chọn nhận 1 pet`, `Mở chọn nhận`
  - ambiguous generic bags like `pet hệ rồng`
  - direct `type = 509` pet keys
- successful item use now:
  - consumes the bag item stack as normal
  - emits the usual bag callback `onDelCharactorSlot` or `onAddCharactorSlot`
  - emits `onAddPet(petDto)` once for each contracted pet
- pet creation must go through batch contract logic with slot-capacity precheck so one bag cannot partially summon pets when the pet list is full
- the new pet batch path lives in `pet.Service.ContractPets`

## Pet Stat Bonus Channels (parity with character)

Base stat audit details and the reported Bạch Hổ case are recorded in `docs/research/2026-10-18_02_PET_STAT_FORMULA_AUDIT.md`. Creature-backed Pet stats use persisted aptitude, template `att* + level - 1`, and flat `GrowRate + GrowRateAdd`; template direct hit/dodge/speed/critical/combo/counter/defy/reborn/resiDefy values must survive recalculation.

The twelve `Kích Thích Tiềm Năng` passive rows 3744–3755 are a catalog-backed exception to ordinary `TBL_BUFF.prop1..6` loading: their prop slots are zero, while descriptions give exact quality-scaled values for Cân Bằng, Ngưng Tụ, Bão Tố, and Nhiệt Huyết. Apply those exact IDs against `GrowRate + GrowRateAdd`; do not infer formulas for other empty-prop buffs.

`Cân Bằng Chi Thược` IDs `3763/3771/3772` are a second reviewed empty-prop exception. Each pre-skill `finalEnergy` point contributes `0.05/0.10/0.15%` MagicAttack. Store the per-point factor in `SkillStatBonuses` and calculate it only after base Energy and MagicAttack are rebuilt, so repeated enrichment/recalculation is idempotent instead of recursively scaling the prior result.

Pet award quality uses two checked encodings. Values `15/19/20/25` are tenths and normalize to `1.5/1.9/2.0/2.5`; raw positive decimals below 10 remain unchanged. Do not auto-rewrite legacy player Pets with `15.000/19.000` without explicit data-remediation permission.

Pet `EquipmentStatBonuses` mirrors the character struct: `Flat`, `Percent`, `Float`. The aggregation pipeline routes raw equipment-template prop ids the same way characters do:

- raw 24 → `Percent[1]` (HP %) and raw 25 → `Percent[2]` (MP %), matching `character.EquipmentPropUsesPercent`
- raw 14, 26-32 → `Float` bonuses for special pet stats (Defy, ResiCritical, ResiDefy, EnhPhyHurt, EnhMagicHurt, BreakReborn, CriticalDamage, DebuffSuccRate)
- everything else → `Flat`, keyed by raw prop id (1=HP, 2=MP, 4=Atk, 5=MAtk, 6=Def, 7=MDef, 8=Hit, 9=Dodge, 11=Speed, 12=Combo, 13=Crit, 14=PraDef, 15-16=ReduceHurt1/2, 17-19=Resi*, 34=Reborn)

Application order inside `applyEquipmentBonuses`: flat → float → percent (percent applied LAST so it scales the flat additions, mirroring character's `applyPct`). Percent is supported only for HP/MP/Atk/MAtk/Def/MDef/Speed - the same canonical seven from `character.applyPercentBonus`.

Skill bonuses (`SkillStatBonuses`) still apply AFTER `applyEquipmentBonuses` via a separate pass. This means equipment % and skill % currently compound multiplicatively across the two passes (within each pass, multiple sources sum first per character formula). If/when full additive aggregation between equipment and skills is required, fold skill bonuses into the equipment accumulator before applying.

### Pet attribute key fix (2026-05-06)

Raw equipment-template attribute prop ids match the character canonical mapping in `character.EquipmentPropToStat`:

- `20 = Strength` (was correct)
- `21 = Stamina` (was incorrectly read as Agility)
- `22 = Intelligence` (was correct)
- `23 = Spirit/Energy` (was correct)
- there is **no raw template key for Agility from main equipment** - characters get Agility only from gems

`pet.RecalculateStats` now reads `attrBonus[21]` into the Stamina line; the Agility line takes only `AptAgilityEx` with no equipment contribution.

## Pet PVE Tower (PPVE) Contract

`challengeNextFloor` is a new inbound RPC (M8 Batch 2). It is fully registered and wired.

- Request body: `[null]`
- Server immediately pushes `changePPLoadingState(charId)` then returns empty `_result`.
- After bookkeeping, pushes `onSendReplayPPVEPanel(result, floorNum, replayId, rankList, myRank)`.
- Live wire shape from catalog_in.md: arg1=-1 (result), arg2=140 (floor), arg3="ts_cid" (replayId), arg4=rankList array, arg5=-1 (myRank).
- PPVE state is persisted via `FeaturePetPVE` (domain/statfeature constant added in M8) into the generic character feature state store.
- `PPVEService` lives at `internal/application/pet/ppve_service.go`.
- Handler lives at `internal/presentation/rtmp/handlers/activity/ppve.go`.
- State fields: `ppvefloor`, `todayFloor`, `kp`, `freeTime`, `goldTime`, `goldClgTime`, `awardTime`, `gold4awardTimeDaily`, `mlv`, `lastResetDay`.
- Daily reset: `freeTime` resets to 3, `todayFloor` resets to -1 on first challenge of a new calendar day.
- Max floor cap: `PPVEMaxFloor = 150`; no advance beyond that.

Open questions (combat-resolution stub):
- Full PPVE combat simulation against defending pets is NOT modeled. `challengeNextFloor` auto-advances floor by 1 on every challenge (always succeeds while free challenges remain).
- Live capture shows result arg1=-1 which client routes into `addPPLog`. The exact meaning of -1 (win/loss/replay code) is not confirmed.
- No kp reward on floor advance is implemented yet; kp management is deferred.
- `onUpdatePPVEPanel` and `GetPPVERank` still return partial/empty data — fixing those is a separate task.

## Pet fight lineup authority (2026-10-18)

- `getPetConf`, `getPetArenaConf`, `petFightConf`, and `petArenaTicket` remain the canonical Flash RPCs.
- Configurations are validated before persistence: owned unique Pets, positions 5–9, at most 24 commands, exact action enum, basic attack/defence IDs, learned skill ownership, available legacy skill level, and target codes.
- Farm, HMTXLS, and arena configurations no longer share one implicit `farm` record. Arena registration persists its own snapshot and battle lead selection reads that snapshot instead of trusting stale ranking `pet_id` state.
- Arena registration requires 2–5 Pets. Farm/HMTXLS setup permits 1–5 because those panels can be configured before a full arena team exists.
- The current arena simulator still resolves only the lead Pet; validation and snapshot authority do not claim full 2–5 Pet command-sequence combat.
- `getPetConf` has no mode argument, so HMTXLS can save an isolated configuration but cannot independently select that readback through the current Flash contract.
