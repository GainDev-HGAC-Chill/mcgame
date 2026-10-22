# Item & Inventory System

## Crafting panel safety and level-6 materials (2026-10-22)

`EquipFuncBag.showSaleItems()` does not null-check system-shop template lookups. `getShopConfig` now publishes only shop rows whose type-12/19/29 target exists in the loaded creature/equipment/item catalog and logs rejected slot IDs, preventing one stale runtime shop row from crashing both the material and jewel sale tabs. Checked-in shop data has no broken reference; this guard handles runtime/catalog drift without inventing a replacement item.

Material level 6 keeps canonical server `colorCode=5` but uses the narrow Flash compatibility alias `color=4`. Lower material tiers keep their original client color, preserving the level-5 rare-mix rate, while `autoInputMake()` no longer excludes level 6 via its `color < 5` condition. Level 6 remains the catalog-authorized maximum; material mixing rejects it without consumption rather than clamping a result back to level 6.

Change-prefix pending state now keeps and confirms the complete randomized preview property snapshot, including main/secondary stats, locked values and template-authorized soul fallback. Partial soul rolls below the current value are shown in the type-2 comparison payload but never persisted, matching the single “Không áp dụng” client state.

## Wish-Tree claim visibility and food buffs (2026-10-17)

Wish-Tree/lotto holding-bag claims persist rewards through `item.Service.AddItemWithBind`, but refreshing only `updateLottoBag` does not repaint the character inventory. `ClaimSingle` and `ClaimAll` now return the actual granted item records; the RTMP handler deduplicates stacked grants by instance ID and sends normalized `onAddCharactorSlot` DTOs before refreshing the relevant holding bag. New items and changed stacks therefore render immediately without `bagSort`.

Type-502 food previously only read `i1/i2`; the catalog's stat cakes have both fields zero and were silently consumed. The food handler now has an exact 33-item mapping to existing `TBL_BUFF` rows for Bánh Trôi, Bánh Ú, Bánh Nhân, Bánh Trung Thu, Bánh Thanh Minh, Christmas cakes and Táo Bình An, with 15-minute, 30-minute, one-hour or six-hour durations from their catalog descriptions/family. Use persists a timed `food_item` buff, emits long-buff callbacks and refreshes character stats. Unmapped type-502 entries with no HP/MP effect fail before consumption instead of returning false success.

## Summon, secret-passage, and social target items (2026-10-15)

The three summon-scroll templates are exact-ID type-500 effects, not generic message-only scrolls. Template 2514 grants timed long-buff 1709 for one hour, 2543 grants 1709 for 30 days, and advanced 2798 grants 1979 for 30 days. The handler preflights an owned locked Pet before consumption: standard books require orange `qLevel >= 3`, while the advanced book accepts purple/orange `qLevel >= 2`. It prefers an already active/following eligible Pet, otherwise the lowest Pet ID, persists follow state, then sends scene `onCancelPet(ownerCid)` followed by `onCreatePet({cid,petData})`. This buff is required because Flash `Core.createPlayer()` gates visible follow-Pet rendering on `SHOW_PET_BUFF=1709` or `SHOW_PET_BUFF2=1979`. No eligible Pet means no item consumption.

Templates 128 and 2137 are the exact Sách Dịch Chuyển Mật Đạo routes. Before consumption they reject battle and any group membership, matching item info. Success consumes one and enters Mật Đạo 1 (`map 140106`) at its safe point via the normal scene-leave/reset/enter transport sequence. Copy-map `t=-1` is intentional and bypasses the ordinary transport-scroll public-map gate.

`useIntimacyItem(targetName, quantity, templateID)` handles only 1302/1303/1315/32. Chocolate/Hoa Hồng grant 60 intimacy each; boxed Chocolate/large rose bouquet grant 660 each. Rose variants require opposite genders. The target must already be a friend, validation occurs before consumption, and both directional friend rows are updated when the reverse row exists. The owner stack callback uses a pre-consumption remaining-count snapshot, and online friend panels receive a confirmed `onInitViewImC` list refresh. Templates 724 and 783 are not double-click items: the real target flows remain `addPopNum(targetName,count)` and `delPopNum(targetName,count)`; generic `useItem` leaves them untouched and tells the player to use the target character's Hoa/Trứng Gà controls.

## Activity reward visibility and utility-bag movement (2026-10-15)

Activity reward handlers must not pass a Go `*item.Item` struct directly to Flash callbacks. `TakeNewGift`, `TakeSPGift`, and `BuySendCombine` now build the popup `onAddItem` notice separately and send a normalized `BuildClientItemDTO` map to `onAddCharactorSlot`; otherwise the item is persisted but the bag can remain visually stale until `bagSort` rebuilds slots. Shop purchases already use before/after stack snapshots and send every changed slot DTO, so that path remains unchanged.

Flash `addtItemFromBag(sourceSID, targetPosition)` and `addmxFromBag(sourceSID, targetPosition)` do not use the generic global SID contract for their second argument. They now have dedicated handlers: paged positions encoded as `page*100+position` are decoded using the client's 24-slot page size, normal utility-bag contents persist at 1-based indexes, and MX contents at 101-based indexes. Routing these RPCs to generic `MoveItem` interprets targets `1..22` as equipment slots and returns `cannot equip item`. Successful moves delete the regular-bag icon and refresh the complete `updateTempBags` payload. Each normal/MX `tempList` entry must use the compact Flash shape `{t: templateID, n: stackCount, b: 0|1, c: colorCode}`; a normal inventory DTO is persisted correctly but renders as an empty utility-bag slot because `TempBagSlot` reads those four short keys directly. Temp-bag sort must preserve the same 1/101-based indexing.

Removal has two Flash paths. Dropping a temp item onto a regular bag slot sends the actual wire RPC `addItemFromTemp(position, targetSID[, count])`; MX sends case-sensitive `addMXFromTemp(position, targetSID[, count])`. Runtime logging confirmed `addItemFromTemp` with args `[1,2106]`. These methods resolve the compact/paged temp position, honor the requested bag SID, support full or partial stack moves, push normalized `onAddCharactorSlot`, and refresh `updateTempBags`. The older panel calls `tempToBag(position)`/`mxtempToBag(position)` remain supported as first-empty-slot aliases. None of these methods may route through generic SID parsing for the temp position.

Latest voucher1132 follow-up deployed17:34:09 (2026-10-12), game running/restart_count0; scoped tests/vet/build PASS. Flash validation NOT TESTED; crafting sale-bag null remains unresolved, not covered by voucher or quest-sort repairs. No item/player repair or reward replay. [Batch results](../research/2026-10-12_09_PANEL_PROTOCOL_ECONOMY_TRIAGE_RESEARCH.md).

## Gift currency vouchers (2026-10-14)

Direct use now handles only template `654` (`Ngân Phiếu Lễ`), `1064` (`Hoàng Kim Lễ`) and `3848` (`Thẻ Kim Phiếu`) before the generic type-550 path. Template 654 grants 10,000 `moneyBind`; template 1064 grants the described 50 `goldBind`; template 3848 securely selects one described `goldBind` amount from 88/188/288/588/888/1888 for each card. Single and multi-use enforce the checked-in template type/useType, required level, character target, count and overflow, consume the stack, persist the character wallet and emit one `onAddMoney` callback with delta/total. Wallet persistence failure after item consumption returns an explicit non-replayable error before success callbacks. No other Hoàng Kim Lễ variant is inferred or changed. Application and item-handler tests PASS; Flash/deploy verification pending.

Magic-crystal award items use game-currency IDs rather than wallet silver: `5947→75×200`, `5948→73×20`, `5949→74×5`, and `5993→75×10`. Award-backed multi-use aggregates the total currency change before callbacks.

## Mảnh Pasar direct-use guard (2026-10-14)

Item template `5351` (`Mảnh Pasar`, type `550`, `useType=1`) is a collection fragment, not a directly usable reward bag. `useItem` now treats this exact template as a silent no-op before quest, book, and generic effect dispatch. Double-clicking it returns no connection/internal error and does not consume or mutate the stack. No other type-550 item is affected.

## Silver voucher 1132 (2026-10-12)

Actual live useItem1132 returned cannot-use-item: type550 has no self-opening award definition. Exact Bạch Ngân Lễ (10 Vạn) implementation grants100000 silver per consumed instance; unbound→money, bound→moneyBind, based on explicit template description (the shorter info text inconsistently says kim phiếu and is not used to credit gold). Template gold100 is a shop price, not payout. Single and multi-use validate template/use target/level/count/overflow, consume before changing character balance, and send one onAddMoney(delta,total) plus correct remaining-stack/SID updates. Item-write failures leave wallet/book unchanged. Later character-write failure remains nontransactional, returns an error without success credit callbacks; never auto-replay. No old voucher use or wallet history was repaired; only1132 is supported by this rule, not inferred million vouchers or unrelated scrolls. Item app/handler tests PASS; Flash retest/deploy pending.

## Expansion book effects (2026-10-12)

Root cause: templates **324** (Sách Mở Rộng Kho), **325** (Sách Mở Rộng Túi Đồ), **326** (Sách Mở Rộng Túi Đựng Pet) are type500/useType1; `scrollEffectHandler` previously returned a message and consumed one without changing capacity. `capacity.go` now intercepts only these exact IDs in `UseItem`/`UseMultiItem`, before generic scroll dispatch. No new catalog IDs, awards, prices or database changes.

- 324 grants **one bank tab = 30 slots**, max **5 tabs / 150 slots**; 325 grants **one bag tab = 30 slots**, max **7 tabs / 210 slots** (six additional tabs beyond the initial tab). Updates both `BankSlotNum`/`BankSlots` or `BagSlotNum`/`BagSlots`.
- 326 grants **one carried-pet slot**, synchronizing `PetMaxNum`/`PetSlots`, max **180** per the existing pet domain/UI contract, not the obsolete catalog description's84. This is not the pet bank or pet-equipment item bag. See [Pet Slot Opening Contract](pet-system.md#pet-slot-opening-contract). No gold/goldBind is charged: paid `petOpenSlot` is a separate flow.
- Character-target only (targetType1; absent application context defaults to character). Requires the actual type500/useType1 template and its level requirement. Cap/overflow/invalid count/insufficient stack/ownership/target failures do not consume or alter capacity. Legacy slot-only fields are read when their count aliases are absent. Single use consumes exactly one, regardless of `UseContext.Quantity`; multi-use preflights the whole count and rejects overflow without partial expansion. Character changes happen only after successful item deletion/stack update.
- Existing handler character-update persistence is reused; PostgreSQL `CharacterRepository.Create/Update` already stores `BagSlotNum`, `BankSlotNum`, `PetMaxNum`. Handlers immediately send the remaining stack (or original-SID deletion when exhausted), then owner-only `onUPP({bankSlotNum|bagSlotNum|petMaxNum: newCount})`. Multi-use also refreshes a remaining book stack and uses actual SID rather than instanceID for deletion. Item consumption and character persistence remain separate existing repository operations, not a newly atomic transaction.
- Persistence failure distinction: item-repository deletion/stack-write errors prevent capacity changes. **Character persistence happens after consumption and is nontransactional: a later `charService.Update` error does NOT guarantee preservation of the consumed books.** For `capacity_updates` only, both use handlers now return a wrapped, explicit persistence error before success/stack/capacity callbacks, without replaying consumption or attempting unsafe rollback writes. Unrelated item-use error behavior is unchanged. Detached-repository regressions cover all three books in single/multi use, one character-write attempt, consumed count remaining consumed, no success callbacks and unchanged persisted capacity; separate regressions preserve generic use behavior.
- Evidence: `docs/database/game_data/TBL_ITEM_TEMPLATE.json:7518–7627`; read-only Flash `GamePredef.as:1164–1165` supplies bag/bank boundaries, `Player.as:1711/1957` computes tabs×30, `BankPanel.as:1834` gates five tabs; pet UI limits are `MAX_PET_TAB=10`/`MAX_PET_PER_PAGE=18` plus the existing authoritative pet memory. `CallBack.as`/`BagPanel.as` decompilation is aborted, so this pass does not claim a newly recovered full callback body.

Regression tests cover all three IDs, exact single consumption, stack exhaustion, final tab/slot, pet84→85, legacy aliases, no currency/other-state changes, level/ownership/target/cap rejection, missing/mismatched templates, delete/update failures, multi-count preflight, unrelated scroll isolation, persisted handler state, immediate callbacks and original SID. Item application/handler suites and vet PASS. No live item use/player mutation, DB writes, deployment or commit; real Flash retest pending.

## Actual useItem auto-equip snapshot fix (2026-10-12)

Operator retest: equip still invisible, unequip visible. Runtime identified actual equipment action as useItem, not explicit equipOn. Application UseItem returned the pre-equip detached bag record; now reloads final equipped record. Handler uses the same original-delete → final-instance-update → final-slot-add owner-only sequence for auto-equip, including displacement and HP/MP/stat/appearance updates. Detached app/handler regressions PASS; Flash retest pending. This supersedes any assumption the previous explicit-equip patch covered all wearing routes.

setTempBag(2112,80) registered as **unavailable**, no item consumption/capacity grant or updateTempBags. Correct installation requires persisted descriptor/expiry/replacement support; current empty tempBag payload is not enough. Actual operation remains NOT IMPLEMENTED. [Research](../research/2026-10-12_03_AUTO_EQUIP_SHOP_QUEST_MISSING_RPC_RESEARCH.md).

## Equipment icon refresh / shop pet growth (2026-10-12)

Operator confirmed bag-sort fix and prior panel errors resolved in real use. EquipOn swap had wrong sourceSID0 for auto slot-1 and unsafe incoming-before-displaced sequencing. Now `equip_slot_refresh.go` deletes all old positions first, sends final instance DTO then final slot DTO, preserving IDs and stats/appearance flows; EquipOff uses actual final bag SID. Empty-slot original move callback was valid, so do not assert the swap bug explains every blank icon. Tests for empty/replaced accessory, two-ring selection/replacement and two unequip modes PASS; real new equip flow NOT TESTED.

Pet-description fallback now honors growth1.5 via scaledquality15 rather than default grow_base0.8. Accent/ascii/comma/newline/bare1.5 parser tests PASS; higher stated quality preserved. Award rows use rawquality1.5. No existing pets repaired. Added13 exact-description shop awards (10 grade4 materials×12, bag1616 item274×10, Camy3254 and DêConII4840 pets1.5), idempotent installer/exported seed; total67 includes preserved54 starter rows. Full325-candidate audit distinguishes no award rows from unsupported flow; unknown odds/class/opaque contents are not fabricated. [Research](../research/2026-10-12_01_EQUIP_SHOP_GM_RESEARCH.md).

Exhaustive snapshot report completed:25 supported pet candidates +12 partial,70 deterministic missing (13 now configured,57 pending target/semantics),32 blocked,112 ambiguous,74 non-reward false positives.48 all-hidden listings. Counts are pre-install static classification, not gameplay/availability. Every325 IDs/listing states in [audit](../research/2026-10-12_02_SHOP_BAG_AUDIT_RESEARCH.md). Common Vải material is liveID25, not jewel12.

## Bag sort display recovery (2026-10-11)

Real bagSort sorted11 records but client continued dragging oldSID2116, yielding item not found; no item loss established. Previous onItemSort SID mapping was already protocol-correct; exact refresh failure is unconfirmed. `bag_sort_refresh.go` now sends all captured pre-sort deletions **before** adding every complete final DTO, then onItemSort([]) for source-free repaint. Sort/merge business rules unchanged; callback errors propagate. Isolated quantity/binding/sort-then-move and callback ordering tests PASS, affected suites/vet/build PASS. Existing client-only stale records may require relogin. Real gameplay NOT TESTED; see [research](../research/2026-10-11_10_BAG_SORT_EVENT_PANELS_RESEARCH.md).

## Local deployment: starter gift bag diagnosis (2026-10-11)

Real RPC at 22:07:10: Gain (character1) calls `useItem(1,-1,2)` for instance2/template2918, **Túi Quà Tân Thủ (cấp 1)**. Type550/useType1 routes to questItemEffectHandler; there are no award rows, no choose-pet description and no pet fallback, so it returns ErrCannotUseItem before consumption. Error mapping exposes `Internal error: cannot use item`. This occurred before the later disk relocation, not caused by the FPS target or moving Docker storage.

Read-only project Supabase check after relocation: `data.data_tbl_item_award` has **0 total rows**, **0 for item2918**; `player.get_character_item_by_id(2)` still returns character1/template2918/stack_count1, with original timestamps. Do not replay quest rewards, remove the item or synthesize an arbitrary starter package. Opening the bag is **BLOCKED** until authoritative contents are recovered or the operator explicitly approves a local reward definition. Existing award handler can process configured rewards; deploying a generic success reply is not a fix.

### Approved reward chain replaces missing-config blocker

The operator subsequently provided and approved the level1–50 chain, purple Trác Việt jewelry and pet91/growRate1.9. Installed54 guaranteed rows for bags2918–2923 (9/9/9/11/8/8); rerun validated idempotent. Source installer: `supabase/data/starter_gift_awards.sql`; exported seed: `supabase/seeds/data_tbl_item_award.sql`. No player rewards were replayed or bags auto-opened. Equipment quality20/prefix5; material16 Cấp4 uses payload.materialColorCode3, handled consistently in capacity preflight/granting by `award_material_color.go`. Other award behavior is unchanged. Isolated full-package/level-gate tests and item suites/vet/build PASS; real Flash opening remains NOT TESTED. See [research](../research/2026-10-11_09_STARTER_GIFT_AWARDS_RESEARCH.md).

Redeploy completed: startup22:51:18 confirms itemAwards54 and RTMP started; infra check PASS. `.dockerignore` now excludes `.local/` so relocated Docker disk/backups/cache are never copied into images. No PostgreSQL/Redis redeploy/reset.

## Item-Use Target Gate

The Flash item-use contract uses `useType` only as a target gate (`1` character, `2` pet, `3` either, `4` not directly usable) before branching by item `type` and special item IDs. The choose-one package panel contract is `useItem` -> server callback `itemMultitoSel({packItemList:{idx:{ti,ii,n,q}}, packGetNum}, sid)` -> client RPC `selMultiItemByIdx(packItemSid, selectedIdx)`; the responder returns `0` to close the panel or a positive remaining count to keep it open. Current server support matches this choose-one flow, but generic item use still misses full server-side `useType` target enforcement, general pet-target consumable routing, paid package `usePackageByMoney` / `onUsePackageByMoney`, and `onItemUseCallBack` confirmation wiring. See `docs/research/2026-04-25_02_ITEM_USE_PANEL_CONTRACT_RESEARCH.md`.

Configured selectable packages now reuse `data.data_tbl_item_award` with reserved metadata `payload.selection={order,getNum}`. Runtime accepts the mode only when every row has valid metadata, orders are exactly `0..N-1`, `getNum` is consistent and bounded, and every reward type is Pet (`12`), equipment (`19`) or item (`29`); mixed/malformed definitions fail closed before the weighted picker. Opening does not consume the package, multi-use is rejected, and selection reuses ordinary award capacity/quality/prefix/material grant logic. Pending selections are serialized per character, revalidate package ownership/template/stack/index, reject duplicate choices and consume only after the final grant. Final reward callbacks are delayed until consume succeeds and precede the package slot callback. A consume failure can still leave an already-persisted reward without consuming the package—the documented legacy cross-domain atomicity risk—but emits no false success callback and drops the pending snapshot to prevent replay. The 3376 blue quality/color override remains only on its legacy description/hardcoded fallback; configured rows control their own visible quality.

## Core Data Tables

TBL_CHARACTOR=2, TBL_CHARACTOR_SLOT=9, TBL_CREATURE=12, TBL_EQUIPT_INSTANCE=18, TBL_EQUIPT_TEMPLATE=19, TBL_ITEM_INSTANCE=28, TBL_ITEM_TEMPLATE=29, TBL_MAP=33, TBL_NPC=35, TBL_PET=39, TBL_QUEST=45, TBL_SKILL=52, TBL_BUFF=66.

Slot object: `{id, sid, type, itemId, stackNum, tid}`.

Client item category constants used by the Flash client:

- Kinds: `1=ITEM_KIND_MAINHAND`, `2=ITEM_KIND_SUBHAND`, `3=ITEM_KIND_DEFENCE`, `4=ITEM_KIND_JEWELRY`, `5=ITEM_KIND_ITEM`, `6=ITEM_KIND_MATERIAL`, `7=ITEM_KIND_PET`, `8=ITEM_KIND_MAGICWEAPON`, `9=ITEM_KIND_PETEQU`, `10=ITEM_KIND_FLYER`, `11=ITEM_KIND_DRESS`, `12=ITEM_KIND_GATHER`, `13=ITEM_KIND_WING`, `14=ITEM_KIND_FEATHER`.
- Kind `1` / main hand: `100=HAMMER`, `101=STICK`, `102=GUN`, `103=SWORDONE`, `104=GUITAR`, `105=BAT`.
- Kind `2` / sub hand: `200=SHIELD`, `201=BOOK`, `202=CUFF`, `203=KNIFE`, `204=PICK`, `205=GLOVE`.
- Kind `3` / defence: `300=HAT`, `301=CLOTHES`, `302=TROUSERS`, `303=BELT`, `304=SHOE`, `305=SHOULDER`.
- Kind `4` / jewelry: `400=NECKLACE`, `401=RING`, `402=JEWELRY1`, `403=JEWELRY2`.
- Kind `5` / item: `500=REEL`, `501=MEDICINE`, `502=FOOD`, `503=JEWEL`, `504=STAR`, `505=SKILLBOOK`, `506=CREBOOK`, `507=PETFUNC`, `508=QUEST`, `509=KEY`, `510=MW_REPAIR`, `511=MW_SKILL_RESET`, `512=PETEQU_LEVELUP`, `513=PETEQU_MODCOLOR`, `514=MW_TRANS`, `515=FISHING_TOOL`, `516=FORMULA`, `517=TEMP_BAG`, `518=WING_ENHANCE`, `521=FAIRY_SKILL_ITEM`. Extra client-only item-related constants: `519=MW_PROP_RESET`, `520=TARGET_ITEM`, `522=MW_STAGE_EIGHT`, `523=SUBLIME`, `524=RESTRAIN`, `527=QILINGSTORE`, `551=STAR_ADD`, `552=STAR_SPEED`.
- Kind `6` / material: `600=DIAMOND`, `601=METAL`, `602=WOOD`, `603=JADE`, `604=CLOTH`, `605=FUR`, `610=FISH`, `611=PLANT`, `612=HERB`.
- Kind `7` / pet: `700=PET_HUMAN`, `701=PET_MONSTER`, `702=PET_PLANT`, `703=PET_MACHINE`, `704=PET_DEVIL`, `705=PET_DRAGON`.
- Kind `8` / magic weapon: `800=MAIN_MAGICWEAPON`, `801=SUB_MAGICWEAPON`.
- Kind `9` / pet equip: `900=PETEQU_SPUR`, `901=PETEQU_NECK`, `902=PETEQU_BELL`, `903=PETEQU_WING`, `904=PETEQU_ARMOR`, `905=PETEQU_CUFF`, `906=PETEQU_DRAGON1`, `907=PETEQU_DRAGON2`.
- Kind `10` / flyer: `1000=FLYER_SUBTYPE`.
- Kind `11` / dress: `1100=DRESS_SUBTYPE`.
- Kind `12` / gather: `1200=GATHER_GLOVE`, `1201=GATHER_ROD`.
- Kind `13` / wing: `1300=WING`.
- Kind `14` / feather: `1400=FEATHER_MIX`, `1401=FEATHER_A`, `1402=FEATHER_B`, `1403=FEATHER_C`, `1404=FEATHER_D`.

## Item Instance Field Mapping

Key mappings (Go -> Flash client):

- `TemplateID` -> `tid`, `cid`, `giid`, `tplId`
- `ID` -> `id`, `itemId`
- `StackCount` -> `stackNum`, `stackCount`
- `IsBound` -> `binded` (NOT "isBound")
- `EnchantLevel` -> `enchantLv`
- `StarLevel` -> `starLv`, `upgradeNum` (client tooltip reads this for star display)
- `ColorCode` -> `color`, `colorCode` (instance overrides template)
- `Durability` -> `endure` (NOT "durability")
- `MaxDurability` -> `maxEndure`
- `SlotType` -> `slotType`, `SlotIndex` -> `slotIndex`
- `CalculateSID()` -> `sid` (computed global slot ID)
- `ItemType` -> `itemType`, `type` (table type 28 or 18)

Item Template fields all use camelCase JSON tags: `itemLevel`, `color`, `colorCode`, `iconCode`, `reqLevel`, `reqClass`, `bindType`, `propType`, `proplNum`, `useType`, etc.

Combined DTO structure: `{"type": 28, "inst": {...}, "temp": {...}}`.

Field priority: Instance `color` overrides template `color`. Instance `binded` for binding checks. Template `itemLevel` for display.

Missing fields fix: `f` (field data JSON string, `"{}"` when empty), `t` (type value from template, defaults to `-1`).

## SID Boundaries

`CalculateSID`/`ParseSID`: Bag=2101..2310, Quest bag=2311..2340, Pet item bag=2341..2370.

## Item RPC Argument Hardening

Flexible integer parser for mixed types (`float`, `int`, `int64`, unsigned ints, numeric strings). Applied to `UseItem`, `DropItem`, `EquipOn`.

## Inventory Bag Fix

Warm inventory logic aligned with slot enums. Slot capacities from domain constants (`DefaultBagSlots`, `DefaultQuestBagSlots`, `DefaultPetItemBagSlots`, `EquipSlotMax`).

## Equipment Property Name Mapping

Equipment properties (numeric ID -> Vietnamese name): 1=Gioi han HP, 2=Gioi han MP, 3=Nang luong, 4=Cong vat ly, 5=Cong ma phap, 6=Phong vat ly, 7=Phong ma phap, 8=Chinh xac, 9=Ne tranh, 10=Phan kich, 11=Toc do, 12=Lien kich, 13=Bao kich, 14=Xuyen phong ngu, 20=Luc (STR), 21=Nhan (AGI), 22=Tri (INT), 23=Than (SPI), 24=Gioi han HP (%), 25=Gioi han MP (%), 27=Khang bao kich, 28=Khang debuff, 29=Khang xuyen.

Active soul properties: 1=Cong vat ly, 2=Cong ma phap, 3=Bao kich, 4=Xuyen phong ngu, 5=Giam sat thuong vat ly, 6=Giam sat thuong ma phap, 7=Giam sat thuong, 8=Tang HP.

## Equipment Flag Fields

`flag`: For standard/pet equipment: `sublimeId`, `sublimeElement`, `sublimeAdd`. For spirit weapons (slots 15-20): `propType` and `propVal`. Example: `{sublimeElement:1,sublimeAdd:0,sublimeId:50}`.

`flag2`: Ky Linh (spirit companion) data. 3 lines (indexes 0-2), each with `t` (property code), `v` (current value), `max` (maximum cap). Example: `{0:{t:60,v:0.025,max:0.025},1:{t:5,v:2500,max:2500},2:{t:59,v:0.011,max:0.025}}`.

`flag3`: Pet equipment stone slots. Keys `1..6` represent pet stone slots. Example: `{1:[1,1,12345],2:[1],3:[1]}`.

## Material Mixing

RPCs: `materialMixOne(num, itemID, tempBagFlag)` and `materialMixAll(num, itemID, tempBagFlag)`. `num` = quantity consumed per attempt (2-5).

Quality system (ColorCode): 0=White, 1=Green, 2=Blue, 3=Purple, 4=Orange, 5=Red (max). Client display: `color = Math.ceil(quality / 5)`.

Success rate: Normal materials (colorCode != 4): `num * 20%`. Rare materials (colorCode == 4, orange): `num * 6%`.

Response format: `{"slotId": <id>, "num": <remaining stack>, "flag": <success bool>, "finalNum": <quantity produced>}`.

## Equipment Quality & Prefix System

Equipment items have a `ColorCode` (0-5) determining display color and Vietnamese quality prefix (`preNameType`).

ColorCode to Prefix & Color: 0=(none), 1=Binh Thuong/White, 2=Cuong Hoa/Green, 3=Tinh Xao/Blue, 4=Hoan My/Purple, 5=Trac Viet/Orange-Yellow.

Quality -> ColorCode: `(quality + 4) / 5` clamped to [0, 5]. Quality 1-5->1, 6-10->2, 11-15->3, 16-20->4, 21-25->5.
Quality -> PrefixType: `quality % 5` (where 0 becomes 5), cycles 1-5 within each tier.
ColorCode -> Representative Quality: `((colorCode - 1) * 6) + 1`.

Equipment creation now derives `preNameType` from `ColorCode` instead of hardcoding `0`. Falls back to template color data when caller doesn't provide color code.

## Equipment Active List Sync

`equipActiveList` built from currently worn equipment. Key = `sid`, value = `true` when template's `activeEquipId > 0` AND required template currently equipped; otherwise `false`.

Emitted at: `onChooseCharactor` (login), `onEquipActiveList` in `equipOn`, `equipOff`, and auto-equip flow of `useItem`.

## Check Equip Edit Compatibility

Registered no-op handler for `checkEquipEdit` RPC. Client probes this when opening equipment/wing panels. Returns no pending state. If server later persists interrupted equipment-edit workflows, should load saved object and trigger `onGetEquipEditObj`.

## Equipment Display Contract

Implemented a shared application-layer equipment display contract in `internal/application/item/display_contract.go`.

Primary entrypoints:

- `BuildClientItemDTO(it *domainitem.Item)`
- `BuildClientItemDTOList(items []*domainitem.Item)`
- `ApplyEquipmentDisplayContract(recordData, templateData)`

Purpose: keep all runtime equipment payloads aligned with the Flash client tooltip contract used by `TipEquip`, `EquipFunc`, inventory callbacks, `equiptList`, and gamedata hydration.

Centralized behavior now includes:

- color-tier fallback resolution in this order: instance `colorCode` -> instance `q` / `quality` -> instance display `color` -> template `colorCode` -> template `color`
- display alias normalization for `color`, `colorCode`, and `preNameType`
- bind normalization to client string flags `binded = "0" | "1"`
- template fallback injection for `mainProp1/2`, `mainPropNum1/2`, `prop1/2`, `propNum1/2`, `activeProp`, `activePropNum`, `bindMainPropNum1/2`, `holeNum`, `endureMax`
- default compatibility fields for `element`, `maker`, `flag`, `flag2`, `flag3`, `t`, and `t1..t10`
- durability alias population for `endureLeft` and `endureMax`

Current runtime coverage:

- item RTMP handlers now route equipment callbacks through the shared builder instead of using raw `ToDTO()` for bag, equip, temp bag, craft, prefix-change, jewel, maker, mix, bind, and use-item flows
- `sendInventoryUpdate` and `getInitSlot` use the shared builder
- appearance sync now uses the shared builder for `equiptList`
- gamedata normalization delegates to `ApplyEquipmentDisplayContract` instead of maintaining separate handler-local color/prefix logic

This was intentionally implemented as application-layer composition, not repository formatting, so domain rules stay in `internal/domain/item/display.go` while presentation only sends normalized payloads.

Focused tests cover:

- equipment DTO normalization from template fallbacks
- quality-driven fallback for `colorCode`, display `color`, and `preNameType`
- default compatibility values for `flag` and `t1..t10`
- durability alias output and bind normalization

Quest reward equipment now randomizes core prop-line assignment from the template stat pool (`mainProp1/2`, `prop1/2`) before rolling the numeric values from the client `EQUIPT_QUALITY[q-1..q]` range. This keeps reward equipment template-derived while no longer forcing a fixed prop layout.

Crafted equipment produced by `newMake` now uses the same template-pool prop shuffle and multiplier-based value rolls when a preview-selected quality is applied. The plain `UpdateEquipmentQuality` path remains non-rerolling. Change-prefix uses its dedicated preview-snapshot apply path so confirmation persists exactly the values shown to the player.

## Item Handler RPCs (40+ methods)

Equipment: `equipOn`, `equipOff`, `getInitSlot`, `isEquSid`. Inventory: `moveItem`, `moveItemNum`, `dropItem`, `bagSort`, `moveItemToPage`, `itemToTarget`. Item Use: `useItem`, `useItemGold`, `useMultiItem`, `useTransport`, `useTransportGroup`. Recovery: `fullHpRecoverByItem`, `fullMpRecoverByItem`. Repairs: `repair`, `repairAll`. Appearance: `setDressHide`, `checkEquipEdit`, `checkLimitWingEffect`, `checkSameDay`. Temp Bags: `getTBag`, `openTempSlot`, `openmxTempSlot`, `mxBagSort`, `tempBagSort`, `oneKeyOpenAllMXTemp`. Equipment Modifications: `changePrefix`, `sureChangePrefix`, `changeSoul`, `sureChangeSoul`, `changeElement`, `sureChangeElement`, `changeBind`, `sureChangeBind`, `changeLevel`. Equipment Operations: `equResolve`, `starOne`, `starAll`, `getStarNum`. Sockets: `getHoleNum`, `holeDig`, `getJewelData`, `jewelSet`, `jewelDel`, `jewelUpdateOne`, `jewelUpdateAll`. Crafting: `newMake`, `getMakeColor`, `materialMixOne`, `materialMixAll`. Item Management: `bindItem`, `changeName`. Social: `useSeek`, `useTrack`, `useFootle`, `useAntiFootle`, `useSnowBall`, `useHalloweenCard`, `useFestFootle`, `useRadar`.

### Transport Scrolls (Track A)

`useTransport(mapID)` consumes a regular transport scroll (priority: 1036, 1363, 2946, 1515, then senior IDs as fallback) and teleports the caller. `useTransportGroup(mapID)` validates group leadership via `groupService.IsGroupLeader`; non-leaders are charged the regular cost. `useItemGold({tid, mid})` is the client's "I have no scroll, pay gold" path — server still tries the matching item priority list defensively before charging gold. All three share `chargeAndTransport` -> `performTransport` in `internal/presentation/rtmp/handlers/item/transport.go`. If the player has no item, gold is deducted from `Money` (when `SelectedMoneyType == 2`) or `MoneyBind` (otherwise) — costs: regular `transportRegularGoldCost = 1000`, senior/group `transportSeniorGoldCost = 5000`. When neither item nor gold is available, the client receives `onSystemMidMsgOrNote` and the teleport is aborted. Item path emits `onDelCharactorSlot`/`onUpChaSlotStackNum`; gold path emits `onUPP` with the updated money field. Map validation rejects `mapID` with `MapTemplate.T <= 0`.

### Social Prank Pattern (Track C)

`useFootle`/`useAntiFootle`/`useSnowBall`/`useHalloweenCard`/`useFestFootle` all share `applySocialPrank(ctx, args, rpcName, templateIDs, buildMessage)` in `friend_items.go`. The helper resolves the target by name, validates same-channel + same-map, consumes one item from `templateIDs`, sends `onDelCharactorSlot`/`onUpChaSlotStackNum`, and broadcasts `onExcludeRedMsg` to the scene. `buildMessage(src, target *Character) string` builds the per-item message body.

### Radar / NPC Locator (Track B)

`useRadar(npcName)` looks up NPCs via `gameData.GetNPCsByName` (filters `Fd > 0`), prefers same-map candidates over the first match, consumes one of `[1397, 1484, 1615]`, and replies with `onNpcPos([{name, posMapId, posX, posY}])`. Returns `onNpcPos(nil)` without consuming when no NPC matches.

### Item Type Stub Coverage

`internal/application/item/service.go` `phase4Stubs` covers types 510, 511, 514, 515, 517, 518, 519, 521, 522, 523, 524, 525, 527, 551, 552 — these emit `"loại vật phẩm <name> chưa được hỗ trợ"` and do **not** consume the item. Types 514, 522, 525 were added in this pass to prevent silent item destruction when the type has no real handler.

## Potion Healing System

Healing potion templates have `I1=I2=0` in the database. Heal amounts are hardcoded server-side in `internal/application/item/potion.go` to match the Flash client (`PortraitCanvas.as`). HP potions: template IDs 1, 2, 4, 110, 113, 321, 371, 1975 (300-9000 HP). MP potions: template IDs 158, 125, 126, 127, 322, 370, 1974 (300-7000 MP). The `applyItemEffect` function uses `GetPotionHealAmount()` to resolve heal amounts when I1/I2 are zero. The `fullHpRecoverByItem`/`fullMpRecoverByItem` RPCs consume multiple items from lowest to highest heal amount until HP/MP is full. The client sends item arrays with `{sid: itemDatabaseID, priority: healAmount - boundStatus}`.

## UseItem Effect Registry (Hybrid Architecture)

Item use effects are dispatched through a two-tier system in `internal/application/item/`. Each handler implements `CanHandle(tpl)` and `Apply(ctx, uctx, char, item, tpl)` returning an `ItemEffectResult`. The `UseContext` carries RPC parameters (targetType, targetID, petID, quantity).

Dispatch tiers:

1. **Type registry** (`typeHandlers[tpl.Type]`) — deterministic lookup by item template Type field
2. **Award table** (`awardEffectHandler`) — items with `data_tbl_item_award` entries
3. **Fallback chain** — `boxEffectHandler` (UseType 1), `walletEffectHandler`, `buffEffectHandler`, `defaultEffectHandler`

### ItemTemplate `use_type` semantics

The Flash client uses `useType` first as a target-eligibility gate, not as the full item-effect dispatcher:

- character target accepts only `useType = 1` or `3`
- pet target accepts only `useType = 2` or `3`
- actual behavior after that still branches mostly by template `type` and a few special item IDs / RPC paths

High-confidence meanings:

- `1` = character-target usable
- `2` = pet-target usable
- `3` = usable on either character or pet
- `4` = not directly usable / material-style item

Observed examples only, not the authoritative meaning:

- `useType = 1` commonly appears on skill books, bags, gift boxes, and other direct character-use items
- `useType = 2` commonly appears on pet buffs and pet skill-book style items
- `useType = 3` commonly appears on consumables that can target either the player or a pet, including many healing/battle items

Server-side rule of thumb: treat `useType` as a coarse access gate, then let the shared UseItem registry decide the real effect from `type`, award rows, and any special-case handlers. Do not treat older documentation examples of `use_type = 10` as confirmed live client behavior.

Type registry (27 types mapped):

- 500=scroll, 501=potion, 502=food, 503=gem socket, 504=star stone, 505=skill book, 506=creature book
- 507=pet func, 508/509/550=quest item (delegates to award→giftBox→petBag), 512/513=pet equip
- 516=formula, 520=target item
- Phase 4 stubs (510,511,515,517,518,519,521,523,524,527,551,552): return "chưa được hỗ trợ"

Award table types: 12=pet, 19=equipment, 29=item, 30=currency (silver/gold/honor), 31=exp, 32=buff.

Gift box flow: `useItem` -> `itemMultitoSel` callback -> client opens MultiItemPanel -> user selects -> `selMultiItemByIdx` RPC -> server grants reward -> returns remaining count (0 = done).

Special item RPCs: `changeNameByCard`, `changeCharSex`, `useGoodCard`, `useWeddingBag`, `useRedBag`, `useIntimacyItem`.

Temp bag rewards: effect handlers can set `TempBagItems` in result, dispatched as `warnTemporaryBag` callback.

Research: `docs/research/2026-04-08_01_USEITEM_HYBRID_ARCHITECTURE_RESEARCH.md`
