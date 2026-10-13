# Shop bag audit — 2026-10-12

## Scope and result

Read-only audit of **every one of the 325 distinct candidates** in `.local/research/shop-bag-audit-20261012.json`, against the current Go item-effect dispatch. No database requests, seed/configuration changes, item grants, or gameplay tests were performed. This is exhaustive for the supplied snapshot, not a claim that its candidate query encompasses every item in the game.

All 325 have `awards=null`, `i1=0`, and `i2=0`. Types: 550 ×283, 509 ×21, 508 ×19, 500 ×2. The **54 approved starter awards are for other bags**, not these candidates. Absence of awards is **not** proof that a pet bag cannot open: description-based pet fallback exists.

**Post-audit deployment update:** the snapshot predates the parent installing13 exact-description awards for1455/1456/1457/1458/1459/1469/1470/1472/1473/1474/1616/3254/4840. Database verified total67 (54 starter +13 shop), server startup00:23:12 loaded67. The classifications/counts below describe the original snapshot, not current missing-row totals: the original70 D candidates now contain13 configured cases and57 still requiring target/semantics validation. Growth-description parsing was also fixed in code. Full shop gameplay remains NOT TESTED. See [deployment research](2026-10-12_01_EQUIP_SHOP_GM_RESEARCH.md).

The snapshot strings contain UTF-8 text decoded through CP437. Names/descriptions below were recovered by reversing that encoding, without changing the snapshot. Existing truncation, `????` for 2266, absent descriptions for 3353/3547/3548, and inconsistent wording are evidence, not permission to invent rewards. Candidate names and classifications use this October snapshot, not July name assumptions. Target IDs not present in the snapshot cannot be independently certified as live from this input alone.

| Code | Classification | Candidates | All listings hidden (`st=5`) |
|---|---|---:|---:|
| S | Existing supported pet route, subject to live creature-name matching and normal capacity/level/service prerequisites | 25 | 7 |
| P | Existing pet route is usable/selected, but incomplete or description semantics are not fully honored | 12 | 2 |
| D | Missing configuration; deterministic named contents/counts recoverable without inventing a pool or rate | 70 | 8 |
| B | Blocked: reward/effect identity or specialized persistence/expiry/binding semantics need resolution beyond a simple bag definition | 32 | 4 |
| A | Ambiguous/incomplete: random pool, probability, range, choice, class mapping, or insufficient contents evidence | 112 | 11 |
| F | False positive for reward-bag repair: storage expansion, quest/social/visual/targeted consumable rather than a reward package | 74 | 16 |
| **Total** | | **325** | **48** |

**37 have an existing pet route (25 S +12 P); 214 genuine reward candidates remain missing/blocked/ambiguous (70 D +32 B +112 A); 74 are not reward-bag fixes.** These are static route classifications, not gameplay passes. “S” does not independently verify the live creature catalog or every eligibility rule.

### Catalog presence is not client availability

The exhaustive table preserves every `shopID:state` listing, including duplicate listings, shop 0, shop -1, and shop 999/state -1. **H means every listing has `st=5`, hence hidden**; these 48 must not be described as currently purchasable bags. A mixed listing with a hidden entry is not globally hidden. The other 277 have at least one non-5 entry, **not** 277 proven client-visible offers. Shop reachability, event activation, shop kind, gates, and the meanings of states 1/2/4/-1 still matter. Sixteen candidates have only shop 0/-1/999 records, not proof of an ordinary accessible shop.

## Actual Go routes and limitations

- `internal/application/item/service.go:initEffectHandlers/applyItemEffect`: types 508/509/550 go directly to `questItemEffectHandler`; type 500 goes to `scrollEffectHandler`. Type routing precedes generic fallback. An ID appearing in `walletTemplateIDs` does **not** make a type-550 item a working currency reward.
- `internal/application/item/effect_quest_item.go`: configured awards first, then description-based gift choices, then pet-bag fallback; otherwise `ErrCannotUseItem`. Under the snapshot award data, all 325 miss the configured-awards branch. Null awards do not exclude the other two branches; subsequent parent configuration is outside this snapshot classification.
- `internal/application/item/pet_bags.go`: ordinary fallback requires type 508/550, useType 1, literal normalized **`pet`**, and a creature alias immediately after `pet`. A named creature without that word is insufficient. `pet he` is rejected. Bonus inventory rewards and their probabilities are not parsed. The `hoac` branch does not establish an authoritative probability distribution, and the second alternative must actually be collected by the matcher.
- `internal/application/item/effect_gift_box.go`: choice detection handles only creature choices, not general skill books, feathers, wings, keys, recipes, or medals. The handler requires an exact normalized full creature-name substring, unlike ordinary fallback's bracket-stripped aliases. A one-choice result can auto-contract through `tryAutoContractPets`; multiple choices use the existing selection flow. Choice-looking text can therefore fail with no creature choices instead of implementing the advertised item choice.
- Description quality uses honor-scaled integers: **15 = growth 1.5**. The parent owns the quality-parser fix; this audit edits no code. Numeric 1.5 shorthand may be supported by that fix, but **“Cam” alone does not authorize assigning 2.3**, and wording such as “ảo hóa cam 2.5” is not necessarily recognized by the numeric-quality parser.
- `internal/application/item/effect_award.go`: known destinations include creature 12, equipment 19, item-template 29, basic currency 30, experience 31, buff 32, title 33, skill 34, registered game currency 35, medal 36. Configuration still needs the correct target table/ID; a named fairy or mount is not automatically an inventory item or a pet. `currency_keys.go` supplies the supported counter registry. Never send a special counter through generic silver.
- `effect_box.go` cannot repair this snapshot via I1/I2: every candidate has I1 zero, and the 508/509/550 dispatcher does not fall through to it. `effect_default.go` rejects those types. `effect_scroll.go` currently returns a success message without storage expansion: **325/326 are false positives with a real no-op-effect risk**, not bags requiring reward contents.
- `internal/presentation/rtmp/handlers/item/special.go` has separate card RPCs for rename 2126 and sex change 2844. Their existence is not generic `useItem` bag support; the F category does not mean every special RPC is absent.

## Deterministic missing definitions: all 70 D candidates

“Deterministic” describes recoverable reward intent, **not installed configuration**. Resolve target IDs by the current live target names, validate reward families, and retain the stated durations/grades. Do not assume an outer bag's usability implies that an inner key/bag also works.

### Exact material/inner-pill mappings supplied to the parent

Each material bag explicitly contains **12 units of grade 4**. Award family 29 with `payload.materialColorCode=3`, **not 4**: client POSTFIX indices 0..4 represent grades 1..5. Equipment-quality conversion must not be used as the material grade. `award_material_color.go` already honors this separate color payload during capacity checking and grant.

| Bag | Snapshot reward name | Target item ID | Count | Client color |
|---:|---|---:|---:|---:|
| 1455 | Kim Loại Hiếm | 24 | 12 | 3 |
| 1456 | Gỗ Tốt | 21 | 12 | 3 |
| 1457 | Pha Lê | 23 | 12 | 3 |
| 1458 | Gấm Vóc | 26 | 12 | 3 |
| 1459 | Da Thú | 28 | 12 | 3 |
| 1469 | Kim Loại | 7 | 12 | 3 |
| 1470 | Gỗ | 18 | 12 | 3 |
| 1472 | Ngọc | 22 | 12 | 3 |
| 1473 | Vải | **25, live common material Vải Thô** | 12 | 3 |
| 1474 | Lông Thú | 27 | 12 | 3 |
| 1616 | Nội Đơn Bách Luyện | 274 | 10 | Not a material-grade reward |

The parent separately queried the compose database and verified common material **Vải Thô25**, kind6/type604, and **Nội Đơn Bách Luyện274**. Bag1473's generic Vải refers to this common fabric, not premium Gấm Vóc26. The original exploratory ID12 guess was a jewel and was not installed. This report's snapshot contains source bags, so the target validation is separate live parent evidence, not a DB call by the auditor.

### Other explicit inventory bundles/counts

| Bag ID(s) | Exact reward intent from current description | Route once target identity is confirmed |
|---|---|---|
| 1463 | 1 Viễn Cổ Thần Binh (chính) +1 Hiền Triết Thạch | Grant the two named items; **not** an invented random orange weapon |
| 1508 | 1 Viễn Cổ Thần Binh (hỗ trợ) +1 Hiền Triết Thạch | Same distinction for the supporting relic |
| 1498 | 3 Sách Tăng Kinh Nghiệm 3 lần +3 Sách Luôn Gặp Địch +2 Sách Dịch Chuyển Mật Đạo | Item29 bundle; resolve the snapshot's exact names |
| 1559 | 8 Nội Đơn Nhiên Thiêu | Item29 |
| 1572 | 1 chìa khóa Yếu Quyết Hệ Long +1 Ma Huyễn Bảo Rương | Item29 bundle; inner key opening remains separate |
| 1579 | 1 chìa khóa kỹ năng cao cấp Hệ Thực Vật +1 Ma Huyễn Bảo Rương | Item29 bundle; no random skill chosen here |
| 1833 | 3000 **Bảo Thạch Thăng Tinh** | Item29; confirm alias before assigning a plain Thăng Tinh Thạch ID |
| 1834 | 10 **Sách Vận Chuyển Mật Đạo** | Item29; description differs from bag name “Dịch Chuyển” |
| 1887 | 1500 Thăng Tinh Thạch | Item29 |
| 2129 | 5 Canh Khoái Khẩu Cực Phẩm +5 Gân Bò Xào +5 Canh Gan Dê +5 Nem Nướng | Item29; do not substitute a generic potion package |
| 2914 | 40 Long Diên Hương | Item29 |
| 2957 | 100 Tinh Túy Linh Giáp | Item29; distinct from Tinh Hồn Linh Giáp |
| 3674 | 8 Nội Đơn Huyền Thú | Item29 |
| 3693 | 24 Tinh Hồn Linh Giáp +2 Túi Quà Sách Kỹ Năng Trung Cao Cấp +3 Nội Đơn Huyền Thú +3 Nội Đơn Chúc Phúc +2 Sách Huấn Luyện Pet Cao Cấp | Item29 bundle; no invented contents for inner bag3012 |
| 3753 | 24 Tinh Hồn Linh Giáp +2 Túi Quà Mệnh Hồn Cao Cấp +2 Chìa Khóa Thủy Tinh +1 Chìa Khóa Nguyệt Quang +1 Túi Quà Sách Kỹ Năng Trung Cao Cấp +3 Nội Đơn Huyền Thú | Item29 bundle; unresolved inner random bags stay unresolved |
| 3922 | 2 Túi Quà Chìa Khóa +2 Sách Huấn Luyện Pet Cao Cấp +10 Ngọc Vinh Quang +2 Lông Vũ Chúc Phúc +1 Chìa Khóa Nguyệt Quang | Item29 bundle |
| 4871 | 1000 Bụi Vô Hạn | Item29; local seed has an inventory template of this name |
| 5029 | 240 Đá Khế Ước | Item29 |
| 5246 | 1 Kim Cương Cấp 3 | Item29 material, grade3 → color2; confirm target material ID live |
| 5911 | 12 Tinh Hồn Linh Giáp +8 Tinh Hoa Phù Văn +1 Túi Quà Kỹ Năng Trung Cấp +3 Túi Quà EXP Ấn Chương Cao Cấp +1 Túi Hủy Diệt Phù Cam | Item29 bundle; unresolved inner-use semantics not repaired by opening outer bag |
| 6030 /6253 | 1000 /100 Kỳ Linh Thạch | Item29 |
| 6047 | 1 Túi Hộ Thú +5 Túi Thú Thạch Cam Tự Chọn +1 Túi Quà Sách Kỹ Năng Cấm +3 Ấn Chương Đấu Pet +50 Bùa Cầu Nguyện +3 Lông Vũ Chúc Phúc Cao Cấp +8 Quả Linh Tê +10 Hộp Chocolate | Resolve every live target and its family; medal-vs-item identity must be checked, not guessed |
| 6254 /6255 | 10 /50 Túi Phúc Niên | Item29 →6249; exact outer contents, not authority to define6249's random pool |

### Named single rewards and supported counters

- **Titles (33):** 1707 → Bách Niên Giai Lão; 1708 → Tình Định Tam Kiếp; 1709 → Vĩnh Thế Duyên Tình; 2910 → Lao Động Là Vinh Quang, explicitly no added stats. Resolve live title IDs. Do not invent duration or title-buff definitions.
- **Named pets needing configured type12 rather than current ordinary fallback:** 3254 →1 Camy/1.5; 3731 →1 Bạo Liệt Long/2.5; 3732 →1 Hỏa Cơ Giáp/2.5; 3733 →1 Thợ Săn Eskimos/2.5; 3734 →1 **Bàng Thạch Long**/2.5, preserving snapshot spelling pending alias validation; 3735 →1 Ma Thạch Tượng/2.5; 4605 →1 Nai Xanh cam/2.5; 4840 →1 Dê Con II Bảo Bảo/1.5; 6297 →1 Võ Sĩ Shiba/1.5; 7036 →1 Lôi Lân Ảnh/1.5. None has a matching `pet <name>` ordinary fallback phrase; do not mistake recognizable pet intent for a currently implemented route.
- **Fixed named wearable/flyer rewards:** 3380 →1 Hỏa Phụng Hoàng (30 Ngày); 3382 →1 Ngựa Một Sừng (30 Ngày). 4818/4819 → Thời Trang Brazil/Bikini (30 Ngày); 4820 → Xe Bay Ếch Xanh (30 Ngày); 4821 → Hỏa Phụng Hoàng (30 Ngày); 4822 → Phi Long (30 Ngày); 4823 → Ngựa Một Sừng (30 Ngày); 4824 → Furi (30 Ngày); 4825/4826 → Máy Bay Giấy Xanh/Hồng (30 Ngày); 4827 →1 Thời Trang Băng Tuyết Kỳ Duyên (30 Ngày). 4918/4919 explicitly grant Thời Trang Arale Đỏ/Hồng, with no duration stated; 6975 explicitly grants1 Thời Trang Fantasy (30 Ngày). Names are exact reward intent, **not authority to invent runtime dress/flyer IDs, expiry payloads, or counts for unstated bundles**. A single named reward is distinguished from an unknown multi-item package; confirm its grantable template/family before configuration.
- **Registered counter rewards (35):** 4054 →1000 `npPnt`; 5215 →500 `runeExp`; 5302 →500 `realSoulCrystal`; 5355 →1000 `warSprite` **and**200 `battleSprite`; 5877 →5000 `monsterHeart`. These have fixed descriptions and existing corresponding counter accessors; type550 prevents the misleading generic-wallet route from implementing them today.

## Supported/partial pet routes: evidence for all S/P rows

- **S ordinary fallback, growth1.5 stated:** 1178 Kim Ngưu; 1310 Hổ Bì Dương; 1398 Tinh Linh Hộ Thú; 1529 Thần Long Viễn Cổ; 1532 Dứa Mật; 1625 Tiểu Lương Tử; 1877 Đinh Long; 2019 Y Tá MM; 2084 Hoàng Tử Thiên Sứ; 2085 Công Chúa Ác Ma; 2128 Kim Hổ; 2262 Momo; 2815 Nấm Điện; 3312 Loni; 3402 Pony; 6501 Burin. Literal pet wording makes these eligible for description-based resolution without award rows.
- **S ordinary fallback, growth2.3 stated:** 1806 Kim Ngưu; 1808 Dứa Mật; 1809 Tiểu Lương Tử; 1810 Y Tá MM; 1811 Thần Long Viễn Cổ; 1812 Đinh Long; 1814 Thiên Sứ Mít Ướt. All their listings are hidden.
- **S creature-choice route:** 3691 explicitly chooses1 purple pet at1.9 from named creatures; 4817 chooses a blue pet at1.4. **4817 says10 but enumerates nine named pets**; the code can expose matched named choices, not recover the missing tenth. This is supported selection mechanics with a catalog-text completeness caveat, not permission to add an option.
- **P1326:** description advertises Thiên Sứ Mít Ướt **or** Ác Ma Quấy Phá/1.5. Ordinary matching requires `pet` before an alias; the text places it before the first alternative only. The code's random-alternative facility is not proof that both alternatives are collected or that the advertised distribution works. Do not declare full two-pet-pool parity.
- **P1813/1815:** literal pet names can contract, but only “Cam” is stated, without a numeric growth. Fallback's default growth is not proof of orange reward parity; do not infer2.3 from adjacent bags.
- **P2516/2789/3624/5233/5937/6582/6693:** pet core is recognized, but the chance bonus is absent from fallback: respectively Quả Hạch Thần Kỳ, Cà Rốt Thần Kỳ, Lân Thánh Naga, Sừng Thiếc, Quả Đào Ngàn Năm, Phô Mai Ngọt Ngào, Thanh Hoa Long Lân. No bonus rates are supplied. Numeric1.5 shorthand/odd wording is the parent's quality-parser concern; do not alter these bags into arbitrary full reward pools.
- **P6132:** `mở chọn nhận` can resolve one Moltres and auto-contract it; the advertised chance Thiên Địa Linh Thạch is not handled by that route.
- **P4786:** named creature choices can be presented, but “thần thú ảo hóa cam2.5” does not use the normal `phẩm chất <number>` phrase. The existing parser must not be assumed to honor2.5 or all ảo-hóa semantics. Description also spells Bàn Thạch Long, whereas3734 spells Bàng Thạch Long; verify actual creature aliases rather than normalizing both into an assumed ID.

## Blocked B rows: deterministic intent is not enough

| IDs | Description evidence and blocker |
|---|---|
| 1063,1132,2271,2272,2273 | Respectively1000,100000,100000,1000000,10000000 bạc **unbound**, but ngân phiếu when bound. Generic money awards do not implement the binding-dependent branch. Do not grant bạc unconditionally. |
| 2336,2992,2993,2994 | Fixed thần tu400000/20000/50000/100000; correct persisted counter identity is not established by the supported currency registry. Not silver or arbitrary EXP. |
| 3304 | Người Tuyết Hung Bạo title, +8% maxHP,30 days, no duration stacking; timed title/buff semantics need resolution, not just title33. |
| 4147 | Anh Hùng Giáng Sinh title/+5%HP; verify title/buff linkage and actual definition. |
| 3394–3398,3609,4030 | One **tinh linh**, respectively Rocky[Xảo], Băng Nữ Koura[Thần], Puca[Nhẫn], Rồng Xanh Abu[Lực], Yêu Tinh Nicole[Trí], Tuần Lộc Miya[Thần], Thần Chết[Thần]; not literal ordinary pet grants. Correct fairy destination and subtype identity need confirmation. |
| 3816,5275 | 25 EXP Thánh Giả Văn Chương /2000 EXP Ấn Chương; do not confuse subsystem EXP with character EXP or a medal count. |
| 4591,4592 | 500/1500 điểm Anh Linh; counter identity unresolved, generic wallet is not reachable here. |
| 4673,4674 | 10/50 Bụi Ấn Thạch. Whether this means an inventory template or the `stoneSealPoint` counter must be established; similarity of names is insufficient. |
| 4803,4804,4805,6492 | Named mount reward Hồng Liên Bôn Lôi/Hàn Sương Thiểm Điện/Viễn Cổ Chiến Hùng/Burin. Need mount-specific identity/persistence.6492 explicitly **must not refresh remaining mount usage time**. |
| 5948 | 20 Bụi Ảo Năng; target item/counter identity unresolved. |
| 6126 | 2520 điểm thám hiểm; do not substitute an unrelated `explorerPnt`/event counter by English resemblance. |
| 6524 | 100 Văn Chỉ; inventory/counter destination unresolved by the supplied target data. |
| 6247 | Names Thệ Ước Lam Ảo, Túi Pet Cánh Tím, Túi Quà Phi Hành Hy Hữu, Pháo Kích, Túi Quà Quái Thú Hủy Diệt, Túi Quà Tân Huy Chương (30 Ngày). Counts are not enumerated; mixed clothing/package/medal destination and duration semantics need confirmation. No invented six-item count1 package. |

## Ambiguous A rows: description-based reasons, no invented pool/rate

- **Mixed bundles with count/identity ambiguity:** 1327 mentions9 roses and5 horns but does not establish counts/identities for every other item; 1462 has “5 Tiên Hoa Trứng Gà” without clearly separating flower/egg quantities. Do not synthesize the rest of either bundle.
- **Keys/eggs/hammer operations with uncertain rewards or ranges:** 273,1361,2089,2109,2354,2407,2787,2788,2899–2903,3354,3679,3689,3690,5950,6559,6561,6589. Key509 has no description-pet fallback even when the text mentions pets. Stated maximums (e.g.2000 Hoa Tinh Tú,100 Linh Vũ,50 Pha Lê Vĩnh Hằng/Con Dấu/Thư,1000 Kẹo) do **not** establish a deterministic count or distribution.
- **Random/scaled/daily-limited special-point consumables:** 2480,2795,2796; amounts/bonuses are unresolved and daily3/5 limits are gameplay requirements, not a rate to guess.
- **Random pool/quality/class outcomes:** 2829 (four wings +chance feather),3229,3245,3311,3368,3385,3719,3730,3864;3569/3570/4602 (class-dependent relic);5904 (180–1000),5906 (30000–100000),3839/6278 (up to500 evolution water). Neither uniform distributions nor “always maximum” are justified.3368 says random; “Tẩy Tủy Đan cao cấp” does not establish a unique variant ID.
- **Non-pet or nested choice:** 3376,3377,3621,3639,3755,3836,3837,3961,4597,4720,4785,4787,4873,4965,5119,5120,5281,5282,6028,6575,6632,6633. Existing creature-choice code is not a generic item-choice implementation.6575's800 Hắc Diệu Thạch vs370 Tinh Hoa is an explicit **choice**, not authority to grant both.3961 also has “30 Hồng” wording for the blue variant;4785's list is truncated.
- **Undefined/general descriptions:** 3012 (“nhấp đôi để mở”),3353 (null),3547/3548 (null; name alone has4/8 but absent description cannot establish exact behavior),3900 (“món ngon”),4667,4722,4879,4880,4922,4996,5236,5237,5285,5889,5890,6765. Do not invent foods, event packages, soul contents, or a dog-food effect.1835 names “Túi Sách x3 Exp (3 giờ)” but target/package/count semantics need confirmation; it is not evidence to grant an arbitrary3-hour buff.
- **Fashion names without explicit reward descriptions:** 4989–4994 only say “túi quà thời trang”;5229/5231 supply flavor text, not explicit contents. The names suggest skins/durations but do not establish a recovered award definition. Keep separate from the D rows whose descriptions actually name the granted wearable.
- **Missing ordinary pet trigger plus unknown chance bonus:** 4190 Hắc Nhân Mã/1.5 +chance Đôi Cánh Thần Kỳ;6797 Kim Tước/1.5 +chance Ly Hỏa Chi Linh;6861 Huyền Minh Linh Vũ/1.5 +chance Vạn Linh Bội;6974 Bạch Hổ Cuồng Nộ/1.5 +chance Bạch Ngọc. These descriptions lack the literal `pet <name>` trigger; unlike the P rows, no usable ordinary pet-core route is established. Exact core intent does not recover bonus probabilities.
- **Seasonal chance pools:** 6168,6186 (leaf variants plus possible extra ingredients),6249,6298 (letter combinations),6503,6546,6557,6566,6585,6599,6626,6706,6718,6741,6788,6815,6999,7052.6585 mentions a guaranteed Túi Cẩu Lương but then a random tail;6626 mentions Túi Cẩu Lương (24 Giờ) plus a random tail. Configuring only the core would erase the advertised tail. Some descriptions mention “Túi Pet” inside a random inventory-pool list; that is **not** evidence of an intended direct creature grant, even if broad text-based code selects a pet-related handler.

## False positives F: all 74 outside reward-bag configuration

- **Storage, not contents:** 325/326 increase inventory/pet slots; current type500 message-only handler does not actually expand them.
- **Quest/dialogue/digging items:** 389,528,533,591,726,843,949,983,1964.726 is a quest-letter parcel, not a license to invent shop rewards. Their current generic508 path cannot implement the advertised quest/world behavior merely by adding arbitrary award rows.
- **Social/intimacy/reputation/PK/cards:** 32,122,133,724,783,1302,1303,1315,1364,2126,2142,2844.1364 deletes a selected pet skill, not a potion-reward bag;2126/2844 have separate special-card RPCs. Do not treat social gift packaging (1315) as a missing chocolate-count bag.
- **Location/teleport/target utility:** 1033,1034,1036,1311,1363,1397,1515,1615,2947,2948,3552.
- **Visual transformations/chat effects:** 1402–1406,1705,2316,2339,2343,2344,2355,2511,2799,4148.
- **Wedding invitations/decorations/gifting:** 1691–1693,1702,1706,1710,1712,1713. Invitations mention gifts for attendees but are not themselves deterministic reward bags.1702 also mentions chance rewards; its main fireworks/event operation is not fixed by an invented drop table.
- **Fireworks/event tools:** 1175,1868,2078,2108,2266,3300,3351; do not equate fireworks success text with a full area/server visual effect.
- **Other direct subsystem consumables:** 1581 (30-minute capture effect),2702 (Achilles tool),3630 (auto-instance completion),3815 (magic-array fruit without an explicit recovered amount),5230/6199 (direct fashion flavor text),6201/6527/6528 (7-day pet-divinity shapes),6232 (event exchange token),6274 (tower skip/time-limit/rebirth behavior). Registered wallet IDs or visual token names do not supply those mechanics.

No type501 potion bags are present in these325. A broad text search can find storage/pet-use/food/potion wording without indicating a missing reward bag; type and actual description intent govern classification.

## Exhaustive candidate ledger

Every snapshot candidate appears once, in snapshot ID order. `shopID:state` retains listing multiplicity; **H = all st5 hidden**. Codes link to the evidence and current/missing routes above; D does not mean already configured, F does not mean its separate special RPC necessarily fails.

| ID | Current snapshot name | shopIDs:states | Audit |
|---:|---|---|---|
| 32 | Bó Hoa Hồng To | 65:0 | F |
| 122 | Thẻ Ly Hôn | 65:4 | F |
| 133 | Thẻ Hối Cải | 65:4, 70:0, 99:0 | F |
| 273 | Chìa Khóa May Mắn | 65:5 | A H |
| 325 | Sách Mở Rộng Túi Đồ | 35:0 | F |
| 326 | Sách Mở Rộng Túi Đựng Pet | 0:5, 35:1 | F |
| 389 | Sách Thần Bí | 68:0 | F |
| 528 | Sơ Đồ Dẫn Đường | 68:0 | F |
| 533 | Mảnh Da Dê Kho Báu | 68:0 | F |
| 591 | Bưu Kiện 5 | 68:0 | F |
| 724 | Tiên Hoa | 5:0, 6:0, 7:0, 8:0, 9:0, 10:0, 11:0, 13:0, 14:0, 15:0, 16:0, 65:0, 91:0, 92:0, 98:0, 101:0, 105:0, 108:0, 115:0, 122:0, 999:-1 | F |
| 726 | Túi Thư | 68:0 | F |
| 783 | Trứng Gà | 5:0, 6:0, 7:0, 8:0, 9:0, 10:0, 11:0, 13:0, 14:0, 15:0, 16:0, 65:0, 91:0, 92:0, 98:0, 101:0, 105:0, 108:0, 115:0, 122:0, 999:-1 | F |
| 843 | Hộp Kim Loại Bí Ẩn | 68:0 | F |
| 949 | Bụi Thủy Tinh Thần Bí | 68:0 | F |
| 983 | Quyển Sách Khắc Chữ | 68:0 | F |
| 1033 | Thiên Nhãn Phù | 5:0, 6:0, 7:0, 8:0, 9:0, 10:0, 11:0, 13:0, 14:0, 16:0, 65:0, 122:0 | F |
| 1034 | Truy Hồn Phù | 11:0, 13:0, 14:0, 16:0, 65:0, 91:0, 122:0, 999:-1 | F |
| 1036 | Tiêu Diêu Phù | 5:0, 6:0, 7:0, 8:0, 9:0, 10:0, 11:0, 13:0, 14:0, 15:0, 16:0, 65:0, 122:0 | F |
| 1063 | Bạch Ngân Lễ | 35:5 | B H |
| 1132 | Bạch Ngân Lễ (10 Vạn) | 35:0 | B |
| 1175 | Pháo Mừng Năm Mới | 65:5 | F H |
| 1178 | Túi Quà Kim Ngưu | 67:0, 70:5, 102:0, 106:0 | S |
| 1302 | Chocolate | 5:0, 6:0, 7:0, 8:0, 9:0, 10:0, 11:0, 13:0, 65:5, 91:0, 92:0, 101:0, 105:0, 108:0, 122:0, 999:-1 | F |
| 1303 | Hoa Hồng | 5:0, 6:0, 7:0, 8:0, 9:0, 10:0, 11:0, 13:0, 65:5, 91:0, 122:0, 999:-1 | F |
| 1310 | Túi Quà Kim Dương | 67:0, 70:5, 106:0 | S |
| 1311 | Hình Ảnh Bất Ly | 65:0 | F |
| 1315 | Hộp Chocolate | 65:0 | F |
| 1326 | Túi Quà Pet Lễ Tình Nhân | 67:0, 70:5, 102:0, 106:0 | P |
| 1327 | Túi Quà Tình Duyên | 65:5 | A H |
| 1361 | Chìa Khóa Thần Bí | 5:0, 6:0, 7:0, 8:0, 9:0, 10:0, 65:5 | A |
| 1363 | Tiêu Diêu Phù (30 Ngày) | 65:1, 108:0 | F |
| 1364 | Thuốc Tẩy Kỹ Năng Pet | 67:0 | F |
| 1397 | Rada Siêu Nhỏ | 11:0, 13:0, 14:0, 16:0, 65:5, 122:0 | F |
| 1398 | Túi Quà Tinh Linh | 67:0, 70:5, 102:0, 106:0 | S |
| 1402 | Thẻ Hoàn Nguyên | 65:5 | F H |
| 1403 | Thẻ Tiểu Cương Thi | 65:5 | F H |
| 1404 | Thẻ Ma Rương | 65:5 | F H |
| 1405 | Thẻ Mê Ngữ | 65:5 | F H |
| 1406 | Thẻ Minh Thần | 65:5 | F H |
| 1455 | Túi Quà Kim Loại Hiếm | 66:2 | D |
| 1456 | Túi Quà Gỗ Tốt | 66:2 | D |
| 1457 | Túi Quà Pha Lê | 66:2 | D |
| 1458 | Túi Quà Gấm Vóc | 66:2 | D |
| 1459 | Túi Quà Da Thú | 66:2 | D |
| 1462 | Tinh Thần Bao | 70:5 | A H |
| 1463 | Túi Quà Viễn Cổ Thần Binh | 70:5 | D H |
| 1469 | Túi Quà Kim Loại | 66:2 | D |
| 1470 | Túi Quà Gỗ | 66:2 | D |
| 1472 | Túi Quà Ngọc | 66:2 | D |
| 1473 | Túi Quà Vải | 66:2 | D |
| 1474 | Túi Quà Lông Thú | 66:2 | D |
| 1498 | Bao Thăng Cấp Cuồng Loạn | 65:0 | D |
| 1508 | Túi Quà Viễn Cổ Thần Binh (Trợ Thủ) | 70:5 | D H |
| 1515 | Tiêu Diêu Phù (7 ngày) | 65:4 | F |
| 1529 | Túi Thần Long | 67:0, 70:5, 99:0, 102:0 | S |
| 1532 | Túi Quà Pet Dứa | 67:0, 70:5, 102:0, 106:0 | S |
| 1559 | Túi Quà Nội Đơn Nhiên Thiêu | 67:5 | D H |
| 1572 | Túi Quà Kỹ Năng Hệ Rồng | 67:5, 70:5 | D H |
| 1579 | Túi Quà Kỹ Năng Cao Cấp Hệ Thực Vật | 67:5, 70:5 | D H |
| 1581 | Luyện Thú Đơn | 67:5 | F H |
| 1615 | Radar Nhỏ (7 ngày) | 65:4 | F |
| 1616 | Túi Nội Đơn Bách Luyện | 67:4, 102:0 | D |
| 1625 | Túi Quà Thanh Lương Nữ Tỳ | 67:0, 70:5, 102:0, 106:0 | S |
| 1691 | Thiệp Hồng Thông Thường | 12:0 | F |
| 1692 | Thiệp Hồng Cao Cấp | 12:0 | F |
| 1693 | Thiệp Hồng Quý Giá | 12:0 | F |
| 1702 | Pháo Hoa Hôn Lễ | 12:0 | F |
| 1705 | Thẻ Xe Diễu Hành | 12:0 | F |
| 1706 | Quả Cầu Trang Trí | 12:0 | F |
| 1707 | Nhẫn Kết Hôn Thông Thường | 12:0 | D |
| 1708 | Nhẫn Kết Hôn Cao Cấp | 12:0 | D |
| 1709 | Nhẫn Kết Hôn Quý Giá | 12:0 | D |
| 1710 | Lì Xì Kết Hôn | 12:0 | F |
| 1712 | Hộp Quà Kẹo Mừng | 12:0 | F |
| 1713 | Giỏ Hoa Tươi | 12:0 | F |
| 1806 | Túi Quà Kim Ngưu Cam | 70:5 | S H |
| 1808 | Túi Quà Dứa Mật Cam | 70:5 | S H |
| 1809 | Túi Quà Tiểu Lương Tử Cam | 70:5 | S H |
| 1810 | Túi Quà Y Tá MM Cam | 70:5 | S H |
| 1811 | Túi Quà Thần Long Viễn Cổ Cam | 70:5 | S H |
| 1812 | Túi Quà Đinh Long Cam | 70:5, 70:5 | S H |
| 1813 | Túi Quà Tinh Linh Hộ Thú Cam | 70:5 | P H |
| 1814 | Túi Quà Thiên Sứ Mít Ướt Cam | 70:5 | S H |
| 1815 | Túi Quà Ác Ma Quấy Phá Cam | 70:5 | P H |
| 1833 | Túi Quà Thăng Tinh Thạch | 70:5 | D H |
| 1834 | Túi Quà Sách Dịch Chuyển Mật Đạo | 70:5 | D H |
| 1835 | Túi Quà Sách x3 Exp | 70:5 | A H |
| 1868 | Pháo Hoa Ngày Lễ | 65:4 | F |
| 1877 | Túi Quà Đinh Long | 67:0, 102:0, 106:0 | S |
| 1887 | Túi Thăng Tinh Thạch | 70:5 | D H |
| 1964 | Tinh Thể Độn Thổ | 68:0 | F |
| 2019 | Túi Quà Y Tá MM | 67:0, 102:0, 106:0 | S |
| 2078 | Pháo Hoa Ngũ Sắc | 0:0 | F |
| 2084 | Túi Quà Hoàng Tử Thiên Sứ | 67:0, 99:0, 102:0 | S |
| 2085 | Túi Quà Công Chúa Ác Ma | 67:0, 99:0, 102:0 | S |
| 2089 | Trứng Thần Thú | 13:0, 122:0 | A |
| 2108 | Pháo Trúc | 95:0 | F |
| 2109 | Ovo de Páscoa Magic Campus | 65:5 | A H |
| 2126 | Thẻ Đổi Tên | 65:4, 99:0, 115:0 | F |
| 2128 | Túi Quà Kim Hổ | 67:5, 99:0, 102:0 | S |
| 2129 | Túi Mỹ Thực Cuồng Chiến | 65:4 | D |
| 2142 | Thẻ Chính Danh | 70:0 | F |
| 2262 | Túi Quà Công Chúa Momo | 67:4, 99:0, 102:0 | S |
| 2266 | ???? | 65:5 | F H |
| 2271 | Rương Tiền (100000) | 4:0, 32:0, 60:0, 61:0, 64:0, 76:0 | B |
| 2272 | Rương Tiền (1 triệu) | 4:0, 32:0, 60:0, 61:0, 64:0, 76:0 | B |
| 2273 | Rương Tiền (10 triệu) | 4:0, 32:0, 60:0, 61:0, 64:0, 76:0, 114:0 | B |
| 2316 | Thẻ Kẹo Ngọt | 65:5 | F H |
| 2336 | Kết Tinh Thần Tu | 114:0 | B |
| 2339 | Thời Trang Học Sinh (15 ngày) | 65:5 | F H |
| 2343 | Thẻ iPhone | 65:5 | F H |
| 2344 | Thẻ iPad | 65:5 | F H |
| 2354 | Chìa Khóa Thần Thú | 67:5 | A H |
| 2355 | Thời Trang Học Sinh (30 ngày) | 65:5 | F H |
| 2407 | Chìa Khóa Momo | 67:5 | A H |
| 2480 | Kết Tinh Thần Tu Cao Cấp | 70:0 | A |
| 2511 | Thẻ Âu Lạc | 65:5 | F H |
| 2516 | Túi Quà Nhím Xanh | 67:4, 99:0, 102:0 | P |
| 2702 | Gai Achilles | 65:4, 85:0 | F |
| 2787 | Túi Rồng Cao Cấp | 67:5 | A H |
| 2788 | Thiếu Nữ Rồng | 67:5 | A H |
| 2789 | Túi Quà Thỏ Baby | 67:4, 99:0, 102:0 | P |
| 2795 | Kết Tinh Thần Tu (Mỗi ngày chỉ dùng 3 cái) | 70:0 | A |
| 2796 | Kết Tinh Thần Tu (Mỗi ngày chỉ dùng 5 cái) | 70:0 | A |
| 2799 | Thẻ Chơi Xỏ Cá Tháng Tư | 65:5 | F H |
| 2815 | Túi Quà Nấm Điện | 67:4, 102:0 | S |
| 2829 | Túi Quà Cánh Sơ Cấp | 87:0, 109:0 | A |
| 2844 | Thẻ Đổi Giới Tính | 65:4, 70:5, 98:0, 99:0 | F |
| 2899 | Búa Sắt | -1:0 | A |
| 2900 | Búa Đồng | -1:0 | A |
| 2901 | Búa Bạc | -1:0 | A |
| 2902 | Búa Vàng | -1:0 | A |
| 2903 | Búa Pha Lê | -1:0 | A |
| 2910 | Huy Chương Lao Động Là Vinh Quang | -1:0 | D |
| 2914 | Long Diên Hương (40 cái) | 122:0 | D |
| 2947 | Tiêu Diêu Phù Cao Cấp | 15:0, 65:4, 101:0 | F |
| 2948 | Tiêu Diêu Phù Cao Cấp (1 giờ) | 65:4 | F |
| 2957 | Túi Quà Lớn Tinh Túy Linh Giáp | 112:0 | D |
| 2992 | Kết Tinh Thần Tu A | 65:5 | B H |
| 2993 | Kết Tinh Thần Tu B | 65:5 | B H |
| 2994 | Kết Tinh Thần Tu C | 65:5 | B H |
| 3012 | Túi Quà Sách Kỹ Năng Trung Cao Cấp | 107:0, 107:0 | A |
| 3229 | Túi Quà Sinh Nhật | 92:0 | A |
| 3245 | Túi Quà Lông Vũ Cấp 2 | 109:0 | A |
| 3254 | Túi Quà Camy | 67:4, 102:0, 106:0 | D |
| 3300 | Khắc Tinh Gà Tây | 94:0 | F |
| 3304 | Huy Chương Người Tuyết Hung Bạo | 999:-1 | B |
| 3311 | Túi Quà Tân Xuân | 0:0 | A |
| 3312 | Túi quà Loni | 67:4, 99:0, 102:0, 106:0 | S |
| 3351 | Pháo Hoa Ngày Xuân | 95:0 | F |
| 3353 | Túi Quà Long Tinh | 67:5 | A H |
| 3354 | Chìa Khóa Tinh Tú | 99:0 | A |
| 3368 | Túi Quà Tẩy Tủy Đan Cao Cấp | 70:0 | A |
| 3376 | Túi Quà Cánh Xanh Dương | 107:0 | A |
| 3377 | Túi Quà Cánh Tím | 107:0, 107:0 | A |
| 3380 | Túi Quà Hỏa Phụng Hoàng (30 ngày) | 109:0 | D |
| 3382 | Túi Quà Ngựa Một Sừng (30 ngày) | 109:0 | D |
| 3385 | Túi Quà Lao Động Vinh Quang | 15:0 | A |
| 3394 | Túi Quà Rocky | 96:0, 109:0 | B |
| 3395 | Túi Quà Băng Nữ Koura | 96:0, 109:0 | B |
| 3396 | Túi Quà Puca | 96:0, 109:0 | B |
| 3397 | Túi Quà Rồng Xanh Abu | 96:0, 109:0 | B |
| 3398 | Túi Quà Yêu Tinh Nicole | 96:0, 109:0 | B |
| 3402 | Túi Quà Pony | 67:4 | S |
| 3547 | Tinh Hồn Linh Giáp (4 cái) | 99:0 | A |
| 3548 | Tinh Hồn Linh Giáp (8 cái) | 99:0 | A |
| 3552 | Sách Hồi Nhóm | 35:4, 101:0 | F |
| 3569 | Túi Quà Viễn Cổ Thần Binh (Chính Trác Việt) | 107:0, 107:0 | A |
| 3570 | Túi Quà Viễn Cổ Thần Binh(Trợ Thủ Trác Việt) | 107:0, 107:0 | A |
| 3609 | Túi Quà Miya | 96:0, 109:0 | B |
| 3621 | Túi Quà Kỹ Năng Trung Cấp | 115:0, 122:0 | A |
| 3624 | Túi Quà Kỳ Kỳ | 67:0, 99:0, 102:0 | P |
| 3630 | Sách Tự Động Hoàn Thành Phụ Bản | 68:0 | F |
| 3639 | Túi Kỹ Năng Tinh Linh | 115:0 | A |
| 3674 | Túi Quà Nội Đơn Huyền Thú Lớn | 0:0, 114:0 | D |
| 3679 | Búa Bách Bảo | 86:0 | A |
| 3689 | Búa Vạn Thú | 86:0 | A |
| 3690 | Chìa Khóa Linh Vũ | 99:0 | A |
| 3691 | Túi Quà Thần Thú Tím | 107:0, 107:0 | S |
| 3693 | Túi Quà Lưu Quang Phi Vũ | 0:0, 115:0 | D |
| 3719 | Túi Quà Mệnh Hồn Cao Cấp | 70:5 | A H |
| 3730 | Túi Quà Đá Kháng Thường | 99:0 | A |
| 3731 | Túi Quà Bạo Liệt Long Cam | 99:0 | D |
| 3732 | Túi Quà Hỏa Cơ Giáp Cam | 99:0 | D |
| 3733 | Túi Quà Thợ Săn Eskimos Cam | 99:0 | D |
| 3734 | Túi Quà Bàng Thạch Long Cam | 99:0 | D |
| 3735 | Túi Quà Ma Thạch Tượng Cam | 99:0 | D |
| 3753 | Túi Quà Ngũ Sắc Rực Rỡ | 114:0 | D |
| 3755 | Túi Quà Sách Kỹ Năng Cao Cấp Kinh Điển | 107:0, 107:0, 122:0 | A |
| 3815 | Nguyên Tố Quả Thực | 109:0 | F |
| 3816 | Ngọc Vinh Quang | 85:0, 114:0 | B |
| 3836 | Túi Quà Mộng Mơ | 0:0, 114:0, 122:0 | A |
| 3837 | Túi Quà Chìa Khóa | 107:0, 107:0, 122:0 | A |
| 3839 | Túi Quà Dược Thủy Tiến Hóa Thú Cưỡi | 0:0, 107:0, 107:0, 114:0, 122:0 | A |
| 3864 | Túi Quà Ngày Hè | 101:0 | A |
| 3900 | Túi Quà Mỹ Vị | 109:0, 122:0 | A |
| 3922 | Túi Quà Sắc Thu | 114:0 | D |
| 3961 | Túi Quà Máy Bay Giấy | 109:0 | A |
| 4030 | Túi Quà Tinh Linh Thần Chết | 96:0, 109:0 | B |
| 4054 | Tinh Hoa Tự Nhiên | 70:0, 107:0, 107:0 | D |
| 4147 | Huy Chương Anh Hùng Giáng Sinh | 71:0 | B |
| 4148 | Gậy Biến Hình Giáng Sinh | 71:0 | F |
| 4190 | Túi Quà Hắc Nhân Mã | 67:4 | A |
| 4591 | Mảnh Vỡ Anh Hồn | 107:0, 107:0 | B |
| 4592 | Đá Anh Hồn | 107:0, 107:0 | B |
| 4597 | Túi Thiên Phú Thạch Cam | 122:0 | A |
| 4602 | Túi Quà Thần Khí Kinh Điển (Vòng 7) | 107:0, 107:0 | A |
| 4605 | Túi Quà Nai Xanh Cam (Xảo) | 107:0, 107:0 | D |
| 4667 | Túi Quà Quốc Sắc Thiên Hương | 114:0 | A |
| 4673 | Mảnh Vụn Ấn Thạch | 107:0, 107:0 | B |
| 4674 | Mảnh Vỡ Ấn Thạch | 107:0, 107:0 | B |
| 4720 | Túi Quà Mệnh Hồn Kinh Điển Mới | 0:0, 115:0 | A |
| 4722 | Túi Quà Kim Xà Cuồng Vũ | 114:0 | A |
| 4785 | Túi Quà Huy Chương Cuồng Chiến | 0:0 | A |
| 4786 | Túi Quà Thú Ảo Hóa Cam | 115:0 | P |
| 4787 | Túi Quà Sách Kỹ Năng Cấm | 107:0, 107:0 | A |
| 4803 | Túi Quà Hồng Liên Bôn Lôi | 107:0, 107:0, 115:0 | B |
| 4804 | Túi Quà Hàn Sương Thiểm Điện | 107:0, 107:0, 115:0 | B |
| 4805 | Túi Quà Viễn Cổ Chiến Hùng | 107:0, 107:0 | B |
| 4817 | Túi Quà Thần Thú Xanh Dương | 108:0 | S |
| 4818 | Túi Quà Thời Trang Brazil (30 Ngày) | 107:0, 107:0 | D |
| 4819 | Túi Quà Thời Trang Bikini (30 Ngày) | 107:0, 107:0 | D |
| 4820 | Túi Quà Xe Bay Ếch Xanh (30 Ngày) | 107:0, 107:0 | D |
| 4821 | Túi Quà Hỏa Phụng Hoàng (30 Ngày). | 107:0, 107:0 | D |
| 4822 | Túi Quà Phi Long (30 Ngày) | 107:0, 107:0, 109:0 | D |
| 4823 | Túi Quà Ngựa Một Sừng (30 Ngày). | 107:0, 107:0 | D |
| 4824 | Túi Quà Furi (30 Ngày) | 107:0, 107:0, 109:0 | D |
| 4825 | Túi Quà Máy Bay Giấy Xanh (30 Ngày) | 107:0, 107:0 | D |
| 4826 | Túi Quà Máy Bay Giấy Hồng (30 Ngày). | 107:0, 107:0 | D |
| 4827 | Túi Quà Băng Tuyết Kỳ Duyên (30 Ngày) | 107:0, 107:0 | D |
| 4840 | Túi Quà Dê Con II | 67:4 | D |
| 4871 | Bụi Vô Hạn (1000) | 112:0 | D |
| 4873 | Túi Quà Thần Khí Phụ Cao Cấp | 109:0 | A |
| 4879 | Túi Quà Liễu Ám Hoa Minh | 114:0 | A |
| 4880 | Túi Quà Phong Hoa Tuyết Nguyệt | 114:0 | A |
| 4918 | Túi Quà Thời Trang Arale Đỏ | 109:0 | D |
| 4919 | Túi Quà Thời Trang Arale Hồng | 109:0 | D |
| 4922 | Túi Quà Vĩnh Kết Đồng Tâm | 114:0 | A |
| 4965 | Túi Quà Thánh Thú Viễn Cổ | 114:0 | A |
| 4989 | Túi Quà Thời Trang Wonderland (30 Ngày) | 109:0 | A |
| 4990 | Túi Quà Thời Trang Tân Xuân (30 Ngày) | 109:0 | A |
| 4991 | Túi Quà Thời Trang Dạ Tiệc (30 Ngày) | 109:0 | A |
| 4992 | Túi Quà Thời Trang Wonderland (15 Ngày) | 109:0 | A |
| 4993 | Túi Quà Thời Trang Tân Xuân (15 Ngày) | 109:0 | A |
| 4994 | Túi Quà Thời Trang Dạ Tiệc (15 Ngày) | 109:0 | A |
| 4996 | Túi Quà Vui Vẻ Cuồng Nhiệt | 114:0 | A |
| 5029 | Túi Quà Đá Khế Ước | 114:0 | D |
| 5119 | Bảo Rương Bí Kíp Mật Bảo Cấp 5 | 98:0, 115:0 | A |
| 5120 | Bảo Rương Bí Kíp Mật Bảo Cấp 6 | 98:0, 115:0 | A |
| 5215 | Tinh Hoa Phù Văn | 114:0, 122:0 | D |
| 5229 | Túi Quà Thời Trang Thiên Thần Ác Quỷ (30 Ngày) | 109:0 | A |
| 5230 | Thời Trang Arale (30 Ngày) | 109:0 | F |
| 5231 | Túi Quà Thời Trang Black Rock Shooter (30 Ngày) | 109:0 | A |
| 5233 | Túi Quà Lạc Lạc | 67:4 | P |
| 5236 | Túi Quà Tình Yêu Thiên Thu | 114:0 | A |
| 5237 | Túi Quà Băng Thanh Ngọc Khiết | 114:0 | A |
| 5246 | Túi Quà Kim Cương Cấp 3 | 85:0 | D |
| 5275 | Túi Quà EXP Ấn Chương Cao Cấp | 122:0 | B |
| 5281 | Túi Tự Chọn Lông Vũ Cấp 5 | 0:0 | A |
| 5282 | Túi Sách Kỹ Năng | 122:0 | A |
| 5285 | Túi Quà Thiếu Niên Hữu Vi | 0:0 | A |
| 5302 | Kết Tinh Thần Dụ Siêu Lớn | 122:0 | D |
| 5355 | Chiến Hồn Ca | 112:0 | D |
| 5877 | Ma Năng Thạch (Lớn) | 112:0, 122:0 | D |
| 5889 | Bảo Rương Tự Chọn Ma Tâm Đỏ Mới | 0:0, 122:0 | A |
| 5890 | Bảo Rương Ma Tâm Hiếm Đỏ Mới | 0:0, 122:0 | A |
| 5904 | Túi Quà Pha Lê Thần Giới Xa Hoa | 112:0, 122:0 | A |
| 5906 | Túi Quà EXP Mệnh Hồn Xa hoa | 122:0 | A |
| 5911 | Túi Quà Tuyết Rơi Lả Tả | 0:0 | D |
| 5937 | Túi Quà Đại Thánh | 67:4 | P |
| 5948 | Túi Bụi Ảo Năng | 122:0 | B |
| 5950 | Chìa Khóa Vĩnh Hằng | 114:0, 122:0 | A |
| 6028 | Túi Thú Thạch Cam Tự Chọn | 122:0 | A |
| 6030 | Túi Quà Kỳ Linh Thạch Siêu Cấp | 112:0, 122:0 | D |
| 6047 | Túi Quà Mùa Hè Tưng Bừng | 114:0 | D |
| 6126 | Túi Quà Thám Hiểm | 112:0 | B |
| 6132 | Túi Quà Moltres | 67:0 | P |
| 6168 | Túi Quà Lá Bánh Ú | 129:0 | A |
| 6186 | Túi Quà Lá Bánh Ú Cao Cấp | 129:0 | A |
| 6199 | Thời Trang Trảm Hắc Kiếm Sĩ (30 Ngày) | 109:0 | F |
| 6201 | Cute Lạc Lối (7 Ngày) | 115:0 | F |
| 6232 | Kỷ Niệm Chương Pháp Thuật | 112:0 | F |
| 6247 | Gói Quà Sinh Nhật | 112:0, 113:0, 113:0 | B |
| 6249 | Túi Phúc Niên | 112:0 | A |
| 6253 | Kỳ Linh Thạch (100) | 112:0 | D |
| 6254 | Túi Phúc Niên (10) | 112:0 | D |
| 6255 | Túi Phúc Niên (50) | 112:0 | D |
| 6274 | Sách Chuyển Dịch Ảo Ma Tháp | 35:5 | F H |
| 6278 | Túi Quà Dược Thủy Tiến Hóa Thú Cưỡi Cao Cấp | 114:0, 122:0 | A |
| 6297 | Túi Quà Võ Sĩ Shiba | 67:5, 115:0, 123:0, 123:0 | D |
| 6298 | Túi Quà Thu Thập Chữ | 13:0 | A |
| 6492 | Túi Quà Thú Cưỡi Burin | 114:0 | B |
| 6501 | Túi Quà Burin | 67:5, 98:0, 115:0 | S |
| 6503 | Túi Quà Kỷ Hợi | 111:0 | A |
| 6524 | Bảo Rương Điêu Khắc | 115:0 | B |
| 6527 | Thumbelina (7 ngày) | 115:0 | F |
| 6528 | Chân Hồn Tinh Thần (7 Ngày) | 115:0 | F |
| 6546 | Túi Quà Lễ Hội May Mắn | 119:0 | A |
| 6557 | Túi Quà Ma Quái | 120:0 | A |
| 6559 | Chìa Khóa Vinh Quang | 94:0 | A |
| 6561 | Hộp Kẹo Noel | 71:0 | A |
| 6566 | Túi Quà May Mắn Tết Nguyên Đán | 121:0 | A |
| 6575 | Bảo Rương Tự Chọn Hắc Diệu Thạch | 122:0 | A |
| 6582 | Túi Quà Jery | 123:0 | P |
| 6585 | Túi Quà May Mắn Ngọt Ngào | -1:0 | A |
| 6589 | Thư Valentine Trắng | 124:0 | A |
| 6599 | Túi Quà Quốc Tế Thiếu Nhi May Mắn | 125:0 | A |
| 6626 | Túi Quà Tình Nhân Bất Ngờ | 126:0 | A |
| 6632 | Túi Quà Tự Chọn Thú Cưỡi (Dành Riêng Sự Kiện Tiêu | 0:0 | A |
| 6633 | Túi Quà Tự Chọn Thời Trang (Dành Riêng Sự Kiện Tiê | 0:0 | A |
| 6693 | Túi Quà Long Tiểu Thanh | 123:0 | P |
| 6706 | Túi Phúc Ngày Xuân | 122:0 | A |
| 6718 | Túi Quà Tiêu Phí Kinh Hỷ | 128:0 | A |
| 6741 | Túi Quà Tết Trung Nguyên Kinh Hỷ | 117:0 | A |
| 6765 | Túi Cẩu Lương (20 Ngày) | 116:0 | A |
| 6788 | Túi Quà Năm Mới Cát Tường | 130:0, 131:0 | A |
| 6797 | Túi Quà Kim Tước | 123:0 | A |
| 6815 | Túi Quà Tiêu Phí Kình Hỉ 23 | 133:0 | A |
| 6861 | Túi Quà Huyền Minh Linh Vũ | 123:0 | A |
| 6974 | Túi Quà Bạch Hổ Cuồng Nộ | 123:0 | A |
| 6975 | Thời Trang Fantasy (30 Ngày) | 122:0 | D |
| 6999 | Gói Lễ Hội May Mắn | 132:0 | A |
| 7036 | Túi Lôi Lân Ảnh | 123:0 | D |
| 7052 | Túi Thời Thượng | 134:0, 134:0 | A |

## Verification boundary and parent handoff

Read-only PowerShell reconciliation verified **325/325 ledger IDs, recovered names, full shop/state listing sequences, and hidden flags; zero mismatches or duplicate ledger IDs**. Recomputed class counts match the summary (S25/P12/D70/B32/A112/F74), and48 are all-hidden. No gameplay PASS is asserted. No arbitrary probability, reward pool, quantity, class mapping, target ID, or database definition was created. The parent may configure the11 supplied exact mappings after its live target checks, then evaluate the other D definitions against the live target catalog. B/A require further authoritative mapping/behavior/choice evidence; P must not be reported as full reward parity merely because a pet contracts. F should be tracked under its actual subsystem, not “missing shop-bag items.”
